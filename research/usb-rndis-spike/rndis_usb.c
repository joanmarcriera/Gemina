// rndis_usb.c — libusb I/O for the Android RNDIS function.
//
// Provenance: clean-room from MS-RNDIS over USB. NOT GPL-derived.

#include "rndis_usb.h"
#include "rndis_lib.h"

#include <string.h>
#include <unistd.h>

#define USB_CLASS_WIRELESS_RNDIS 0xE0
#define USB_CLASS_CDC_DATA 0x0A
#define RNDIS_SEND_ENCAPSULATED 0x00
#define RNDIS_GET_ENCAPSULATED 0x01

int rndis_usb_open(rndis_usb_t *u) {
    memset(u, 0, sizeof(*u));
    if (libusb_init(&u->ctx) != 0)
        return -1;

    libusb_device **list;
    ssize_t n = libusb_get_device_list(u->ctx, &list);
    for (ssize_t i = 0; i < n && !u->h; i++) {
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
            if (libusb_open(list[i], &u->h) == 0) {
                u->ctrl_if = c;
                u->data_if = d;
                u->ep_in = in;
                u->ep_out = out;
            }
        }
    }
    libusb_free_device_list(list, 1);

    if (!u->h) {
        libusb_exit(u->ctx);
        u->ctx = NULL;
        return -1;
    }

    libusb_set_auto_detach_kernel_driver(u->h, 1);
    if (libusb_claim_interface(u->h, u->ctrl_if) != 0 ||
        libusb_claim_interface(u->h, u->data_if) != 0) {
        rndis_usb_close(u);
        return -1;
    }
    return 0;
}

void rndis_usb_close(rndis_usb_t *u) {
    if (u->h) {
        libusb_release_interface(u->h, u->ctrl_if);
        libusb_release_interface(u->h, u->data_if);
        libusb_close(u->h);
        u->h = NULL;
    }
    if (u->ctx) {
        libusb_exit(u->ctx);
        u->ctx = NULL;
    }
}

static int rndis_command(rndis_usb_t *u, uint8_t *msg, int len, uint8_t *resp,
                         int resp_cap) {
    int sent = libusb_control_transfer(u->h, 0x21, RNDIS_SEND_ENCAPSULATED, 0,
                                       u->ctrl_if, msg, len, 1000);
    if (sent < 0)
        return sent;
    usleep(20 * 1000);
    return libusb_control_transfer(u->h, 0xA1, RNDIS_GET_ENCAPSULATED, 0,
                                   u->ctrl_if, resp, resp_cap, 1000);
}

int rndis_usb_initialize(rndis_usb_t *u) {
    uint8_t init[24], resp[1025];
    memset(init, 0, sizeof(init));
    rl_wr32(init + 0, RNDIS_MSG_INIT);
    rl_wr32(init + 4, 24);
    rl_wr32(init + 8, 1);       // RequestId
    rl_wr32(init + 12, 1);      // MajorVersion
    rl_wr32(init + 16, 0);      // MinorVersion
    rl_wr32(init + 20, 0x4000); // host MaxTransferSize
    int got = rndis_command(u, init, sizeof(init), resp, sizeof(resp));
    if (got < 16 || rl_rd32(resp) != RNDIS_MSG_INIT_CMPLT ||
        rl_rd32(resp + 12) != 0)
        return -1;
    return 0;
}

int rndis_usb_set_filter(rndis_usb_t *u, uint32_t filter) {
    uint8_t set[32], resp[1025];
    memset(set, 0, sizeof(set));
    rl_wr32(set + 0, RNDIS_MSG_SET);
    rl_wr32(set + 4, 32);                             // MessageLength
    rl_wr32(set + 8, 2);                              // RequestId
    rl_wr32(set + 12, OID_GEN_CURRENT_PACKET_FILTER); // Oid
    rl_wr32(set + 16, 4);                             // InformationBufferLength
    rl_wr32(set + 20, 20);                            // InformationBufferOffset
    rl_wr32(set + 24, 0);                             // DeviceVcHandle
    rl_wr32(set + 28, filter);
    int got = rndis_command(u, set, sizeof(set), resp, sizeof(resp));
    if (got < 16 || rl_rd32(resp) != RNDIS_MSG_SET_CMPLT ||
        rl_rd32(resp + 12) != 0)
        return -1;
    return 0;
}

int rndis_usb_send_frame(rndis_usb_t *u, const uint8_t *frame, int frame_len) {
    uint8_t msg[2048];
    int total = rl_wrap_packet(frame, frame_len, msg, sizeof(msg));
    if (total < 0)
        return -1;
    int wrote = 0;
    int rc = libusb_bulk_transfer(u->h, u->ep_out, msg, total, &wrote, 1000);
    return (rc == 0 && wrote == total) ? 0 : -1;
}

int rndis_usb_recv_frame(rndis_usb_t *u, uint8_t *buf, int cap,
                         const uint8_t **frame) {
    int got = 0;
    int rc = libusb_bulk_transfer(u->h, u->ep_in, buf, cap, &got, 1500);
    if (rc != 0)
        return -1;
    return rl_unwrap_packet(buf, got, frame);
}
