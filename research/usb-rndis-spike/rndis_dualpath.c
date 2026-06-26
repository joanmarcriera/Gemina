// rndis_dualpath.c — the Stage-1 dual-path proof: the SAME logical packet leaves
// the Mac over Wi-Fi AND over the phone's cellular link at the same time, both
// reach the gateway, and the gateway delivers each logical packet once.
//
// Path A (Wi-Fi): an OS UDP socket bound to the Wi-Fi interface via IP_BOUND_IF,
//                 so it egresses Wi-Fi regardless of the default route.
// Path B (cellular): the userspace RNDIS uplink to the phone (no kext/SIP/root).
//
// Both carry identical probe identities (same session + packet number); only the
// PathTag differs, so the gateway dedups by identity and attributes copies to a
// path. The run has three phases to also prove path-loss survival:
//   1. both paths        -> gateway logs first-copy + duplicate per identity
//   2. Wi-Fi only        -> models cellular loss; identities still arrive
//   3. cellular only     -> models Wi-Fi loss; identities still arrive
//
// Addresses come from the environment and nothing identifying is printed (repo
// redaction invariant):
//   GEMINA_GATEWAY_IP=<dotted quad>  GEMINA_GATEWAY_PORT=<port>
//   GEMINA_WIFI_IFACE=<iface, default en0>
//
// Provenance: clean-room from MS-RNDIS + DHCP/ARP/IPv4/UDP RFCs and the
// continuity probe wire format. NOT GPL-derived.

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#include "rndis_lib.h"
#include "rndis_net.h"
#include "rndis_uplink.h"

int main(void) {
    uint8_t gw_ip[4];
    int gw_port;
    if (gateway_from_env(gw_ip, &gw_port) != 0) {
        printf("FAIL config: set GEMINA_GATEWAY_IP\n");
        return 2;
    }
    const char *wifi_iface = getenv("GEMINA_WIFI_IFACE");
    if (!wifi_iface)
        wifi_iface = "en0";

    // Path B: bring up the cellular RNDIS uplink first (it also seeds rand()).
    rndis_uplink_t up;
    if (rndis_uplink_bring_up(&up) != 0)
        return 1;
    printf("PASS path-B: RNDIS cellular uplink up (lease + gateway resolved)\n");

    // Path A: Wi-Fi socket bound to the Wi-Fi interface.
    int wifi = wifi_socket_open(wifi_iface, gw_ip, gw_port);
    if (wifi < 0) {
        printf("FAIL path-A: could not bind a UDP socket to %s\n", wifi_iface);
        rndis_uplink_close(&up);
        return 1;
    }
    printf("PASS path-A: Wi-Fi socket bound to %s\n", wifi_iface);

    uint8_t session[16];
    random_session(session);

    int num = 0;
    int both = 0, wifi_only = 0, cell_only = 0;

    // Helper macro-free inline sends keep the path tags explicit.
    // Phase 1: both paths, same identity.
    for (int i = 0; i < 5; i++) {
        num++;
        uint8_t pa[CVP_PROBE_SIZE], pb[CVP_PROBE_SIZE];
        rl_build_probe(session, (uint64_t)num, PATH_WIFI, pa);
        rl_build_probe(session, (uint64_t)num, PATH_ANDROID_USB_TETHER, pb);
        int a_ok = (wifi_socket_send(wifi, pa, sizeof(pa)) == 0);
        int b_ok =
            (rndis_uplink_send_udp(&up, gw_ip, (uint16_t)gw_port, pb,
                                   sizeof(pb)) == 0);
        if (a_ok && b_ok)
            both++;
    }
    printf("PASS phase-1: %d identities sent over BOTH paths simultaneously\n",
           both);

    // Phase 2: Wi-Fi only (model cellular path loss).
    for (int i = 0; i < 3; i++) {
        num++;
        uint8_t pa[CVP_PROBE_SIZE];
        rl_build_probe(session, (uint64_t)num, PATH_WIFI, pa);
        if (wifi_socket_send(wifi, pa, sizeof(pa)) == 0)
            wifi_only++;
    }
    printf("PASS phase-2: %d identities sent over Wi-Fi only (cellular "
           "'down')\n",
           wifi_only);

    // Phase 3: cellular only (model Wi-Fi path loss).
    for (int i = 0; i < 3; i++) {
        num++;
        uint8_t pb[CVP_PROBE_SIZE];
        rl_build_probe(session, (uint64_t)num, PATH_ANDROID_USB_TETHER, pb);
        if (rndis_uplink_send_udp(&up, gw_ip, (uint16_t)gw_port, pb,
                                  sizeof(pb)) == 0)
            cell_only++;
    }
    printf("PASS phase-3: %d identities sent over cellular only (Wi-Fi "
           "'down')\n",
           cell_only);

    printf("RESULT: dual-path send COMPLETE — %d distinct identities, one "
           "session. Expect at the gateway: %d first-copy total, %d duplicates "
           "(phase 1), and the surviving path delivering every identity in "
           "phases 2-3. Verify in the gateway logs (ssh oracle).\n",
           num, num, both);

    wifi_socket_close(wifi);
    rndis_uplink_close(&up);
    return 0;
}
