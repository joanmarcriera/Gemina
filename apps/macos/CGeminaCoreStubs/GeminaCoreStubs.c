/*
 * GeminaCoreStubs.c — no-op stubs for the Go transport core ABI symbols.
 *
 * WiFiPathSenderCheck links GeminaVPNPacketTunnelExtension, which in turn
 * references CGeminaCore symbols (cc_session_new etc.).  In the Xcode project
 * those symbols come from the Go c-archive; in the headless SwiftPM check the
 * archive is absent, so we provide link-time stubs.
 *
 * The stubs are never called by WiFiPathSender (which uses Network.framework
 * exclusively) so returning dummy values is safe.
 */

#include <stdint.h>
#include <string.h>

uint64_t cc_session_new(uint8_t *sessionID, uint8_t *key, int role, int capacity) {
    (void)sessionID; (void)key; (void)role; (void)capacity;
    return 0;
}

int cc_handshake_begin(uint8_t *gatewayPub, char *token,
                       uint8_t *out, int outCap, uint64_t *hsHandle) {
    (void)gatewayPub; (void)token; (void)out; (void)outCap;
    if (hsHandle) *hsHandle = 0;
    return -3; /* CC_ERR_CORE */
}

uint64_t cc_handshake_complete(uint64_t hsHandle, uint8_t *serverHello,
                               int serverHelloLen, int capacity) {
    (void)hsHandle; (void)serverHello; (void)serverHelloLen; (void)capacity;
    return 0;
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
