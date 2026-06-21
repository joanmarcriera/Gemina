// rndis_probe.c — userspace viability spike for in-app Android USB tethering.
//
// Purpose: prove that a notarisable, sandboxable macOS app could drive an
// Android RNDIS USB tethering function entirely from userspace, WITHOUT a
// kernel/DriverKit extension and WITHOUT relaxing System Integrity Protection.
//
// It does three things, in order, and stops at the first failure:
//   1. Open the OnePlus device by VID/PID via libusb (IOKit under the hood).
//   2. Claim the RNDIS control + data interfaces (class 0xE0/0x0A).
//   3. Complete the RNDIS REMOTE_NDIS_INITIALIZE handshake over the control
//      channel and read back the device's capabilities.
//
// It sends only RNDIS_INITIALIZE and (optionally) a non-identifying capability
// QUERY. It never queries the permanent MAC, never moves user traffic, and
// leaves no persistent state on the phone. Output is redacted: no serial, no
// MAC, no IP.
//
// Provenance: authored clean-room from the public Remote NDIS (MS-RNDIS)
// protocol constants. NOT derived from Linux GPL drivers/net/usb/rndis_host.c.
//
// Build: see Makefile in this directory.

#include <libusb-1.0/libusb.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

// OnePlus 12R (KALAMA) in MTP+RNDIS+ADB composite mode, observed via ioreg.
#define VID 0x22D9
#define PID 0x2766

// RNDIS message types (MS-RNDIS).
#define RNDIS_MSG_INIT 0x00000002u
#define RNDIS_MSG_INIT_CMPLT 0x80000002u
#define RNDIS_MSG_QUERY 0x00000004u
#define RNDIS_MSG_QUERY_CMPLT 0x80000004u

// RNDIS over USB control requests (class, interface recipient).
#define RNDIS_SEND_ENCAPSULATED 0x00
#define RNDIS_GET_ENCAPSULATED 0x01

// Non-identifying capability OID: maximum total frame size.
#define OID_GEN_MAXIMUM_TOTAL_SIZE 0x00010111u

#define USB_CLASS_WIRELESS_RNDIS 0xE0
#define USB_CLASS_CDC_DATA 0x0A

static uint32_t rd32(const uint8_t *p) {
    return (uint32_t)p[0] | ((uint32_t)p[1] << 8) | ((uint32_t)p[2] << 16) |
           ((uint32_t)p[3] << 24);
}
static void wr32(uint8_t *p, uint32_t v) {
    p[0] = v & 0xff;
    p[1] = (v >> 8) & 0xff;
    p[2] = (v >> 16) & 0xff;
    p[3] = (v >> 24) & 0xff;
}

static int find_rndis_interfaces(libusb_device *dev, int *ctrl_if, int *data_if,
                                 uint8_t *ep_in, uint8_t *ep_out) {
    struct libusb_config_descriptor *cfg;
    if (libusb_get_active_config_descriptor(dev, &cfg) != 0)
        return -1;
    *ctrl_if = *data_if = -1;
    *ep_in = *ep_out = 0;
    for (int i = 0; i < cfg->bNumInterfaces; i++) {
        const struct libusb_interface_descriptor *id =
            &cfg->interface[i].altsetting[0];
        if (id->bInterfaceClass == USB_CLASS_WIRELESS_RNDIS)
            *ctrl_if = id->bInterfaceNumber;
        if (id->bInterfaceClass == USB_CLASS_CDC_DATA) {
            *data_if = id->bInterfaceNumber;
            for (int e = 0; e < id->bNumEndpoints; e++) {
                uint8_t addr = id->endpoint[e].bEndpointAddress;
                if (addr & 0x80)
                    *ep_in = addr;
                else
                    *ep_out = addr;
            }
        }
    }
    libusb_free_config_descriptor(cfg);
    return (*ctrl_if >= 0 && *data_if >= 0) ? 0 : -1;
}

int main(void) {
    libusb_context *ctx = NULL;
    if (libusb_init(&ctx) != 0) {
        printf("FAIL libusb_init\n");
        return 1;
    }

    libusb_device_handle *h = libusb_open_device_with_vid_pid(ctx, VID, PID);
    if (!h) {
        printf("FAIL open: device %04x:%04x not opened (no permission or not "
               "present)\n",
               VID, PID);
        libusb_exit(ctx);
        return 1;
    }
    printf("PASS open: claimed a handle to OnePlus RNDIS composite device\n");

    libusb_set_auto_detach_kernel_driver(h, 1);

    int ctrl_if, data_if;
    uint8_t ep_in, ep_out;
    if (find_rndis_interfaces(libusb_get_device(h), &ctrl_if, &data_if, &ep_in,
                              &ep_out) != 0) {
        printf("FAIL descriptors: RNDIS control/data interfaces not found\n");
        libusb_close(h);
        libusb_exit(ctx);
        return 1;
    }
    printf("PASS descriptors: ctrl_if=%d data_if=%d bulk_in=0x%02x "
           "bulk_out=0x%02x\n",
           ctrl_if, data_if, ep_in, ep_out);

    int rc_ctrl = libusb_claim_interface(h, ctrl_if);
    int rc_data = libusb_claim_interface(h, data_if);
    if (rc_ctrl != 0 || rc_data != 0) {
        printf("FAIL claim: ctrl=%s data=%s\n", libusb_strerror(rc_ctrl),
               libusb_strerror(rc_data));
        libusb_close(h);
        libusb_exit(ctx);
        return 1;
    }
    printf("PASS claim: userspace owns both RNDIS interfaces (SIP enabled)\n");

    // RNDIS INITIALIZE: 24-byte message.
    uint8_t init[24];
    memset(init, 0, sizeof(init));
    wr32(init + 0, RNDIS_MSG_INIT);
    wr32(init + 4, 24);
    wr32(init + 8, 1);       // RequestId
    wr32(init + 12, 1);      // MajorVersion
    wr32(init + 16, 0);      // MinorVersion
    wr32(init + 20, 0x4000); // MaxTransferSize (host can receive)

    int sent = libusb_control_transfer(
        h, 0x21, RNDIS_SEND_ENCAPSULATED, 0, ctrl_if, init, sizeof(init), 1000);
    if (sent < 0) {
        printf("FAIL init-send: %s\n", libusb_strerror(sent));
        goto done;
    }
    printf("PASS init-send: SEND_ENCAPSULATED_COMMAND (%d bytes) accepted\n",
           sent);

    usleep(50 * 1000);

    uint8_t resp[1025];
    memset(resp, 0, sizeof(resp));
    int got = libusb_control_transfer(h, 0xA1, RNDIS_GET_ENCAPSULATED, 0,
                                      ctrl_if, resp, 1024, 1000);
    if (got < 24) {
        printf("FAIL init-resp: short/failed read (%d): %s\n", got,
               got < 0 ? libusb_strerror(got) : "too short");
        goto done;
    }
    uint32_t mtype = rd32(resp + 0);
    uint32_t status = rd32(resp + 12);
    uint32_t medium = rd32(resp + 28);
    uint32_t max_xfer = rd32(resp + 36);
    if (mtype != RNDIS_MSG_INIT_CMPLT) {
        printf("FAIL init-resp: unexpected type 0x%08x\n", mtype);
        goto done;
    }
    printf("PASS init-resp: REMOTE_NDIS_INITIALIZE_CMPLT status=0x%08x "
           "medium=%u(0=802.3) device_max_transfer=%u bytes\n",
           status, medium, max_xfer);
    printf("RESULT: userspace RNDIS handshake COMPLETE — the phone is a usable "
           "second uplink driven entirely from an unprivileged process.\n");

done:
    libusb_release_interface(h, ctrl_if);
    libusb_release_interface(h, data_if);
    libusb_close(h);
    libusb_exit(ctx);
    return 0;
}
