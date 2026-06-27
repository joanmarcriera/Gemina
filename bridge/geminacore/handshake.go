package main

import (
	"crypto/ed25519"
	"sync"

	"github.com/joanmarcriera/gemina/pkg/clientcore"
)

// Client-side handshake support for the C ABI. The handshake (ADR-0007) is a
// two-message exchange the host (Swift) pumps over its socket: beginHandshake
// produces the ClientHello bytes to send and stashes the in-flight client state
// behind an opaque handle; completeHandshake consumes the gateway's ServerHello,
// verifies it against the pinned identity, derives the session key, and promotes
// the state into a live session in the session registry.
//
// Keeping the in-flight *clientcore.ClientHandshake Go-side (never handed to C)
// preserves the bridge's contract that C holds no Go pointer: the host carries
// only the opaque uint64 handle between the two calls. The crypto and wire format
// stay entirely in Go, so Swift never re-implements them.

// hsRegistry maps opaque handles to in-flight client handshakes, mirroring the
// session registry. Handles are monotonic and never reused, and a handshake is
// one-shot: completeHandshake removes it whether it succeeds or fails, so a
// captured handle cannot be replayed to mint extra sessions.
type hsRegistry struct {
	mu      sync.Mutex
	next    uint64
	pending map[uint64]*clientcore.ClientHandshake
}

func newHSRegistry() *hsRegistry {
	return &hsRegistry{pending: make(map[uint64]*clientcore.ClientHandshake)}
}

// add registers an in-flight handshake and returns its handle. Handles start at 1
// so 0 stays reserved as the "failed" sentinel across the C ABI.
func (r *hsRegistry) add(hs *clientcore.ClientHandshake) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	h := r.next
	r.pending[h] = hs
	return h
}

// take returns the handshake for a handle and removes it (one-shot), or false if
// no such handle exists.
func (r *hsRegistry) take(h uint64) (*clientcore.ClientHandshake, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	hs, ok := r.pending[h]
	if ok {
		delete(r.pending, h)
	}
	return hs, ok
}

// hsReg is the process-wide in-flight-handshake registry behind the C ABI.
var hsReg = newHSRegistry()

// beginHandshake starts a handshake to a gateway whose Ed25519 identity public key
// is gatewayPub, presenting token. On success it copies the ClientHello into out
// and returns (helloLen, handle): helloLen is the number of bytes written and
// handle is the non-zero in-flight handshake handle to pass to completeHandshake.
// On failure it returns a negative error code and a 0 handle:
//
//	-errCodeBadArgs    gatewayPub is not a valid Ed25519 public key
//	-errCodeBufferSize out is too small to hold the ClientHello
//	-errCodeCore       the core could not start the handshake
//
// gatewayPub and token are copied by the core; the caller may free them after.
func beginHandshake(gatewayPub []byte, token string, out []byte) (int, uint64) {
	if len(gatewayPub) != ed25519.PublicKeySize {
		return -errCodeBadArgs, 0
	}
	// Copy the pinned identity so no caller-owned slice is retained.
	pub := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(pub, gatewayPub)

	hello, hs, err := clientcore.BeginClientHandshake(pub, token)
	if err != nil {
		return -errCodeCore, 0
	}
	if len(hello) > len(out) {
		// Drop the in-flight state rather than register a handshake the caller
		// cannot send: it never reached the wire.
		return -errCodeBufferSize, 0
	}
	copy(out, hello)
	return len(hello), hsReg.add(hs)
}

// completeHandshake consumes the gateway's ServerHello for the in-flight handshake
// named by hsHandle. It verifies the gateway signature against the pinned identity,
// derives the session key, registers the resulting initiator session with the
// given inbound dedup capacity, and returns its non-zero session handle. It returns
// 0 on any failure (unknown handle, malformed or forged ServerHello, key
// derivation failure). The handshake handle is consumed on every call, success or
// failure, so it can never be reused.
//
// On success, when assignedIPv4Out is at least 4 bytes, it is filled with the
// gateway-assigned tunnel IPv4 carried in-band in the ServerHello (zero =
// unassigned). It is left untouched on failure.
func completeHandshake(hsHandle uint64, serverHello []byte, dedupCapacity int, assignedIPv4Out []byte) uint64 {
	hs, ok := hsReg.take(hsHandle)
	if !ok {
		return 0
	}
	session, err := hs.Complete(serverHello, dedupCapacity)
	if err != nil {
		return 0
	}
	if len(assignedIPv4Out) >= 4 {
		ip := session.AssignedIPv4()
		copy(assignedIPv4Out, ip[:])
	}
	return reg.add(session)
}

// cancelHandshake discards an in-flight handshake without completing it, freeing
// its state (including the client's ephemeral private key) from hsReg. The host
// calls it when it abandons a begun handshake — e.g. the socket errored before a
// ServerHello arrived — so a never-completed handshake cannot leak. Cancelling an
// unknown handle is a no-op; the handle is consumed either way and must not be
// reused.
func cancelHandshake(hsHandle uint64) {
	hsReg.take(hsHandle)
}
