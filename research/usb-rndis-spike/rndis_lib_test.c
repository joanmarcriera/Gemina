// rndis_lib_test.c — unit tests for the pure RNDIS/DHCP framing logic.
//
// No hardware: these exercise rndis_lib.c byte-buffer in / out. Run with
// `make test`. Uses only synthetic, non-identifying data (a locally-administered
// test MAC, no real IPs) so it stays within the repo redaction invariant.

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

static void test_inet_csum_self_cancels(void) {
    // A correctly-checksummed header must checksum to zero over its whole span.
    uint8_t frame[DHCP_FRAME_CAP];
    rl_build_dhcp_discover(TEST_MAC, 0xDEADBEEF, frame);
    const uint8_t *ip = frame + 14;
    CHECK(rl_inet_csum(ip, 20) == 0,
          "inet_csum: built IP header does not verify to 0");
}

static void test_dhcp_discover_fields(void) {
    uint8_t f[DHCP_FRAME_CAP];
    int len = rl_build_dhcp_discover(TEST_MAC, 0x01020304, f);

    CHECK(len > 14 + 20 + 8 + 240, "discover: frame too short");
    // Ethernet: broadcast dst, our src, IPv4 ethertype.
    CHECK(f[0] == 0xff && f[5] == 0xff, "discover: dst not broadcast");
    CHECK(memcmp(f + 6, TEST_MAC, 6) == 0, "discover: src MAC mismatch");
    CHECK(f[12] == 0x08 && f[13] == 0x00, "discover: ethertype not IPv4");
    // IP: UDP protocol.
    CHECK(f[14 + 9] == 17, "discover: IP proto not UDP");
    // UDP: client(68) -> server(67).
    const uint8_t *udp = f + 14 + 20;
    CHECK(udp[0] == 0 && udp[1] == 68, "discover: UDP src not 68");
    CHECK(udp[2] == 0 && udp[3] == 67, "discover: UDP dst not 67");
    // DHCP: BOOTREQUEST, magic cookie, option 53 == DISCOVER(1).
    const uint8_t *dhcp = udp + 8;
    CHECK(dhcp[0] == 1, "discover: op not BOOTREQUEST");
    CHECK(dhcp[236] == 0x63 && dhcp[237] == 0x82 && dhcp[238] == 0x53 &&
              dhcp[239] == 0x63,
          "discover: DHCP magic cookie wrong on the wire");
    CHECK(dhcp[240] == 53 && dhcp[241] == 1 && dhcp[242] == 1,
          "discover: option 53 not DISCOVER");
}

// Turn a DISCOVER frame into a synthetic OFFER for parser testing: swap UDP
// ports to 67->68, set op=BOOTREPLY, set option 53 value to `msg_type`.
static int make_offerlike(uint8_t *f, uint8_t msg_type) {
    int len = rl_build_dhcp_discover(TEST_MAC, 0xAABBCCDD, f);
    uint8_t *udp = f + 14 + 20;
    udp[0] = 0; udp[1] = 67; // src 67
    udp[2] = 0; udp[3] = 68; // dst 68
    uint8_t *dhcp = udp + 8;
    dhcp[0] = 2;             // BOOTREPLY
    dhcp[242] = msg_type;    // option 53 value
    return len;
}

static void test_is_dhcp_offer(void) {
    uint8_t f[DHCP_FRAME_CAP];

    int len = make_offerlike(f, 2); // DHCPOFFER
    CHECK(rl_is_dhcp_offer(f, len) == 1, "is_offer: true OFFER not detected");

    len = make_offerlike(f, 5); // DHCPACK
    CHECK(rl_is_dhcp_offer(f, len) == 0, "is_offer: ACK misdetected as OFFER");

    // A plain DISCOVER (client->server) is not an offer.
    len = rl_build_dhcp_discover(TEST_MAC, 1, f);
    CHECK(rl_is_dhcp_offer(f, len) == 0, "is_offer: DISCOVER misdetected");

    // Too short.
    CHECK(rl_is_dhcp_offer(f, 40) == 0, "is_offer: short frame not rejected");
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

    // Wrong message type -> 0 (ignored, not malformed).
    rl_wr32(hdr, RNDIS_MSG_INIT_CMPLT);
    CHECK(rl_unwrap_packet(hdr, sizeof(hdr), &out) == 0,
          "unwrap: non-packet not ignored");

    // Too short -> -1.
    CHECK(rl_unwrap_packet(hdr, 10, &out) == -1,
          "unwrap: short buffer not rejected");

    // DataOffset/DataLength past the buffer -> -1.
    rl_wr32(hdr, RNDIS_MSG_PACKET);
    rl_wr32(hdr + 8, 36);
    rl_wr32(hdr + 12, 9999);
    CHECK(rl_unwrap_packet(hdr, sizeof(hdr), &out) == -1,
          "unwrap: overflowing length not rejected");
}

int main(void) {
    test_inet_csum_self_cancels();
    test_dhcp_discover_fields();
    test_is_dhcp_offer();
    test_rndis_wrap_unwrap_roundtrip();
    test_unwrap_rejects();

    if (failures == 0) {
        printf("PASS rndis_lib: all framing/DHCP unit tests passed\n");
        return 0;
    }
    printf("FAIL rndis_lib: %d check(s) failed\n", failures);
    return 1;
}
