/*
 * geminacore.h — narrow C ABI for the Go transport core.
 *
 * This is the stable, hand-authored boundary the macOS NEPacketTunnelProvider
 * (Swift) links against to drive pkg/clientcore.Session. It mirrors the symbols
 * exported by bridge/geminacore (cgo //export) and is the contract the Swift
 * side should import; the cgo-generated header is a build artefact and is not
 * checked in.
 *
 * See docs/adr/0005-dual-path-data-plane.md.
 *
 * Memory-ownership contract (per ADR-0002/ADR-0005):
 *   - The caller (Swift) owns all buffers. Input bytes are copied into Go memory
 *     during the call, so input buffers may be freed as soon as the call returns.
 *   - The bridge holds no caller pointer across calls and returns no Go pointer.
 *   - Sessions are addressed by an opaque uint64 handle, never a Go pointer.
 *   - Output is written into caller-provided buffers; the caller sizes them.
 *
 * Error codes: framing functions return a non-negative byte count on success, or
 * one of the negative codes below on failure.
 */

#ifndef GEMINACORE_H
#define GEMINACORE_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* Negative return codes from cc_outbound / cc_inbound. */
#define CC_ERR_BAD_HANDLE  (-1) /* no session for the given handle           */
#define CC_ERR_BUFFER_SIZE (-2) /* caller output buffer too small            */
#define CC_ERR_CORE        (-3) /* transport core rejected the call          */

/* Role values for cc_session_new. */
#define CC_ROLE_INITIATOR  0    /* the client */
#define CC_ROLE_RESPONDER  1    /* the gateway */

/*
 * cc_session_new creates a session and returns an opaque non-zero handle, or 0
 * on any error.
 *
 *   sessionID  pointer to exactly 16 bytes (must not be all-zero)
 *   key        pointer to exactly 32 bytes
 *   role       CC_ROLE_INITIATOR or CC_ROLE_RESPONDER
 *   capacity   inbound dedup-window capacity (recent identities remembered)
 *
 * Both buffers are copied; free them after the call if you wish.
 */
extern uint64_t cc_session_new(uint8_t *sessionID, uint8_t *key, int role, int capacity);

/*
 * cc_handshake_begin starts a client handshake to a gateway (ADR-0007). It does
 * the key agreement and wire framing in Go so the host never re-implements the
 * crypto; the host only pumps the two messages over its socket.
 *
 *   gatewayPub  pointer to exactly 32 bytes — the gateway's pinned Ed25519
 *               identity public key
 *   token       NUL-terminated entitlement token presented to the gateway
 *   out         caller buffer that receives the ClientHello to send
 *   outCap      capacity of out in bytes
 *   hsHandle    set to a non-zero in-flight handshake handle on success, else 0
 *
 * Returns the ClientHello length written into out, or a negative CC_ERR_* code
 * (CC_ERR_BUFFER_SIZE if out is too small, CC_ERR_CORE on a core failure). Send
 * the ClientHello to the gateway, then pass its ServerHello reply together with
 * *hsHandle to cc_handshake_complete. Both inputs are copied.
 */
extern int cc_handshake_begin(uint8_t *gatewayPub, char *token,
                              uint8_t *out, int outCap, uint64_t *hsHandle);

/*
 * cc_handshake_complete consumes the gateway's ServerHello for the in-flight
 * handshake named by hsHandle. It verifies the gateway signature against the
 * pinned identity, derives the session key, and returns a session handle ready
 * for cc_outbound / cc_inbound.
 *
 *   hsHandle        in-flight handle from cc_handshake_begin
 *   serverHello     pointer to serverHelloLen received ServerHello bytes (copied)
 *   serverHelloLen  length of serverHello in bytes
 *   capacity        inbound dedup-window capacity for the new session
 *   assignedIPv4    optional 4-byte out buffer (may be NULL); on success it is
 *                   filled with the gateway-assigned tunnel IPv4 carried in-band
 *                   in the ServerHello (all zero = unassigned), left untouched on
 *                   error. Used to build the packet tunnel's network settings.
 *
 * Returns a non-zero session handle on success, or 0 on any error (unknown
 * handle, malformed or forged ServerHello). The handshake handle is consumed on
 * every call, success or failure, so it must not be reused.
 */
extern uint64_t cc_handshake_complete(uint64_t hsHandle, uint8_t *serverHello,
                                      int serverHelloLen, int capacity,
                                      uint8_t *assignedIPv4);

/*
 * cc_outbound frames and encrypts a payload for transmission.
 *
 *   handle      session handle from cc_session_new
 *   payload     pointer to payloadLen plaintext bytes (copied)
 *   payloadLen  length of payload in bytes
 *   out         caller buffer that receives the framed datagram
 *   outCap      capacity of out in bytes
 *
 * Returns the number of bytes written into out, or a negative CC_ERR_* code.
 * Send the framed bytes unchanged over every active path.
 */
extern int cc_outbound(uint64_t handle, uint8_t *payload, int payloadLen,
                       uint8_t *out, int outCap);

/*
 * cc_inbound authenticates, decrypts and deduplicates a received datagram.
 *
 *   handle    session handle from cc_session_new
 *   wire      pointer to wireLen received framed bytes (copied)
 *   wireLen   length of wire in bytes
 *   path      NUL-terminated opaque path label (e.g. "wifi", "cellular")
 *   out       caller buffer that receives the recovered payload
 *   outCap    capacity of out in bytes
 *   deliver   set to 1 for the first copy (deliver out) or 0 for a duplicate
 *
 * Returns the payload length written into out (0 is valid, e.g. a suppressed
 * duplicate), or a negative CC_ERR_* code. On error, *deliver is set to 0.
 */
extern int cc_inbound(uint64_t handle, uint8_t *wire, int wireLen, char *path,
                      uint8_t *out, int outCap, int *deliver);

/*
 * cc_session_free removes a session. Freeing an unknown handle is a no-op. After
 * this call the handle must not be reused.
 */
extern void cc_session_free(uint64_t handle);

#ifdef __cplusplus
}
#endif

#endif /* GEMINACORE_H */
