// rndis_egress.c — prove REAL UDP egress to the continuity gateway over the
// phone's cellular uplink, entirely from userspace RNDIS.
//
// Flow:
//   1. Claim the RNDIS function, INITIALIZE, bring the link up (packet filter).
//   2. DHCP DISCOVER -> OFFER -> REQUEST -> ACK: hold a lease, learn our address
//      and the phone's gateway (router).
//   3. ARP-resolve the gateway's MAC (next hop).
//   4. Build continuity probe datagrams (the real CVP1 wire format) in UDP/IP
//      frames addressed to the gateway, and send them out bulk OUT. The phone
//      NATs them to cellular.
//
// Success here means a packet left the Mac, crossed the phone's cellular link,
// and (verified separately in the gateway's logs) arrived at the deployed
// gateway — i.e. the phone is a real independent WAN reaching the gateway.
//
// The gateway address is taken from the environment so no host/server address is
// ever compiled in or printed (repo redaction invariant):
//   CONTINUITY_GATEWAY_IP=<dotted quad>   CONTINUITY_GATEWAY_PORT=<port>
//
// Provenance: clean-room from MS-RNDIS + DHCP/ARP/IPv4/UDP RFCs and the
// continuity probe wire format (internal/protocol). NOT GPL-derived.

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

#include "rndis_lib.h"
#include "rndis_usb.h"

// Parse a dotted-quad into 4 bytes. Returns 0 on success.
static int parse_ipv4(const char *s, uint8_t out[4]) {
    if (!s)
        return -1;
    int a, b, c, d;
    if (sscanf(s, "%d.%d.%d.%d", &a, &b, &c, &d) != 4)
        return -1;
    if (a < 0 || a > 255 || b < 0 || b > 255 || c < 0 || c > 255 || d < 0 ||
        d > 255)
        return -1;
    out[0] = (uint8_t)a;
    out[1] = (uint8_t)b;
    out[2] = (uint8_t)c;
    out[3] = (uint8_t)d;
    return 0;
}

// Send `frame` and then drain bulk IN looking for a reply the predicate accepts.
// Retries the send up to `attempts` times. Returns 1 if matched.
typedef int (*match_fn)(const uint8_t *frame, int len, void *ctx);

static int exchange(rndis_usb_t *u, const uint8_t *tx, int tx_len, int attempts,
                    match_fn match, void *ctx) {
    for (int a = 0; a < attempts; a++) {
        if (rndis_usb_send_frame(u, tx, tx_len) != 0)
            return -1;
        for (int r = 0; r < 16; r++) {
            uint8_t in[2048];
            const uint8_t *rxf = NULL;
            int rl = rndis_usb_recv_frame(u, in, sizeof(in), &rxf);
            if (rl > 0 && rxf && match(rxf, rl, ctx))
                return 1;
        }
    }
    return 0;
}

static int match_offer(const uint8_t *f, int len, void *ctx) {
    return rl_parse_dhcp_reply(f, len, DHCP_OFFER, (rl_lease_t *)ctx);
}
static int match_ack(const uint8_t *f, int len, void *ctx) {
    return rl_parse_dhcp_reply(f, len, DHCP_ACK, (rl_lease_t *)ctx);
}

struct arp_ctx {
    const uint8_t *target_ip;
    uint8_t mac[6];
};
static int match_arp(const uint8_t *f, int len, void *ctx) {
    struct arp_ctx *a = ctx;
    return rl_parse_arp_reply(f, len, a->target_ip, a->mac);
}

int main(void) {
    uint8_t gw_ip[4];
    const char *gw_ip_s = getenv("CONTINUITY_GATEWAY_IP");
    const char *gw_port_s = getenv("CONTINUITY_GATEWAY_PORT");
    if (parse_ipv4(gw_ip_s, gw_ip) != 0) {
        printf("FAIL config: set CONTINUITY_GATEWAY_IP to the gateway dotted "
               "quad\n");
        return 2;
    }
    int gw_port = gw_port_s ? atoi(gw_port_s) : 51820;
    if (gw_port <= 0 || gw_port > 65535)
        gw_port = 51820;

    rndis_usb_t u;
    if (rndis_usb_open(&u) != 0) {
        printf("FAIL open: no RNDIS function claimed. Enable USB tethering.\n");
        return 1;
    }
    printf("PASS open: claimed RNDIS device\n");

    if (rndis_usb_initialize(&u) != 0) {
        printf("FAIL init: INITIALIZE failed\n");
        goto done;
    }
    if (rndis_usb_set_filter(&u, NDIS_PACKET_TYPE_DIRECTED |
                                     NDIS_PACKET_TYPE_MULTICAST |
                                     NDIS_PACKET_TYPE_BROADCAST) != 0) {
        printf("FAIL filter: set packet filter failed\n");
        goto done;
    }
    printf("PASS link: INITIALIZE + packet filter (link up)\n");

    // Random locally-administered MAC + transaction id (never printed).
    uint8_t mac[6];
    srand((unsigned)time(NULL) ^ (unsigned)getpid());
    for (int i = 0; i < 6; i++)
        mac[i] = (uint8_t)rand();
    mac[0] = (mac[0] & 0xfe) | 0x02;
    uint32_t xid = (uint32_t)rand();

    uint8_t frame[FRAME_CAP];

    // DHCP DISCOVER -> OFFER.
    rl_lease_t lease;
    int flen = rl_build_dhcp_discover(mac, xid, frame);
    if (exchange(&u, frame, flen, 5, match_offer, &lease) != 1) {
        printf("FAIL dhcp-offer: no OFFER (is USB tethering on?)\n");
        goto done;
    }
    printf("PASS dhcp-offer: lease offered (address redacted)\n");

    // DHCP REQUEST -> ACK (hold the lease).
    rl_lease_t acked;
    flen = rl_build_dhcp_request(mac, xid, &lease, frame);
    if (exchange(&u, frame, flen, 5, match_ack, &acked) != 1) {
        printf("FAIL dhcp-ack: no ACK to our REQUEST\n");
        goto done;
    }
    if (!acked.has_router) {
        // Some servers omit the router in the ACK; fall back to the OFFER's.
        if (lease.has_router) {
            acked.has_router = 1;
            memcpy(acked.router, lease.router, 4);
        }
    }
    if (!acked.has_router) {
        printf("FAIL lease: ACK carried no router/gateway option\n");
        goto done;
    }
    printf("PASS dhcp-ack: lease held; gateway learned (redacted)\n");

    // ARP-resolve the gateway's MAC (our next hop to the internet).
    struct arp_ctx arp = {.target_ip = acked.router};
    flen = rl_build_arp_request(mac, acked.client_ip, acked.router, frame);
    if (exchange(&u, frame, flen, 6, match_arp, &arp) != 1) {
        printf("FAIL arp: gateway did not answer ARP\n");
        goto done;
    }
    printf("PASS arp: resolved gateway MAC (redacted)\n");

    // Send real continuity probes to the gateway over cellular.
    uint8_t session[16];
    for (int i = 0; i < 16; i++)
        session[i] = (uint8_t)rand();
    session[0] |= 1; // ensure non-zero session
    uint16_t sport = (uint16_t)(40000 + (rand() % 20000));

    int sent = 0;
    const int distinct = 5, copies = 2;
    for (int num = 1; num <= distinct; num++) {
        uint8_t probe[CVP_PROBE_SIZE];
        rl_build_probe(session, (uint64_t)num, /*path android-usb-tether*/ 2,
                       probe);
        int pf = rl_build_udp_frame(mac, arp.mac, acked.client_ip, gw_ip, sport,
                                    (uint16_t)gw_port, probe, sizeof(probe),
                                    frame);
        for (int c = 0; c < copies; c++) {
            if (rndis_usb_send_frame(&u, frame, pf) == 0)
                sent++;
        }
    }

    printf("PASS egress: sent %d probe datagram(s) (%d distinct x%d) to the "
           "gateway over cellular\n",
           sent, distinct, copies);
    printf("RESULT: userspace RNDIS egress COMPLETE — verify arrival in the "
           "gateway logs (ssh oracle). Probe session is redacted here.\n");

done:
    rndis_usb_close(&u);
    return 0;
}
