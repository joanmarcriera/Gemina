/*
 * continuitycore.h — narrow C ABI for the Go transport core.
 *
 * This is the stable, hand-authored boundary the macOS NEPacketTunnelProvider
 * (Swift) links against to drive pkg/clientcore.Session. It mirrors the symbols
 * exported by bridge/continuitycore (cgo //export) and is the contract the Swift
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

#ifndef CONTINUITYCORE_H
#define CONTINUITYCORE_H

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

#endif /* CONTINUITYCORE_H */
