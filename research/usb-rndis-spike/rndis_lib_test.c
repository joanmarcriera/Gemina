// rndis_lib_test.c — unit tests for the pure RNDIS/DHCP/ARP/UDP framing logic.
//
// No hardware: these exercise rndis_lib.c byte-buffer in / out. Run with
// `make test`. Uses only synthetic, non-identifying data (locally-administered
// test MACs, TEST-NET-style addresses) so it stays within the repo redaction
// invariant.

#include "rndis_lib.h"

#include <stdio.h>
#include <string.h>

static int failures = 0;
#define CHECK(cond, msg)                                                        \
    do {                                                                        \
        if (!(cond)) {                                                          \
            printf("FAIL %s\n", msg);                                           \
            failures++;                                                         \
        }                                                                       \
    } while (0)

static const uint8_t TEST_MAC[6] = {0x02, 0x11, 0x22, 0x33, 0x44, 0x55};
static const uint8_t GW_MAC[6] = {0x02, 0xaa, 0xbb, 0xcc, 0xdd, 0xee};
// Documentation/test addresses only (RFC 5737 TEST-NET-1 / RFC 1918).
static const uint8_t SRC_IP[4] = {192, 168, 42, 100};
static const uint8_t GW_IP[4] = {192, 168, 42, 1};
static const uint8_t DST_IP[4] = {198, 51, 100, 7};

static void wbe16_test(uint8_t *p, uint16_t v) {
    p[0] = (v >> 8) & 0xff;
    p[1] = v & 0xff;
}

static void test_inet_csum_self_cancels(void) {
    uint8_t frame[FRAME_CAP];
    rl_build_dhcp_discover(TEST_MAC, 0xDEADBEEF, frame);
    const uint8_t *ip = frame + 14;
    CHECK(rl_inet_csum(ip, 20) == 0,
          "inet_csum: built IP header does not verify to 0");
}

static void test_dhcp_discover_fields(void) {
    uint8_t f[FRAME_CAP];
    int len = rl_build_dhcp_discover(TEST_MAC, 0x01020304, f);

    CHECK(len > 14 + 20 + 8 + 240, "discover: frame too short");
    CHECK(f[0] == 0xff && f[5] == 0xff, "discover: dst not broadcast");
    CHECK(memcmp(f + 6, TEST_MAC, 6) == 0, "discover: src MAC mismatch");
    CHECK(f[12] == 0x08 && f[13] == 0x00, "discover: ethertype not IPv4");
    CHECK(f[14 + 9] == 17, "discover: IP proto not UDP");
    const uint8_t *udp = f + 14 + 20;
    CHECK(udp[0] == 0 && udp[1] == 68, "discover: UDP src not 68");
    CHECK(udp[2] == 0 && udp[3] == 67, "discover: UDP dst not 67");
    const uint8_t *dhcp = udp + 8;
    CHECK(dhcp[0] == 1, "discover: op not BOOTREQUEST");
    CHECK(dhcp[236] == 0x63 && dhcp[237] == 0x82 && dhcp[238] == 0x53 &&
              dhcp[239] == 0x63,
          "discover: DHCP magic cookie wrong on the wire");
    CHECK(dhcp[240] == 53 && dhcp[241] == 1 && dhcp[242] == 1,
          "discover: option 53 not DISCOVER");
}

// Build a synthetic DHCP server->client reply (OFFER or ACK) for parser testing.
static int make_reply(uint8_t *f, uint8_t msg_type, const uint8_t yiaddr[4]) {
    int len = rl_build_dhcp_discover(TEST_MAC, 0xAABBCCDD, f);
    uint8_t *udp = f + 14 + 20;
    udp[0] = 0; udp[1] = 67; // src 67
    udp[2] = 0; udp[3] = 68; // dst 68
    uint8_t *dhcp = udp + 8;
    dhcp[0] = 2;                  // BOOTREPLY
    memcpy(dhcp + 16, yiaddr, 4); // yiaddr
    // Rewrite options: 53=msg_type, 54=server_id(GW_IP), 3=router(GW_IP), end.
    uint8_t *opt = dhcp + 240;
    *opt++ = 53; *opt++ = 1; *opt++ = msg_type;
    *opt++ = 54; *opt++ = 4; memcpy(opt, GW_IP, 4); opt += 4;
    *opt++ = 3;  *opt++ = 4; memcpy(opt, GW_IP, 4); opt += 4;
    *opt++ = 255;
    int newlen = (int)(opt - f);
    return newlen > len ? newlen : len;
}

static void test_parse_dhcp_reply(void) {
    uint8_t f[FRAME_CAP];
    rl_lease_t lease;

    int len = make_reply(f, DHCP_OFFER, SRC_IP);
    CHECK(rl_parse_dhcp_reply(f, len, DHCP_OFFER, &lease) == 1,
          "parse: OFFER not matched");
    CHECK(memcmp(lease.client_ip, SRC_IP, 4) == 0, "parse: yiaddr wrong");
    CHECK(lease.has_server_id && memcmp(lease.server_id, GW_IP, 4) == 0,
          "parse: server id wrong");
    CHECK(lease.has_router && memcmp(lease.router, GW_IP, 4) == 0,
          "parse: router wrong");

    // Wanting ACK but given OFFER must not match.
    CHECK(rl_parse_dhcp_reply(f, len, DHCP_ACK, &lease) == 0,
          "parse: OFFER misread as ACK");

    len = make_reply(f, DHCP_ACK, SRC_IP);
    CHECK(rl_parse_dhcp_reply(f, len, DHCP_ACK, &lease) == 1,
          "parse: ACK not matched");

    // is_dhcp_offer convenience wrapper.
    len = make_reply(f, DHCP_OFFER, SRC_IP);
    CHECK(rl_is_dhcp_offer(f, len) == 1, "is_offer: OFFER not detected");
    CHECK(rl_is_dhcp_offer(f, 40) == 0, "is_offer: short frame not rejected");
}

static void test_build_dhcp_request(void) {
    rl_lease_t lease;
    memset(&lease, 0, sizeof(lease));
    memcpy(lease.client_ip, SRC_IP, 4);
    memcpy(lease.server_id, GW_IP, 4);
    lease.has_server_id = 1;

    uint8_t f[FRAME_CAP];
    int len = rl_build_dhcp_request(TEST_MAC, 0x11223344, &lease, f);
    const uint8_t *dhcp = f + 14 + 20 + 8;
    CHECK(len > 240, "request: too short");
    CHECK(dhcp[240] == 53 && dhcp[242] == DHCP_REQUEST,
          "request: option 53 not REQUEST");
    // Option 50 (requested IP) must carry the offered client_ip somewhere.
    int found_req_ip = 0, found_server = 0;
    const uint8_t *opt = dhcp + 240;
    const uint8_t *end = f + len;
    while (opt + 2 <= end && *opt != 255) {
        uint8_t code = opt[0], l = opt[1];
        if (code == 50 && l == 4 && memcmp(opt + 2, SRC_IP, 4) == 0)
            found_req_ip = 1;
        if (code == 54 && l == 4 && memcmp(opt + 2, GW_IP, 4) == 0)
            found_server = 1;
        opt += 2 + l;
    }
    CHECK(found_req_ip, "request: option 50 requested-IP missing/wrong");
    CHECK(found_server, "request: option 54 server-id missing/wrong");
}

static void test_arp_build_and_parse(void) {
    uint8_t req[64];
    int len = rl_build_arp_request(TEST_MAC, SRC_IP, GW_IP, req);
    CHECK(len == 42, "arp: request not 42 bytes");
    CHECK(req[12] == 0x08 && req[13] == 0x06, "arp: ethertype not ARP");
    CHECK(req[14 + 6] == 0x00 && req[14 + 7] == 0x01, "arp: oper not request");
    CHECK(memcmp(req + 14 + 24, GW_IP, 4) == 0, "arp: target IP wrong");

    // Craft a reply: oper=2, sha=GW_MAC, spa=GW_IP.
    uint8_t rep[42];
    memset(rep, 0, sizeof(rep));
    rep[12] = 0x08; rep[13] = 0x06;
    uint8_t *arp = rep + 14;
    wbe16_test(arp, 0x0001);
    arp[6] = 0x00; arp[7] = 0x02; // reply
    memcpy(arp + 8, GW_MAC, 6);   // sha
    memcpy(arp + 14, GW_IP, 4);   // spa
    uint8_t out[6];
    CHECK(rl_parse_arp_reply(rep, sizeof(rep), GW_IP, out) == 1,
          "arp: reply for target not matched");
    CHECK(memcmp(out, GW_MAC, 6) == 0, "arp: resolved MAC wrong");

    // A reply about a different IP must not match.
    CHECK(rl_parse_arp_reply(rep, sizeof(rep), SRC_IP, out) == 0,
          "arp: reply for wrong IP matched");
}

static void test_udp_frame_checksums(void) {
    const uint8_t payload[30] = {'C', 'V', 'P', '1', 1, 2, 0};
    uint8_t f[FRAME_CAP];
    int len = rl_build_udp_frame(TEST_MAC, GW_MAC, SRC_IP, DST_IP, 40000, 51820,
                                 payload, sizeof(payload), f);
    CHECK(len == 14 + 20 + 8 + 30, "udp: frame length wrong");
    CHECK(memcmp(f + 0, GW_MAC, 6) == 0, "udp: dst MAC wrong");
    CHECK(memcmp(f + 6, TEST_MAC, 6) == 0, "udp: src MAC wrong");

    const uint8_t *ip = f + 14;
    CHECK(rl_inet_csum(ip, 20) == 0, "udp: IP header checksum invalid");
    CHECK(ip[9] == 17, "udp: proto not UDP");
    CHECK(memcmp(ip + 12, SRC_IP, 4) == 0, "udp: src IP wrong");
    CHECK(memcmp(ip + 16, DST_IP, 4) == 0, "udp: dst IP wrong");

    const uint8_t *udp = ip + 20;
    CHECK(udp[2] == (51820 >> 8) && udp[3] == (51820 & 0xff),
          "udp: dst port wrong");

    // Verify the UDP checksum over pseudo-header + segment == 0.
    int udp_len = 8 + 30;
    uint8_t pseudo[12 + 8 + 30];
    memcpy(pseudo + 0, SRC_IP, 4);
    memcpy(pseudo + 4, DST_IP, 4);
    pseudo[8] = 0; pseudo[9] = 17;
    pseudo[10] = (udp_len >> 8); pseudo[11] = (udp_len & 0xff);
    memcpy(pseudo + 12, udp, udp_len);
    CHECK(rl_inet_csum(pseudo, 12 + udp_len) == 0,
          "udp: UDP checksum does not verify");
    CHECK(memcmp(f + 14 + 20 + 8, payload, sizeof(payload)) == 0,
          "udp: payload corrupted");
}

static void test_build_probe(void) {
    uint8_t session[16];
    for (int i = 0; i < 16; i++)
        session[i] = (uint8_t)(i + 1);
    uint8_t out[CVP_PROBE_SIZE];
    rl_build_probe(session, 0x0102030405060708ull, 2, out);

    CHECK(memcmp(out, "CVP1", 4) == 0, "probe: magic wrong");
    CHECK(out[4] == 1, "probe: version not 1");
    CHECK(out[5] == 2, "probe: path tag wrong");
    CHECK(memcmp(out + 6, session, 16) == 0, "probe: session mismatch");
    // Big-endian number.
    CHECK(out[22] == 0x01 && out[29] == 0x08, "probe: number not big-endian");
}

static void test_rndis_wrap_unwrap_roundtrip(void) {
    uint8_t frame[64];
    for (int i = 0; i < 64; i++)
        frame[i] = (uint8_t)(i * 7 + 1);

    uint8_t msg[256];
    int total = rl_wrap_packet(frame, 64, msg, sizeof(msg));
    CHECK(total == RNDIS_PACKET_HDR + 64, "wrap: wrong total length");
    CHECK(rl_rd32(msg) == RNDIS_MSG_PACKET, "wrap: type not PACKET_MSG");
    CHECK(rl_rd32(msg + 4) == (uint32_t)total, "wrap: MessageLength wrong");
    CHECK(rl_rd32(msg + 8) == RNDIS_PACKET_HDR - 8, "wrap: DataOffset wrong");
    CHECK(rl_rd32(msg + 12) == 64, "wrap: DataLength wrong");

    const uint8_t *out = NULL;
    int got = rl_unwrap_packet(msg, total, &out);
    CHECK(got == 64, "unwrap: wrong frame length");
    CHECK(out && memcmp(out, frame, 64) == 0, "unwrap: frame bytes differ");
}

static void test_unwrap_rejects(void) {
    const uint8_t *out = NULL;
    uint8_t hdr[RNDIS_PACKET_HDR];
    memset(hdr, 0, sizeof(hdr));

    rl_wr32(hdr, RNDIS_MSG_INIT_CMPLT);
    CHECK(rl_unwrap_packet(hdr, sizeof(hdr), &out) == 0,
          "unwrap: non-packet not ignored");
    CHECK(rl_unwrap_packet(hdr, 10, &out) == -1,
          "unwrap: short buffer not rejected");
    rl_wr32(hdr, RNDIS_MSG_PACKET);
    rl_wr32(hdr + 8, 36);
    rl_wr32(hdr + 12, 9999);
    CHECK(rl_unwrap_packet(hdr, sizeof(hdr), &out) == -1,
          "unwrap: overflowing length not rejected");
}

int main(void) {
    test_inet_csum_self_cancels();
    test_dhcp_discover_fields();
    test_parse_dhcp_reply();
    test_build_dhcp_request();
    test_arp_build_and_parse();
    test_udp_frame_checksums();
    test_build_probe();
    test_rndis_wrap_unwrap_roundtrip();
    test_unwrap_rejects();

    if (failures == 0) {
        printf("PASS rndis_lib: all framing/DHCP/ARP/UDP unit tests passed\n");
        return 0;
    }
    printf("FAIL rndis_lib: %d check(s) failed\n", failures);
    return 1;
}
