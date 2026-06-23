// rndis_dataplane.c — userspace RNDIS *data plane* viability spike.
//
// Proves the data plane: claim the Android RNDIS function, INITIALIZE, bring the
// link up with a packet-filter OID, then send a DHCP DISCOVER framed in a
// REMOTE_NDIS_PACKET_MSG on bulk OUT and read the phone tether's DHCP OFFER back
// on bulk IN. A DHCP round-trip is the ideal first packet: it exercises both
// directions and needs no prior knowledge of the phone's address.
//
// USB I/O is in rndis_usb.{c,h}; pure framing in rndis_lib.{c,h} (unit-tested in
// rndis_lib_test.c). Output is redacted: no MAC, no serial, no IP.
//
// Provenance: clean-room from MS-RNDIS + DHCP/BOOTP RFCs. NOT GPL-derived.

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <time.h>

#include "rndis_lib.h"
#include "rndis_usb.h"

int main(void) {
    rndis_usb_t u;
    if (rndis_usb_open(&u) != 0) {
        printf("FAIL open: no RNDIS (class 0xE0) function claimed. Enable USB "
               "tethering on the phone.\n");
        return 1;
    }
    printf("PASS open: claimed RNDIS device (ctrl_if=%d data_if=%d "
           "bulk_in=0x%02x bulk_out=0x%02x)\n",
           u.ctrl_if, u.data_if, u.ep_in, u.ep_out);

    if (rndis_usb_initialize(&u) != 0) {
        printf("FAIL init: REMOTE_NDIS_INITIALIZE did not complete\n");
        goto done;
    }
    printf("PASS init: REMOTE_NDIS_INITIALIZE complete\n");

    if (rndis_usb_set_filter(&u, NDIS_PACKET_TYPE_DIRECTED |
                                     NDIS_PACKET_TYPE_MULTICAST |
                                     NDIS_PACKET_TYPE_BROADCAST) != 0) {
        printf("FAIL filter: SET OID_GEN_CURRENT_PACKET_FILTER failed\n");
        goto done;
    }
    printf("PASS filter: link up (packet filter set, directed+mcast+bcast)\n");

    uint8_t mac[6];
    srand((unsigned)time(NULL));
    for (int i = 0; i < 6; i++)
        mac[i] = (uint8_t)rand();
    mac[0] = (mac[0] & 0xfe) | 0x02; // locally administered, unicast
    uint32_t xid = (uint32_t)rand();

    uint8_t frame[FRAME_CAP];
    int flen = rl_build_dhcp_discover(mac, xid, frame);

    int offered = 0;
    for (int attempt = 0; attempt < 4 && !offered; attempt++) {
        if (rndis_usb_send_frame(&u, frame, flen) != 0) {
            printf("FAIL tx: bulk OUT of REMOTE_NDIS_PACKET_MSG failed\n");
            goto done;
        }
        for (int r = 0; r < 12 && !offered; r++) {
            uint8_t in[2048];
            const uint8_t *rxf = NULL;
            int rl = rndis_usb_recv_frame(&u, in, sizeof(in), &rxf);
            if (rl > 0 && rxf && rl_is_dhcp_offer(rxf, rl))
                offered = 1;
        }
    }

    if (!offered) {
        printf("FAIL dhcp: no DHCP OFFER seen (tx worked; is USB tethering "
               "on?)\n");
        goto done;
    }
    printf("PASS dhcp: REMOTE_NDIS_PACKET_MSG round-trip OK — phone's tether "
           "DHCP server offered a lease (address redacted)\n");
    printf("RESULT: userspace RNDIS DATA PLANE works — L2 frames move both ways "
           "over the phone tether from an unprivileged process.\n");

done:
    rndis_usb_close(&u);
    return 0;
}
