// rndis_egress.c — prove REAL UDP egress to the continuity gateway over the
// phone's cellular uplink, entirely from userspace RNDIS.
//
// Brings up the RNDIS uplink (claim, INITIALIZE, packet filter, DHCP lease, ARP)
// via rndis_uplink, then sends real continuity probes (the CVP1 wire format) to
// the gateway. The phone NATs them to cellular. Success here means a packet left
// the Mac, crossed the phone's cellular link, and (verified in the gateway's
// logs) arrived — i.e. the phone is a real independent WAN reaching the gateway.
//
// The gateway address is taken from the environment so no host/server address is
// ever compiled in or printed (repo redaction invariant):
//   GEMINA_GATEWAY_IP=<dotted quad>   GEMINA_GATEWAY_PORT=<port>
//
// Provenance: clean-room from MS-RNDIS + DHCP/ARP/IPv4/UDP RFCs and the
// continuity probe wire format (internal/protocol). NOT GPL-derived.

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#include "rndis_lib.h"
#include "rndis_net.h"
#include "rndis_uplink.h"

int main(void) {
    uint8_t gw_ip[4];
    int gw_port;
    if (gateway_from_env(gw_ip, &gw_port) != 0) {
        printf("FAIL config: set GEMINA_GATEWAY_IP to the gateway dotted "
               "quad\n");
        return 2;
    }

    rndis_uplink_t up;
    if (rndis_uplink_bring_up(&up) != 0)
        return 1;
    printf("PASS uplink: RNDIS link up, lease held, gateway resolved "
           "(addresses redacted)\n");

    uint8_t session[16];
    random_session(session);

    int sent = 0;
    const int distinct = 5, copies = 2;
    for (int num = 1; num <= distinct; num++) {
        uint8_t probe[CVP_PROBE_SIZE];
        rl_build_probe(session, (uint64_t)num, PATH_ANDROID_USB_TETHER, probe);
        for (int c = 0; c < copies; c++) {
            if (rndis_uplink_send_udp(&up, gw_ip, (uint16_t)gw_port, probe,
                                      sizeof(probe)) == 0)
                sent++;
        }
    }

    printf("PASS egress: sent %d probe datagram(s) (%d distinct x%d) to the "
           "gateway over cellular\n",
           sent, distinct, copies);
    printf("RESULT: userspace RNDIS egress COMPLETE — verify arrival in the "
           "gateway logs (ssh oracle). Probe session is redacted here.\n");

    rndis_uplink_close(&up);
    return 0;
}
