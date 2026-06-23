// rndis_lib.c — implementation of the pure RNDIS/DHCP framing logic.
//
// Provenance: clean-room from MS-RNDIS + DHCP/BOOTP RFCs. NOT GPL-derived.

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

// Network byte order (big-endian) writer, for IP/UDP/DHCP headers.
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

int rl_build_dhcp_discover(const uint8_t src_mac[6], uint32_t xid,
                           uint8_t *frame) {
    memset(frame, 0, DHCP_FRAME_CAP);
    // Ethernet header.
    memset(frame + 0, 0xff, 6);    // dst broadcast
    memcpy(frame + 6, src_mac, 6); // src
    wbe16(frame + 12, 0x0800);     // EtherType IPv4
    uint8_t *ip = frame + 14;
    uint8_t *udp = ip + 20;
    uint8_t *dhcp = udp + 8;

    // DHCP/BOOTP payload (236 fixed + magic + options).
    dhcp[0] = 1;                   // op = BOOTREQUEST
    dhcp[1] = 1;                   // htype = Ethernet
    dhcp[2] = 6;                   // hlen
    rl_wr32(dhcp + 4, xid);        // xid (echoed; endianness irrelevant)
    wbe16(dhcp + 10, 0x8000);      // flags = broadcast
    memcpy(dhcp + 28, src_mac, 6); // chaddr
    uint8_t *opt = dhcp + 236;
    rl_wr32(opt, 0x63538263); // magic cookie 0x63825363 on the wire
    opt += 4;
    *opt++ = 53; *opt++ = 1; *opt++ = 1;          // DHCP Message Type = DISCOVER
    *opt++ = 55; *opt++ = 3; *opt++ = 1; *opt++ = 3; *opt++ = 6; // param req list
    *opt++ = 255;                                 // end
    int dhcp_len = (int)(opt - dhcp);

    // UDP header (src 68 -> dst 67).
    int udp_len = 8 + dhcp_len;
    wbe16(udp + 0, 68);
    wbe16(udp + 2, 67);
    wbe16(udp + 4, (uint16_t)udp_len);
    wbe16(udp + 6, 0); // checksum optional for IPv4

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
    memset(ip + 12, 0, 4);    // src = unspecified (all zero)
    memset(ip + 16, 0xff, 4); // dst = limited broadcast (all ones)
    wbe16(ip + 10, rl_inet_csum(ip, 20));

    return 14 + ip_len;
}

int rl_is_dhcp_offer(const uint8_t *f, int len) {
    if (len < 14 + 20 + 8 + 240)
        return 0;
    if (f[12] != 0x08 || f[13] != 0x00)
        return 0; // not IPv4
    const uint8_t *ip = f + 14;
    if (ip[9] != 17)
        return 0; // not UDP
    int ihl = (ip[0] & 0x0f) * 4;
    const uint8_t *udp = ip + ihl;
    if (!(udp[0] == 0 && udp[1] == 67 && udp[2] == 0 && udp[3] == 68))
        return 0; // not server(67)->client(68)
    const uint8_t *dhcp = udp + 8;
    if (dhcp[0] != 2)
        return 0; // not BOOTREPLY
    const uint8_t *opt = dhcp + 240; // 236 fixed + 4 magic
    const uint8_t *end = f + len;
    while (opt + 2 <= end && *opt != 255) {
        if (*opt == 0) {
            opt++;
            continue;
        }
        uint8_t code = opt[0], l = opt[1];
        if (code == 53 && l >= 1)
            return opt[2] == 2; // DHCPOFFER
        opt += 2 + l;
    }
    return 0;
}

int rl_wrap_packet(const uint8_t *frame, int frame_len, uint8_t *msg,
                   int msg_cap) {
    int total = RNDIS_PACKET_HDR + frame_len;
    if (frame_len < 0 || total > msg_cap)
        return -1;
    memset(msg, 0, RNDIS_PACKET_HDR);
    rl_wr32(msg + 0, RNDIS_MSG_PACKET);
    rl_wr32(msg + 4, total);                       // MessageLength
    rl_wr32(msg + 8, RNDIS_PACKET_HDR - 8);        // DataOffset (from byte 8)
    rl_wr32(msg + 12, frame_len);                  // DataLength
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
