package clientcore

import (
	"sync"

	"continuity-vpn/internal/dedup"
	"continuity-vpn/internal/protocol"
)

// Role is the endpoint's position in a session. The two endpoints share a key
// but seal in opposite directions, which keeps their nonces disjoint.
type Role uint8

const (
	RoleInitiator Role = iota // the client
	RoleResponder             // the gateway
)

func (r Role) sendDir() byte {
	if r == RoleResponder {
		return dirResponder
	}
	return dirInitiator
}

// peerDir is the direction this endpoint expects on inbound packets (the peer's
// send direction).
func (r Role) peerDir() byte {
	if r == RoleResponder {
		return dirInitiator
	}
	return dirResponder
}

// Session frames, encrypts and deduplicates one logical connection's packets. It
// is safe for concurrent use: Outbound is typically driven by the tunnel read
// loop while Inbound is driven by several path receivers at once.
type Session struct {
	id     protocol.SessionID
	role   Role
	sealer *sealer

	mu     sync.Mutex
	next   protocol.PacketNumber
	window *dedup.Window
}

// NewSession creates a session with the given identity, 32-byte key, role and
// inbound dedup-window capacity (the number of recent packet identities
// remembered for suppression).
func NewSession(id protocol.SessionID, key []byte, role Role, dedupCapacity int) (*Session, error) {
	if id.IsZero() {
		return nil, errBadIdentity
	}
	s, err := newSealer(key)
	if err != nil {
		return nil, err
	}
	window, err := dedup.NewWindow(dedupCapacity)
	if err != nil {
		return nil, err
	}
	return &Session{id: id, role: role, sealer: s, window: window}, nil
}

// Outbound assigns the next packet number, encrypts the payload and frames it for
// transmission. The caller sends the returned bytes over every active path; the
// peer's Inbound authenticates, decrypts and deduplicates them.
func (s *Session) Outbound(payload []byte) ([]byte, error) {
	if len(payload) > maxPayload {
		return nil, errOversize
	}
	s.mu.Lock()
	s.next++
	id := protocol.PacketID{Session: s.id, Number: s.next}
	s.mu.Unlock()

	dir := s.role.sendDir()
	header := frameHeader(id, dir)
	ciphertext := s.sealer.seal(dir, id, header, payload)
	return append(header, ciphertext...), nil
}

// Inbound authenticates and decrypts a received datagram, then reports whether
// this is the first copy of its logical packet (deliver the returned payload) or
// a duplicate (drop it). A datagram that fails authentication is rejected before
// it can touch the dedup window, so a forged identity cannot suppress a real
// packet. path is an opaque label for dedup bookkeeping/attribution.
func (s *Session) Inbound(wire []byte, path string) (payload []byte, first bool, err error) {
	id, dir, off, err := parseHeader(wire)
	if err != nil {
		return nil, false, err
	}
	if dir != s.role.peerDir() {
		return nil, false, errBadDirection
	}

	plaintext, err := s.sealer.open(dir, id, wire[:off], wire[off:])
	if err != nil {
		return nil, false, err
	}

	s.mu.Lock()
	result := s.window.Observe(id, dedup.PathID(path))
	s.mu.Unlock()

	return plaintext, result.Decision == dedup.DecisionFirstCopy, nil
}
