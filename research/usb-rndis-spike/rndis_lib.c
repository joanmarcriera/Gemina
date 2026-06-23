// rndis_lib.c — implementation of the pure RNDIS/DHCP/ARP/UDP framing logic.
//
// Provenance: clean-room from MS-RNDIS + DHCP/BOOTP/ARP/IPv4/UDP RFCs. NOT
// GPL-derived.

#include "rndis_lib.h"

#include <string.h>

uint32_t rl_rd32(const uint8_t *p) {
    return (uint32_t)p[0] | ((uint32_t)p[1] << 8) | ((uint32_t)p[2] << 16) |
           ((uint32_t)p[3] << 24);
}

void rl_wr32(uint8_t *p, uint32_t v) {
    p[0] = v & 0xff;
    p[1] = (v >> 8) & 0xff;
    p[2] = (v >> 16) & 0xff;
    p[3] = (v >> 24) & 0xff;
}

// Network byte order (big-endian) 16-bit writer, for IP/UDP/DHCP/ARP headers.
static void wbe16(uint8_t *p, uint16_t v) {
    p[0] = (v >> 8) & 0xff;
    p[1] = v & 0xff;
}

uint16_t rl_inet_csum(const uint8_t *data, size_t len) {
    uint32_t sum = 0;
    for (size_t i = 0; i + 1 < len; i += 2)
        sum += ((uint32_t)data[i] << 8) | data[i + 1];
    if (len & 1)
        sum += (uint32_t)data[len - 1] << 8;
    while (sum >> 16)
        sum = (sum & 0xffff) + (sum >> 16);
    return (uint16_t)~sum;
}

// Assemble Ethernet + IPv4 + UDP around a payload. Computes the IP header
// checksum and the UDP checksum (with pseudo-header). Returns frame length or -1.
static int udp_ip_eth(uint8_t *frame, const uint8_t dst_mac[6],
                      const uint8_t src_mac[6], const uint8_t src_ip[4],
                      const uint8_t dst_ip[4], uint16_t sport, uint16_t dport,
                      const uint8_t *payload, int plen) {
    int total = 14 + 20 + 8 + plen;
    if (plen < 0 || total > FRAME_CAP)
        return -1;
    memset(frame, 0, total);

    // Ethernet.
    memcpy(frame + 0, dst_mac, 6);
    memcpy(frame + 6, src_mac, 6);
    wbe16(frame + 12, 0x0800); // IPv4

    uint8_t *ip = frame + 14;
    uint8_t *udp = ip + 20;
    uint8_t *data = udp + 8;
    memcpy(data, payload, plen);

    // UDP header.
    int udp_len = 8 + plen;
    wbe16(udp + 0, sport);
    wbe16(udp + 2, dport);
    wbe16(udp + 4, (uint16_t)udp_len);
    wbe16(udp + 6, 0); // checksum placeholder

    // IPv4 header.
    int ip_len = 20 + udp_len;
    ip[0] = 0x45;
    ip[1] = 0;
    wbe16(ip + 2, (uint16_t)ip_len);
    wbe16(ip + 4, 0);
    wbe16(ip + 6, 0);
    ip[8] = 64; // TTL
    ip[9] = 17; // UDP
    wbe16(ip + 10, 0);
    memcpy(ip + 12, src_ip, 4);
    memcpy(ip + 16, dst_ip, 4);
    wbe16(ip + 10, rl_inet_csum(ip, 20));

    // UDP checksum over pseudo-header + UDP segment.
    uint8_t pseudo[12 + 8 + 512];
    if (udp_len <= (int)sizeof(pseudo) - 12) {
        memcpy(pseudo + 0, src_ip, 4);
        memcpy(pseudo + 4, dst_ip, 4);
        pseudo[8] = 0;
        pseudo[9] = 17;
        wbe16(pseudo + 10, (uint16_t)udp_len);
        memcpy(pseudo + 12, udp, udp_len);
        uint16_t csum = rl_inet_csum(pseudo, 12 + udp_len);
        if (csum == 0)
            csum = 0xffff; // 0 means "no checksum"; send all-ones instead
        wbe16(udp + 6, csum);
    }

    return total;
}

// Fill the DHCP/BOOTP body (op..options) into body. Returns body length.
// req_ip / server_id may be NULL (DISCOVER); when set, adds options 50 / 54.
static int dhcp_body(uint8_t *body, const uint8_t mac[6], uint32_t xid,
                     uint8_t msgtype, const uint8_t *req_ip,
                     const uint8_t *server_id) {
    memset(body, 0, 240);
    body[0] = 1;                 // op = BOOTREQUEST
    body[1] = 1;                 // htype = Ethernet
    body[2] = 6;                 // hlen
    rl_wr32(body + 4, xid);      // xid (echoed)
    wbe16(body + 10, 0x8000);    // flags = broadcast
    memcpy(body + 28, mac, 6);   // chaddr
    uint8_t *opt = body + 236;
    rl_wr32(opt, 0x63538263); // magic cookie 0x63825363 on the wire
    opt += 4;
    *opt++ = 53; *opt++ = 1; *opt++ = msgtype; // DHCP Message Type
    if (req_ip) {
        *opt++ = 50; *opt++ = 4;
        memcpy(opt, req_ip, 4);
        opt += 4;
    }
    if (server_id) {
        *opt++ = 54; *opt++ = 4;
        memcpy(opt, server_id, 4);
        opt += 4;
    }
    *opt++ = 55; *opt++ = 3; *opt++ = 1; *opt++ = 3; *opt++ = 6; // param req list
    *opt++ = 255;                                                 // end
    return (int)(opt - body);
}

int rl_build_dhcp_discover(const uint8_t src_mac[6], uint32_t xid,
                           uint8_t *frame) {
    uint8_t body[300];
    int blen = dhcp_body(body, src_mac, xid, DHCP_DISCOVER, NULL, NULL);
    const uint8_t bcast_mac[6] = {0xff, 0xff, 0xff, 0xff, 0xff, 0xff};
    const uint8_t any_ip[4] = {0, 0, 0, 0};
    const uint8_t bcast_ip[4] = {0xff, 0xff, 0xff, 0xff};
    return udp_ip_eth(frame, bcast_mac, src_mac, any_ip, bcast_ip, 68, 67, body,
                      blen);
}

int rl_build_dhcp_request(const uint8_t src_mac[6], uint32_t xid,
                          const rl_lease_t *lease, uint8_t *frame) {
    uint8_t body[300];
    const uint8_t *sid = lease->has_server_id ? lease->server_id : NULL;
    int blen =
        dhcp_body(body, src_mac, xid, DHCP_REQUEST, lease->client_ip, sid);
    const uint8_t bcast_mac[6] = {0xff, 0xff, 0xff, 0xff, 0xff, 0xff};
    const uint8_t any_ip[4] = {0, 0, 0, 0};
    const uint8_t bcast_ip[4] = {0xff, 0xff, 0xff, 0xff};
    return udp_ip_eth(frame, bcast_mac, src_mac, any_ip, bcast_ip, 68, 67, body,
                      blen);
}

// Locate the DHCP body in an Ethernet/IP/UDP frame and validate it is a
// server(67)->client(68) BOOTREPLY. Returns a pointer to the DHCP body and its
// length via *dhcp_len, or NULL.
static const uint8_t *dhcp_reply_body(const uint8_t *f, int len, int *dhcp_len) {
    if (len < 14 + 20 + 8 + 240)
        return NULL;
    if (f[12] != 0x08 || f[13] != 0x00)
        return NULL; // not IPv4
    const uint8_t *ip = f + 14;
    if (ip[9] != 17)
        return NULL; // not UDP
    int ihl = (ip[0] & 0x0f) * 4;
    if (ihl < 20)
        return NULL;
    const uint8_t *udp = ip + ihl;
    if (udp + 8 > f + len)
        return NULL;
    if (!(udp[0] == 0 && udp[1] == 67 && udp[2] == 0 && udp[3] == 68))
        return NULL; // not server->client
    const uint8_t *dhcp = udp + 8;
    if (dhcp + 240 > f + len)
        return NULL;
    if (dhcp[0] != 2)
        return NULL; // not BOOTREPLY
    *dhcp_len = (int)(f + len - dhcp);
    return dhcp;
}

int rl_parse_dhcp_reply(const uint8_t *f, int len, uint8_t want_type,
                        rl_lease_t *lease) {
    int dhcp_len = 0;
    const uint8_t *dhcp = dhcp_reply_body(f, len, &dhcp_len);
    if (!dhcp)
        return 0;

    memset(lease, 0, sizeof(*lease));
    memcpy(lease->client_ip, dhcp + 16, 4); // yiaddr

    int matched_type = 0;
    const uint8_t *opt = dhcp + 240; // 236 fixed + 4 magic
    const uint8_t *end = dhcp + dhcp_len;
    while (opt + 2 <= end && *opt != 255) {
        if (*opt == 0) {
            opt++;
            continue;
        }
        uint8_t code = opt[0], l = opt[1];
        if (opt + 2 + l > end)
            break;
        const uint8_t *val = opt + 2;
        if (code == 53 && l >= 1)
            matched_type = (val[0] == want_type);
        else if (code == 54 && l == 4) {
            memcpy(lease->server_id, val, 4);
            lease->has_server_id = 1;
        } else if (code == 3 && l >= 4) {
            memcpy(lease->router, val, 4);
            lease->has_router = 1;
        }
        opt += 2 + l;
    }
    return matched_type;
}

int rl_is_dhcp_offer(const uint8_t *f, int len) {
    rl_lease_t scratch;
    return rl_parse_dhcp_reply(f, len, DHCP_OFFER, &scratch);
}

int rl_build_arp_request(const uint8_t src_mac[6], const uint8_t src_ip[4],
                         const uint8_t target_ip[4], uint8_t *frame) {
    memset(frame, 0, 42);
    memset(frame + 0, 0xff, 6);    // dst broadcast
    memcpy(frame + 6, src_mac, 6); // src
    wbe16(frame + 12, 0x0806);     // ARP
    uint8_t *arp = frame + 14;
    wbe16(arp + 0, 0x0001);        // htype Ethernet
    wbe16(arp + 2, 0x0800);        // ptype IPv4
    arp[4] = 6;                    // hlen
    arp[5] = 4;                    // plen
    wbe16(arp + 6, 0x0001);        // oper request
    memcpy(arp + 8, src_mac, 6);   // sha
    memcpy(arp + 14, src_ip, 4);   // spa
    // tha left zero
    memcpy(arp + 24, target_ip, 4); // tpa
    return 42;
}

int rl_parse_arp_reply(const uint8_t *f, int len, const uint8_t target_ip[4],
                       uint8_t out_mac[6]) {
    if (len < 42)
        return 0;
    if (f[12] != 0x08 || f[13] != 0x06)
        return 0; // not ARP
    const uint8_t *arp = f + 14;
    if (!(arp[6] == 0x00 && arp[7] == 0x02))
        return 0; // not a reply
    if (memcmp(arp + 14, target_ip, 4) != 0)
        return 0; // sender protocol addr != who we asked about
    memcpy(out_mac, arp + 8, 6); // sender hardware addr
    return 1;
}

int rl_build_udp_frame(const uint8_t src_mac[6], const uint8_t dst_mac[6],
                       const uint8_t src_ip[4], const uint8_t dst_ip[4],
                       uint16_t src_port, uint16_t dst_port,
                       const uint8_t *payload, int payload_len, uint8_t *frame) {
    return udp_ip_eth(frame, dst_mac, src_mac, src_ip, dst_ip, src_port,
                      dst_port, payload, payload_len);
}

int rl_wrap_packet(const uint8_t *frame, int frame_len, uint8_t *msg,
                   int msg_cap) {
    int total = RNDIS_PACKET_HDR + frame_len;
    if (frame_len < 0 || total > msg_cap)
        return -1;
    memset(msg, 0, RNDIS_PACKET_HDR);
    rl_wr32(msg + 0, RNDIS_MSG_PACKET);
    rl_wr32(msg + 4, total);                // MessageLength
    rl_wr32(msg + 8, RNDIS_PACKET_HDR - 8); // DataOffset (from byte 8)
    rl_wr32(msg + 12, frame_len);           // DataLength
    memcpy(msg + RNDIS_PACKET_HDR, frame, frame_len);
    return total;
}

int rl_unwrap_packet(const uint8_t *buf, int got, const uint8_t **frame) {
    if (got < RNDIS_PACKET_HDR)
        return -1;
    if (rl_rd32(buf) != RNDIS_MSG_PACKET)
        return 0; // some other RNDIS message
    uint32_t data_off = rl_rd32(buf + 8);
    uint32_t data_len = rl_rd32(buf + 12);
    if ((uint64_t)8 + data_off + data_len > (uint64_t)got)
        return -1;
    *frame = buf + 8 + data_off;
    return (int)data_len;
}

void rl_build_probe(const uint8_t session[16], uint64_t number, uint8_t path,
                    uint8_t *out) {
    out[0] = 'C';
    out[1] = 'V';
    out[2] = 'P';
    out[3] = '1';
    out[4] = 1;    // version
    out[5] = path; // PathTag
    memcpy(out + 6, session, 16);
    // 8-byte big-endian packet number at offset 22.
    for (int i = 0; i < 8; i++)
        out[22 + i] = (uint8_t)(number >> (56 - 8 * i));
}
