// rndis_usb.h — libusb I/O for the Android RNDIS function, shared by the
// data-plane and egress spike drivers. The pure framing logic is in rndis_lib;
// this is the thin USB layer around it.
//
// Provenance: clean-room from MS-RNDIS over USB. NOT GPL-derived.

#ifndef RNDIS_USB_H
#define RNDIS_USB_H

#include <libusb-1.0/libusb.h>
#include <stdint.h>

typedef struct {
    libusb_context *ctx;
    libusb_device_handle *h;
    int ctrl_if;
    int data_if;
    uint8_t ep_in;
    uint8_t ep_out;
} rndis_usb_t;

// Open and claim the first device exposing an RNDIS control interface
// (class 0xE0), keyed on interface class not VID/PID. Returns 0 on success.
int rndis_usb_open(rndis_usb_t *u);
void rndis_usb_close(rndis_usb_t *u);

// REMOTE_NDIS_INITIALIZE handshake. Returns 0 on success.
int rndis_usb_initialize(rndis_usb_t *u);

// SET OID_GEN_CURRENT_PACKET_FILTER to bring the link up. Returns 0 on success.
int rndis_usb_set_filter(rndis_usb_t *u, uint32_t filter);

// Send an Ethernet frame wrapped in a REMOTE_NDIS_PACKET_MSG on bulk OUT.
int rndis_usb_send_frame(rndis_usb_t *u, const uint8_t *frame, int frame_len);

// Read one bulk IN transfer; on a REMOTE_NDIS_PACKET_MSG, set *frame to the
// inner Ethernet frame and return its length. Returns 0 for a non-packet
// message, -1 on I/O error or malformed framing.
int rndis_usb_recv_frame(rndis_usb_t *u, uint8_t *buf, int cap,
                         const uint8_t **frame);

#endif // RNDIS_USB_H
