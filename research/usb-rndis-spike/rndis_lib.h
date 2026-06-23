// rndis_lib.h — pure, hardware-independent RNDIS/DHCP framing logic.
//
// Split out of rndis_dataplane.c so it can be unit-tested without a phone. All
// USB I/O stays in rndis_dataplane.c; everything here is byte-buffer in / out.
//
// Provenance: clean-room from the public MS-RNDIS message layout and the
// DHCP/BOOTP RFCs (2131/951). NOT derived from any GPL source.

#ifndef RNDIS_LIB_H
#define RNDIS_LIB_H

#include <stddef.h>
#include <stdint.h>

// RNDIS message types (MS-RNDIS).
#define RNDIS_MSG_PACKET 0x00000001u
#define RNDIS_MSG_INIT 0x00000002u
#define RNDIS_MSG_INIT_CMPLT 0x80000002u
#define RNDIS_MSG_SET 0x00000005u
#define RNDIS_MSG_SET_CMPLT 0x80000005u

// OID + NDIS packet-filter flags.
#define OID_GEN_CURRENT_PACKET_FILTER 0x0001010Eu
#define NDIS_PACKET_TYPE_DIRECTED 0x00000001u
#define NDIS_PACKET_TYPE_MULTICAST 0x00000002u
#define NDIS_PACKET_TYPE_BROADCAST 0x00000010u

// REMOTE_NDIS_PACKET_MSG header size (single packet, no OOB/per-packet-info).
#define RNDIS_PACKET_HDR 44

// Minimum scratch size build_dhcp_discover needs.
#define DHCP_FRAME_CAP 600

uint32_t rl_rd32(const uint8_t *p);
void rl_wr32(uint8_t *p, uint32_t v);

// Internet 16-bit one's-complement checksum.
uint16_t rl_inet_csum(const uint8_t *data, size_t len);

// Build a DHCP DISCOVER Ethernet frame into `frame` (must be >= DHCP_FRAME_CAP).
// Returns the frame length.
int rl_build_dhcp_discover(const uint8_t src_mac[6], uint32_t xid,
                           uint8_t *frame);

// Report whether an Ethernet frame is a DHCP OFFER (server 67 -> client 68,
// BOOTREPLY, option 53 == 2).
int rl_is_dhcp_offer(const uint8_t *frame, int len);

// Wrap an Ethernet frame in a REMOTE_NDIS_PACKET_MSG. Returns total message
// length, or -1 if it would not fit in msg_cap.
int rl_wrap_packet(const uint8_t *frame, int frame_len, uint8_t *msg,
                   int msg_cap);

// Extract the Ethernet frame from a received REMOTE_NDIS_PACKET_MSG. Sets *frame
// and returns frame length; returns 0 if not a packet message, -1 if malformed.
int rl_unwrap_packet(const uint8_t *buf, int got, const uint8_t **frame);

#endif // RNDIS_LIB_H
