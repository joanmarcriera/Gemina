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

	"continuity-vpn/pkg/clientcore"
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
