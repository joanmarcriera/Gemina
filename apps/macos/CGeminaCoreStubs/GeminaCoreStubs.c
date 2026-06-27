/*
 * GeminaCoreStubs.c — link-time stubs for the Go transport core ABI symbols.
 *
 * The headless SwiftPM checks (WiFiPathSenderCheck, CoreTransportCheck) link
 * GeminaVPNPacketTunnelExtension, which references CGeminaCore symbols
 * (cc_session_new etc.). In the Xcode project those symbols come from the Go
 * c-archive; in the headless SwiftPM checks the archive is absent, so we provide
 * stubs.
 *
 * Most are inert no-ops (WiFiPathSender uses Network.framework only). The
 * handshake pair below is instead a DETERMINISTIC FAKE so CoreTransportCheck can
 * exercise the CoreTransport.connect Swift glue — closure ordering, buffer
 * plumbing and assigned-IP extraction — without the real Go crypto (which is
 * covered by the Go bridge tests). The fake assigned IP is 10.99.0.5.
 */

#include <stdint.h>
#include <string.h>

/* The deterministic assigned tunnel IP the fake cc_handshake_complete reports.
 * CoreTransportCheck asserts CoreTransport.connect surfaces exactly this. */
#define STUB_ASSIGNED_IP_0 10
#define STUB_ASSIGNED_IP_1 99
#define STUB_ASSIGNED_IP_2 0
#define STUB_ASSIGNED_IP_3 5

uint64_t cc_session_new(uint8_t *sessionID, uint8_t *key, int role, int capacity) {
    (void)sessionID; (void)key; (void)role; (void)capacity;
    return 0;
}

int cc_handshake_begin(uint8_t *gatewayPub, char *token,
                       uint8_t *out, int outCap, uint64_t *hsHandle) {
    (void)gatewayPub; (void)token;
    /* A fixed, non-empty fake ClientHello so connect's sendClientHello fires. */
    static const uint8_t canned[8] = {0xC0, 0xA0, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06};
    if (outCap < (int)sizeof(canned)) {
        if (hsHandle) *hsHandle = 0;
        return -2; /* CC_ERR_BUFFER_SIZE */
    }
    memcpy(out, canned, sizeof(canned));
    if (hsHandle) *hsHandle = 1;
    return (int)sizeof(canned);
}

uint64_t cc_handshake_complete(uint64_t hsHandle, uint8_t *serverHello,
                               int serverHelloLen, int capacity,
                               uint8_t *assignedIPv4) {
    (void)hsHandle; (void)serverHello; (void)serverHelloLen; (void)capacity;
    if (assignedIPv4) {
        assignedIPv4[0] = STUB_ASSIGNED_IP_0;
        assignedIPv4[1] = STUB_ASSIGNED_IP_1;
        assignedIPv4[2] = STUB_ASSIGNED_IP_2;
        assignedIPv4[3] = STUB_ASSIGNED_IP_3;
    }
    return 0x1001; /* a fixed, non-zero fake session handle */
}

int cc_outbound(uint64_t handle, uint8_t *payload, int payloadLen,
                uint8_t *out, int outCap) {
    (void)handle; (void)payload; (void)payloadLen; (void)out; (void)outCap;
    return -1; /* CC_ERR_BAD_HANDLE */
}

int cc_inbound(uint64_t handle, uint8_t *wire, int wireLen, char *path,
               uint8_t *out, int outCap, int *deliver) {
    (void)handle; (void)wire; (void)wireLen; (void)path; (void)out; (void)outCap;
    if (deliver) *deliver = 0;
    return -1; /* CC_ERR_BAD_HANDLE */
}

void cc_session_free(uint64_t handle) {
    (void)handle;
}
