// rndis_lib.h — pure, hardware-independent RNDIS/DHCP/ARP/UDP framing logic.
//
// Split out of the spike binaries so it can be unit-tested without a phone. All
// USB I/O stays in the *_dataplane.c / *_egress.c drivers; everything here is
// byte-buffer in / out.
//
// Provenance: clean-room from the public MS-RNDIS message layout and the
// DHCP/BOOTP (RFC 2131/951), ARP (RFC 826) and IPv4/UDP (RFC 791/768) headers.
// NOT derived from any GPL source.

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

// Scratch size callers should provide for any frame builder here.
#define FRAME_CAP 600

// DHCP message types (option 53).
#define DHCP_DISCOVER 1
#define DHCP_OFFER 2
#define DHCP_REQUEST 3
#define DHCP_ACK 5

uint32_t rl_rd32(const uint8_t *p);
void rl_wr32(uint8_t *p, uint32_t v);

// Internet 16-bit one's-complement checksum.
uint16_t rl_inet_csum(const uint8_t *data, size_t len);

// A lease learned from a DHCP OFFER/ACK. All addresses are raw 4-byte IPv4; the
// caller must never print or store them (repo redaction invariant).
typedef struct {
    uint8_t client_ip[4]; // yiaddr
    uint8_t server_id[4]; // option 54
    uint8_t router[4];    // option 3 (default gateway / next hop)
    int has_server_id;
    int has_router;
} rl_lease_t;

// --- DHCP ---------------------------------------------------------------

// Build a broadcast DHCP DISCOVER Ethernet frame. Returns frame length.
int rl_build_dhcp_discover(const uint8_t src_mac[6], uint32_t xid,
                           uint8_t *frame);

// Build a broadcast DHCP REQUEST selecting the offered lease (option 50 requested
// IP + option 54 server id). Returns frame length.
int rl_build_dhcp_request(const uint8_t src_mac[6], uint32_t xid,
                          const rl_lease_t *lease, uint8_t *frame);

// Parse a DHCP server->client reply. If it is a BOOTREPLY of message type
// want_type (DHCP_OFFER / DHCP_ACK), fills *lease (yiaddr + options) and returns
// 1; otherwise returns 0.
int rl_parse_dhcp_reply(const uint8_t *frame, int len, uint8_t want_type,
                        rl_lease_t *lease);

// --- ARP ----------------------------------------------------------------

// Build a broadcast ARP request "who has target_ip, tell src_ip". 42 bytes.
int rl_build_arp_request(const uint8_t src_mac[6], const uint8_t src_ip[4],
                         const uint8_t target_ip[4], uint8_t *frame);

// If frame is an ARP reply whose sender protocol address is target_ip, copy the
// sender hardware address into out_mac and return 1; else 0.
int rl_parse_arp_reply(const uint8_t *frame, int len, const uint8_t target_ip[4],
                       uint8_t out_mac[6]);

// --- UDP ----------------------------------------------------------------

// Build an Ethernet/IPv4/UDP frame carrying payload, with correct IP and UDP
// checksums. Returns frame length, or -1 if it would not fit in FRAME_CAP.
int rl_build_udp_frame(const uint8_t src_mac[6], const uint8_t dst_mac[6],
                       const uint8_t src_ip[4], const uint8_t dst_ip[4],
                       uint16_t src_port, uint16_t dst_port,
                       const uint8_t *payload, int payload_len, uint8_t *frame);

// --- RNDIS data framing -------------------------------------------------

// Report whether an Ethernet frame is a DHCP OFFER (kept for the data-plane
// probe; equivalent to rl_parse_dhcp_reply with want_type DHCP_OFFER).
int rl_is_dhcp_offer(const uint8_t *frame, int len);

// Wrap an Ethernet frame in a REMOTE_NDIS_PACKET_MSG. Returns total message
// length, or -1 if it would not fit in msg_cap.
int rl_wrap_packet(const uint8_t *frame, int frame_len, uint8_t *msg,
                   int msg_cap);

// Extract the Ethernet frame from a received REMOTE_NDIS_PACKET_MSG. Sets *frame
// and returns frame length; returns 0 if not a packet message, -1 if malformed.
int rl_unwrap_packet(const uint8_t *buf, int got, const uint8_t **frame);

// --- continuity probe ---------------------------------------------------

// 30-byte Stage-1 probe wire size (must match internal/protocol ProbeWireSize).
#define CVP_PROBE_SIZE 30

// Build the 30-byte continuity probe datagram (magic CVP1, version 1, path,
// 16-byte session, 8-byte big-endian number) into out (>= CVP_PROBE_SIZE).
void rl_build_probe(const uint8_t session[16], uint64_t number, uint8_t path,
                    uint8_t *out);

#endif // RNDIS_LIB_H
