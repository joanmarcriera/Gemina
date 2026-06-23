// rndis_uplink.c — RNDIS tether bring-up + UDP send (see rndis_uplink.h).
//
// Provenance: clean-room from MS-RNDIS + DHCP/ARP/IPv4/UDP RFCs. NOT GPL-derived.

#include "rndis_uplink.h"
#include "rndis_lib.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

// Send `tx` then drain bulk IN looking for a reply the predicate accepts.
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

int rndis_uplink_bring_up(rndis_uplink_t *up) {
    memset(up, 0, sizeof(*up));

    if (rndis_usb_open(&up->usb) != 0) {
        printf("FAIL open: no RNDIS function claimed. Enable USB tethering.\n");
        return 1;
    }
    if (rndis_usb_initialize(&up->usb) != 0) {
        printf("FAIL init: INITIALIZE failed\n");
        rndis_usb_close(&up->usb);
        return 1;
    }
    if (rndis_usb_set_filter(&up->usb, NDIS_PACKET_TYPE_DIRECTED |
                                           NDIS_PACKET_TYPE_MULTICAST |
                                           NDIS_PACKET_TYPE_BROADCAST) != 0) {
        printf("FAIL filter: set packet filter failed\n");
        rndis_usb_close(&up->usb);
        return 1;
    }

    srand((unsigned)time(NULL) ^ (unsigned)getpid());
    for (int i = 0; i < 6; i++)
        up->mac[i] = (uint8_t)rand();
    up->mac[0] = (up->mac[0] & 0xfe) | 0x02; // locally administered, unicast
    uint32_t xid = (uint32_t)rand();
    up->sport = (uint16_t)(40000 + (rand() % 20000));

    uint8_t frame[FRAME_CAP];

    rl_lease_t offer;
    int flen = rl_build_dhcp_discover(up->mac, xid, frame);
    if (exchange(&up->usb, frame, flen, 5, match_offer, &offer) != 1) {
        printf("FAIL dhcp-offer: no OFFER (is USB tethering on?)\n");
        rndis_usb_close(&up->usb);
        return 1;
    }

    rl_lease_t ack;
    flen = rl_build_dhcp_request(up->mac, xid, &offer, frame);
    if (exchange(&up->usb, frame, flen, 5, match_ack, &ack) != 1) {
        printf("FAIL dhcp-ack: no ACK to our REQUEST\n");
        rndis_usb_close(&up->usb);
        return 1;
    }
    if (!ack.has_router && offer.has_router) {
        ack.has_router = 1;
        memcpy(ack.router, offer.router, 4);
    }
    if (!ack.has_router) {
        printf("FAIL lease: no router/gateway option in the lease\n");
        rndis_usb_close(&up->usb);
        return 1;
    }
    memcpy(up->client_ip, ack.client_ip, 4);

    struct arp_ctx arp = {.target_ip = ack.router};
    flen = rl_build_arp_request(up->mac, up->client_ip, ack.router, frame);
    if (exchange(&up->usb, frame, flen, 6, match_arp, &arp) != 1) {
        printf("FAIL arp: gateway did not answer ARP\n");
        rndis_usb_close(&up->usb);
        return 1;
    }
    memcpy(up->gw_mac, arp.mac, 6);

    up->ready = 1;
    return 0;
}

int rndis_uplink_send_udp(rndis_uplink_t *up, const uint8_t dst_ip[4],
                          uint16_t dport, const uint8_t *payload, int plen) {
    if (!up->ready)
        return -1;
    uint8_t frame[FRAME_CAP];
    int flen = rl_build_udp_frame(up->mac, up->gw_mac, up->client_ip, dst_ip,
                                  up->sport, dport, payload, plen, frame);
    if (flen < 0)
        return -1;
    return rndis_usb_send_frame(&up->usb, frame, flen);
}

void rndis_uplink_close(rndis_uplink_t *up) {
    rndis_usb_close(&up->usb);
    up->ready = 0;
}
