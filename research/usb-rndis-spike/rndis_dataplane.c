// rndis_dataplane.c — userspace RNDIS *data plane* viability spike.
//
// Builds on rndis_probe.c (claim + REMOTE_NDIS_INITIALIZE). This step proves the
// data plane: bring the link up with a packet-filter OID, then send a real
// Ethernet frame (a DHCP DISCOVER) wrapped in a REMOTE_NDIS_PACKET_MSG on the
// bulk OUT endpoint and read the phone tether's DHCP OFFER back on bulk IN.
//
// A DHCP round-trip is the ideal first packet: it exercises both directions of
// frame transport and needs no prior knowledge of the phone's address — the
// phone's own tethering DHCP server answers. Success means the userspace path
// can move L2 frames, which is everything NEPacketTunnelProvider needs below it.
//
// The pure framing logic lives in rndis_lib.{c,h} and is unit-tested in
// rndis_lib_test.c; this file is the USB I/O around it.
//
// Output is redacted: no MAC, no serial, no IP is ever printed (repo invariant).
//
// Provenance: clean-room from the public MS-RNDIS message layout and the
// DHCP/BOOTP RFCs (2131/951). NOT derived from Linux GPL
// drivers/net/usb/rndis_host.c or any GPL DHCP client.
//
// Build: see Makefile (`make rndis_dataplane` / `make run-dataplane`).

#include <libusb-1.0/libusb.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

#include "rndis_lib.h"

#define USB_CLASS_WIRELESS_RNDIS 0xE0
#define USB_CLASS_CDC_DATA 0x0A

// RNDIS over USB control requests (class, interface recipient).
#define RNDIS_SEND_ENCAPSULATED 0x00
#define RNDIS_GET_ENCAPSULATED 0x01

// Find the first device exposing an RNDIS control interface (class 0xE0). Keyed
// on interface class, not VID/PID, so it survives the phone re-enumerating with
// a different product id when adb is in the composite (seen: 0x2766 vs 0x276A).
static libusb_device_handle *open_rndis_device(libusb_context *ctx, int *ctrl_if,
                                               int *data_if, uint8_t *ep_in,
                                               uint8_t *ep_out) {
    libusb_device **list;
    ssize_t n = libusb_get_device_list(ctx, &list);
    libusb_device_handle *result = NULL;
    for (ssize_t i = 0; i < n && !result; i++) {
        struct libusb_config_descriptor *cfg;
        if (libusb_get_active_config_descriptor(list[i], &cfg) != 0)
            continue;
        int c = -1, d = -1;
        uint8_t in = 0, out = 0;
        for (int j = 0; j < cfg->bNumInterfaces; j++) {
            const struct libusb_interface_descriptor *id =
                &cfg->interface[j].altsetting[0];
            if (id->bInterfaceClass == USB_CLASS_WIRELESS_RNDIS)
                c = id->bInterfaceNumber;
            if (id->bInterfaceClass == USB_CLASS_CDC_DATA) {
                d = id->bInterfaceNumber;
                for (int e = 0; e < id->bNumEndpoints; e++) {
                    uint8_t a = id->endpoint[e].bEndpointAddress;
                    if (a & 0x80)
                        in = a;
                    else
                        out = a;
                }
            }
        }
        libusb_free_config_descriptor(cfg);
        if (c >= 0 && d >= 0 && in && out) {
            if (libusb_open(list[i], &result) == 0) {
                *ctrl_if = c;
                *data_if = d;
                *ep_in = in;
                *ep_out = out;
            }
        }
    }
    libusb_free_device_list(list, 1);
    return result;
}

static int rndis_command(libusb_device_handle *h, int ctrl_if, uint8_t *msg,
                         int len, uint8_t *resp, int resp_cap) {
    int sent = libusb_control_transfer(h, 0x21, RNDIS_SEND_ENCAPSULATED, 0,
                                       ctrl_if, msg, len, 1000);
    if (sent < 0)
        return sent;
    usleep(20 * 1000);
    return libusb_control_transfer(h, 0xA1, RNDIS_GET_ENCAPSULATED, 0, ctrl_if,
                                   resp, resp_cap, 1000);
}

static int rndis_initialize(libusb_device_handle *h, int ctrl_if) {
    uint8_t init[24], resp[1025];
    memset(init, 0, sizeof(init));
    rl_wr32(init + 0, RNDIS_MSG_INIT);
    rl_wr32(init + 4, 24);
    rl_wr32(init + 8, 1);       // RequestId
    rl_wr32(init + 12, 1);      // MajorVersion
    rl_wr32(init + 16, 0);      // MinorVersion
    rl_wr32(init + 20, 0x4000); // host MaxTransferSize
    int got = rndis_command(h, ctrl_if, init, sizeof(init), resp, sizeof(resp));
    if (got < 16 || rl_rd32(resp) != RNDIS_MSG_INIT_CMPLT ||
        rl_rd32(resp + 12) != 0)
        return -1;
    return 0;
}

static int rndis_set_packet_filter(libusb_device_handle *h, int ctrl_if,
                                   uint32_t filter) {
    // RNDIS_SET: 28-byte header + 4-byte info buffer.
    uint8_t set[32], resp[1025];
    memset(set, 0, sizeof(set));
    rl_wr32(set + 0, RNDIS_MSG_SET);
    rl_wr32(set + 4, 32);                             // MessageLength
    rl_wr32(set + 8, 2);                              // RequestId
    rl_wr32(set + 12, OID_GEN_CURRENT_PACKET_FILTER); // Oid
    rl_wr32(set + 16, 4);                             // InformationBufferLength
    rl_wr32(set + 20, 20);                            // InformationBufferOffset
    rl_wr32(set + 24, 0);                             // DeviceVcHandle
    rl_wr32(set + 28, filter);                        // the filter bitmask
    int got = rndis_command(h, ctrl_if, set, sizeof(set), resp, sizeof(resp));
    if (got < 16 || rl_rd32(resp) != RNDIS_MSG_SET_CMPLT ||
        rl_rd32(resp + 12) != 0)
        return -1;
    return 0;
}

static int rndis_send_frame(libusb_device_handle *h, uint8_t ep_out,
                            const uint8_t *frame, int frame_len) {
    uint8_t msg[2048];
    int total = rl_wrap_packet(frame, frame_len, msg, sizeof(msg));
    if (total < 0)
        return -1;
    int wrote = 0;
    int rc = libusb_bulk_transfer(h, ep_out, msg, total, &wrote, 1000);
    return (rc == 0 && wrote == total) ? 0 : -1;
}

static int rndis_recv_frame(libusb_device_handle *h, uint8_t ep_in, uint8_t *buf,
                            int cap, const uint8_t **frame) {
    int got = 0;
    int rc = libusb_bulk_transfer(h, ep_in, buf, cap, &got, 1500);
    if (rc != 0)
        return -1;
    return rl_unwrap_packet(buf, got, frame);
}

int main(void) {
    libusb_context *ctx = NULL;
    if (libusb_init(&ctx) != 0) {
        printf("FAIL libusb_init\n");
        return 1;
    }

    int ctrl_if, data_if;
    uint8_t ep_in, ep_out;
    libusb_device_handle *h =
        open_rndis_device(ctx, &ctrl_if, &data_if, &ep_in, &ep_out);
    if (!h) {
        printf("FAIL open: no RNDIS (class 0xE0) function found. Enable USB "
               "tethering on the phone.\n");
        libusb_exit(ctx);
        return 1;
    }
    printf("PASS open: claimed RNDIS device (ctrl_if=%d data_if=%d "
           "bulk_in=0x%02x bulk_out=0x%02x)\n",
           ctrl_if, data_if, ep_in, ep_out);

    libusb_set_auto_detach_kernel_driver(h, 1);
    if (libusb_claim_interface(h, ctrl_if) != 0 ||
        libusb_claim_interface(h, data_if) != 0) {
        printf("FAIL claim: could not claim RNDIS interfaces\n");
        libusb_close(h);
        libusb_exit(ctx);
        return 1;
    }
    printf("PASS claim: userspace owns both RNDIS interfaces\n");

    if (rndis_initialize(h, ctrl_if) != 0) {
        printf("FAIL init: REMOTE_NDIS_INITIALIZE did not complete\n");
        goto done;
    }
    printf("PASS init: REMOTE_NDIS_INITIALIZE complete\n");

    if (rndis_set_packet_filter(h, ctrl_if,
                                NDIS_PACKET_TYPE_DIRECTED |
                                    NDIS_PACKET_TYPE_MULTICAST |
                                    NDIS_PACKET_TYPE_BROADCAST) != 0) {
        printf("FAIL filter: SET OID_GEN_CURRENT_PACKET_FILTER failed\n");
        goto done;
    }
    printf("PASS filter: link up (packet filter set, directed+mcast+bcast)\n");

    // Locally-administered random source MAC (never printed).
    uint8_t mac[6];
    srand((unsigned)time(NULL));
    for (int i = 0; i < 6; i++)
        mac[i] = (uint8_t)rand();
    mac[0] = (mac[0] & 0xfe) | 0x02; // locally administered, unicast
    uint32_t xid = (uint32_t)rand();

    uint8_t frame[DHCP_FRAME_CAP];
    int flen = rl_build_dhcp_discover(mac, xid, frame);

    int offered = 0;
    for (int attempt = 0; attempt < 4 && !offered; attempt++) {
        if (rndis_send_frame(h, ep_out, frame, flen) != 0) {
            printf("FAIL tx: bulk OUT of REMOTE_NDIS_PACKET_MSG failed\n");
            goto done;
        }
        // Drain bulk IN for a short window looking for our OFFER.
        for (int r = 0; r < 12 && !offered; r++) {
            uint8_t in[2048];
            const uint8_t *rxf = NULL;
            int rl = rndis_recv_frame(h, ep_in, in, sizeof(in), &rxf);
            if (rl > 0 && rxf && rl_is_dhcp_offer(rxf, rl))
                offered = 1;
        }
    }

    if (!offered) {
        printf("FAIL dhcp: no DHCP OFFER seen (tx worked; phone may not be "
               "running its tether DHCP server — is USB tethering on?)\n");
        goto done;
    }
    printf("PASS dhcp: REMOTE_NDIS_PACKET_MSG round-trip OK — phone's tether "
           "DHCP server offered a lease (address redacted)\n");
    printf("RESULT: userspace RNDIS DATA PLANE works — L2 frames move both ways "
           "over the phone tether from an unprivileged process.\n");

done:
    libusb_release_interface(h, ctrl_if);
    libusb_release_interface(h, data_if);
    libusb_close(h);
    libusb_exit(ctx);
    return 0;
}
