// Command continuitycore is a C-shared bridge that exposes the Go transport core
// (pkg/clientcore.Session) over a narrow C ABI so the macOS
// NEPacketTunnelProvider (Swift) can drive the proven dual-path
// duplicate/deduplicate behaviour without re-deriving the transport brain in
// Swift. See docs/adr/0005-dual-path-data-plane.md.
//
// Memory-ownership contract (per ADR-0002/ADR-0005): the bridge holds NO caller
// pointer across calls and the caller holds NO Go pointer. Sessions are
// addressed by an opaque uint64 handle (cgo forbids passing Go pointers to C and
// storing them on the C side, so a handle registry is required). All payload and
// wire bytes are copied in and out of caller-provided buffers; nothing is
// retained.
//
// This file contains only Go-typed logic (registry + marshalling) and imports no
// C, so it is unit-testable with a normal go test. The thin C-typed //export
// layer lives in bridge.go and delegates here.
package main

import (
	"sync"

	"continuity-vpn/internal/protocol"
	"continuity-vpn/pkg/clientcore"
)

// Bridge error codes returned (negated) across the C ABI. They are small and
// stable so the Swift side can map them without parsing strings. The framing
// functions return a non-negative byte count on success.
const (
	errCodeBadHandle  = 1 // no session registered for the given handle
	errCodeBufferSize = 2 // caller-provided output buffer is too small
	errCodeCore       = 3 // the transport core rejected the call
	errCodeBadArgs    = 4 // malformed fixed-size argument (id/key length)
)

// registry maps opaque uint64 handles to live sessions. A handle is never a Go
// pointer, so it is safe to hand to C and store there. Handles are monotonic and
// never reused within a process, which avoids a freed handle aliasing a fresh
// session.
type registry struct {
	mu       sync.Mutex
	next     uint64
	sessions map[uint64]*clientcore.Session
}

func newRegistry() *registry {
	return &registry{sessions: make(map[uint64]*clientcore.Session)}
}

// add registers a session and returns its handle. Handles start at 1 so that 0
// can be reserved as the "creation failed" sentinel across the C ABI.
func (r *registry) add(s *clientcore.Session) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	h := r.next
	r.sessions[h] = s
	return h
}

// get returns the session for a handle, or false if no such handle exists.
func (r *registry) get(h uint64) (*clientcore.Session, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[h]
	return s, ok
}

// remove drops a handle. Removing an unknown handle is a no-op.
func (r *registry) remove(h uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, h)
}

// reg is the process-wide registry behind the C ABI.
var reg = newRegistry()

// createSession builds a session from copies of the caller's bytes and registers
// it, returning a handle, or 0 on any error. id must be 16 bytes and key 32
// bytes; both are copied, so the caller may free its buffers immediately.
func createSession(id, key []byte, role clientcore.Role, capacity int) uint64 {
	sid, err := protocol.NewSessionID(id)
	if err != nil {
		return 0
	}
	// clientcore.NewSession copies the key into its AEAD, but copy defensively so
	// no caller-owned slice is retained by this layer.
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)

	s, err := clientcore.NewSession(sid, keyCopy, role, capacity)
	if err != nil {
		return 0
	}
	return reg.add(s)
}

// outboundInto frames+encrypts payload for the handle's session and copies the
// framed bytes into out, returning the number of bytes written or a negative
// error code. payload is copied by the core; out must be large enough.
func outboundInto(handle uint64, payload, out []byte) int {
	s, ok := reg.get(handle)
	if !ok {
		return -errCodeBadHandle
	}
	framed, err := s.Outbound(payload)
	if err != nil {
		return -errCodeCore
	}
	if len(framed) > len(out) {
		return -errCodeBufferSize
	}
	copy(out, framed)
	return len(framed)
}

// inboundInto authenticates, decrypts and deduplicates wire for the handle's
// session. On success it copies the recovered payload into out, sets *deliver to
// true for the first copy of a logical packet (deliver it) or false for a
// duplicate (drop it), and returns the payload length written. A duplicate may
// legitimately return 0 with deliver=false. On error it returns a negative error
// code and leaves deliver false.
func inboundInto(handle uint64, wire []byte, path string, out []byte, deliver *bool) int {
	*deliver = false
	s, ok := reg.get(handle)
	if !ok {
		return -errCodeBadHandle
	}
	payload, first, err := s.Inbound(wire, path)
	if err != nil {
		return -errCodeCore
	}
	if len(payload) > len(out) {
		return -errCodeBufferSize
	}
	copy(out, payload)
	*deliver = first
	return len(payload)
}
