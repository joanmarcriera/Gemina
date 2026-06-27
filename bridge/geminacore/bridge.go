package main

// This file is the thin C-typed boundary. It converts C types to Go types,
// copying every byte across the boundary, and delegates all logic to the
// Go-typed, unit-tested helpers in registry.go. It must stay free of business
// logic so the testable layer remains the single source of truth.
//
// Memory-ownership contract: each function copies the caller's input bytes into
// Go memory and writes results into caller-provided output buffers. The bridge
// retains no caller pointer beyond the call, and returns no Go pointer. Sessions
// are addressed by an opaque uint64 handle, never a Go pointer.
//
// No key material or payload bytes are ever logged.

// #include <stdint.h>
// #include <string.h>
import "C"

import (
	"unsafe"

	"github.com/joanmarcriera/gemina/pkg/clientcore"
)

// cc_session_new creates a session from a 16-byte session id and a 32-byte key,
// with the given role (0 = initiator, 1 = responder) and inbound dedup-window
// capacity. It returns an opaque non-zero handle on success, or 0 on any error
// (bad id/key length, zero id, bad key, or bad capacity). Both buffers are
// copied; the caller may free them immediately after the call returns.
//
//export cc_session_new
func cc_session_new(sessionID *C.uint8_t, key *C.uint8_t, role C.int, capacity C.int) C.uint64_t {
	id := C.GoBytes(unsafe.Pointer(sessionID), 16)
	keyBytes := C.GoBytes(unsafe.Pointer(key), 32)

	var r clientcore.Role
	switch role {
	case 0:
		r = clientcore.RoleInitiator
	case 1:
		r = clientcore.RoleResponder
	default:
		return 0
	}

	return C.uint64_t(createSession(id, keyBytes, r, int(capacity)))
}

// cc_handshake_begin starts a client handshake to a gateway whose Ed25519 identity
// public key is the 32 bytes at gatewayPub, presenting the NUL-terminated token.
// On success it writes the ClientHello into out (capacity outCap), sets *hsHandle
// to a non-zero in-flight handshake handle, and returns the ClientHello length.
// On failure it returns a negative error code (-2 buffer too small, -3 core error)
// and sets *hsHandle to 0. gatewayPub and token are copied; the caller may free
// them after the call. Send the ClientHello bytes to the gateway, then pass the
// ServerHello reply and *hsHandle to cc_handshake_complete.
//
//export cc_handshake_begin
func cc_handshake_begin(gatewayPub *C.uint8_t, token *C.char, out *C.uint8_t, outCap C.int, hsHandle *C.uint64_t) C.int {
	pub := C.GoBytes(unsafe.Pointer(gatewayPub), 32)
	tok := C.GoString(token)
	dst := make([]byte, int(outCap))

	n, h := beginHandshake(pub, tok, dst)
	*hsHandle = C.uint64_t(h)
	if n < 0 {
		return C.int(n)
	}
	if n > 0 {
		C.memcpy(unsafe.Pointer(out), unsafe.Pointer(&dst[0]), C.size_t(n))
	}
	return C.int(n)
}

// cc_handshake_complete consumes the gateway's ServerHello (serverHelloLen bytes at
// serverHello) for the in-flight handshake named by hsHandle. It verifies the
// gateway signature against the pinned identity, derives the session key, and
// returns a non-zero session handle for cc_outbound/cc_inbound, or 0 on any error
// (unknown handle, malformed or forged ServerHello). The handshake handle is
// consumed on every call, so it must not be reused. The wire bytes are copied.
//
// assignedIPv4 is an optional caller-allocated 4-byte buffer (may be NULL). On a
// successful handshake it is filled with the gateway-assigned tunnel IPv4 carried
// in-band in the ServerHello (all zero = unassigned), which the packet-tunnel
// provider uses to build its NEPacketTunnelNetworkSettings. It is left untouched
// on error.
//
//export cc_handshake_complete
func cc_handshake_complete(hsHandle C.uint64_t, serverHello *C.uint8_t, serverHelloLen C.int, dedupCapacity C.int, assignedIPv4 *C.uint8_t) C.uint64_t {
	wire := C.GoBytes(unsafe.Pointer(serverHello), serverHelloLen)
	var ip [4]byte
	handle := completeHandshake(uint64(hsHandle), wire, int(dedupCapacity), ip[:])
	if handle != 0 && assignedIPv4 != nil {
		C.memcpy(unsafe.Pointer(assignedIPv4), unsafe.Pointer(&ip[0]), 4)
	}
	return C.uint64_t(handle)
}

// cc_handshake_cancel discards an in-flight handshake named by hsHandle without
// completing it, freeing its state (including the ephemeral private key). Call it
// when abandoning a handshake begun with cc_handshake_begin — e.g. the socket
// errored before a ServerHello arrived — so it cannot leak. Cancelling an unknown
// handle is a no-op; the handle is consumed and must not be reused.
//
//export cc_handshake_cancel
func cc_handshake_cancel(hsHandle C.uint64_t) {
	cancelHandshake(uint64(hsHandle))
}

// cc_outbound frames+encrypts payloadLen bytes at payload for the session named
// by handle, writing the framed datagram into the out buffer (capacity outCap).
// It returns the number of bytes written into out, or a negative error code:
// -1 bad handle, -2 buffer too small, -3 core error. The payload is copied; out
// must be at least the framed size (payload size plus header and AEAD overhead).
//
//export cc_outbound
func cc_outbound(handle C.uint64_t, payload *C.uint8_t, payloadLen C.int, out *C.uint8_t, outCap C.int) C.int {
	in := C.GoBytes(unsafe.Pointer(payload), payloadLen)
	dst := make([]byte, int(outCap))

	n := outboundInto(uint64(handle), in, dst)
	if n < 0 {
		return C.int(n)
	}
	if n > 0 {
		C.memcpy(unsafe.Pointer(out), unsafe.Pointer(&dst[0]), C.size_t(n))
	}
	return C.int(n)
}

// cc_inbound authenticates, decrypts and deduplicates wireLen bytes at wire for
// the session named by handle, using the NUL-terminated path label for dedup
// bookkeeping. On success it writes the recovered payload into out (capacity
// outCap), sets *deliver to 1 for the first copy of a logical packet (deliver
// it) or 0 for a duplicate (drop it), and returns the payload length written.
// A duplicate may return 0 length with *deliver == 0. On error it returns a
// negative error code (-1 bad handle, -2 buffer too small, -3 core error) and
// sets *deliver to 0. The wire bytes are copied; out must be large enough for
// the recovered payload.
//
//export cc_inbound
func cc_inbound(handle C.uint64_t, wire *C.uint8_t, wireLen C.int, path *C.char, out *C.uint8_t, outCap C.int, deliver *C.int) C.int {
	in := C.GoBytes(unsafe.Pointer(wire), wireLen)
	p := C.GoString(path)
	dst := make([]byte, int(outCap))

	var delivered bool
	n := inboundInto(uint64(handle), in, p, dst, &delivered)

	if delivered {
		*deliver = 1
	} else {
		*deliver = 0
	}

	if n < 0 {
		return C.int(n)
	}
	if n > 0 {
		C.memcpy(unsafe.Pointer(out), unsafe.Pointer(&dst[0]), C.size_t(n))
	}
	return C.int(n)
}

// cc_session_free removes the session named by handle. Freeing an unknown handle
// is a no-op. After this call the handle must not be reused.
//
//export cc_session_free
func cc_session_free(handle C.uint64_t) {
	reg.remove(uint64(handle))
}

// main is required for a //export c-archive build; it is never executed when the
// archive is linked into the host process.
func main() {}
