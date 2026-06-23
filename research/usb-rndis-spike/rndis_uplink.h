// rndis_uplink.h — bring an Android RNDIS tether up to a usable IP uplink and
// send UDP datagrams over it, from userspace. Wraps rndis_usb (libusb I/O) and
// rndis_lib (framing) into one reusable uplink the spike drivers share.
//
// Provenance: clean-room from MS-RNDIS + DHCP/ARP/IPv4/UDP RFCs. NOT GPL-derived.

#ifndef RNDIS_UPLINK_H
#define RNDIS_UPLINK_H

#include <stdint.h>

#include "rndis_usb.h"

typedef struct {
    rndis_usb_t usb;
    uint8_t mac[6];       // our locally-administered source MAC (never printed)
    uint8_t client_ip[4]; // address leased from the phone's tether DHCP server
    uint8_t gw_mac[6];    // phone gateway MAC (our next hop to the internet)
    uint16_t sport;       // ephemeral UDP source port for this uplink
    int ready;
} rndis_uplink_t;

// Claim the RNDIS function, INITIALIZE, bring the link up, hold a DHCP lease, and
// ARP-resolve the phone gateway. Returns 0 on success; on failure prints a
// FAIL line explaining the stage and returns non-zero.
int rndis_uplink_bring_up(rndis_uplink_t *up);

// Send one UDP datagram (payload) to dst_ip:dport over the uplink. Returns 0 on
// success. The phone NATs it to cellular.
int rndis_uplink_send_udp(rndis_uplink_t *up, const uint8_t dst_ip[4],
                          uint16_t dport, const uint8_t *payload, int plen);

void rndis_uplink_close(rndis_uplink_t *up);

#endif // RNDIS_UPLINK_H
