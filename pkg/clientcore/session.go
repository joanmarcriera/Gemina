package clientcore

import (
	"sync"

	"github.com/joanmarcriera/gemina/internal/dedup"
	"github.com/joanmarcriera/gemina/internal/protocol"
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

	// assignedIPv4 is the gateway-leased tunnel address learned from the
	// ServerHello during the client handshake (zero for a responder session or
	// when the gateway assigned none). It is read-only after construction.
	assignedIPv4 [4]byte

	mu     sync.Mutex
	next   protocol.PacketNumber
	window *dedup.ReplayWindow // RFC 6479 sliding-window anti-replay; self-synchronised
}

// AssignedIPv4 returns the gateway-leased tunnel IPv4 address for this session,
// or the zero value if none was assigned. The packet-tunnel provider uses it to
// build the interface's NEPacketTunnelNetworkSettings.
func (s *Session) AssignedIPv4() [4]byte {
	return s.assignedIPv4
}

// NewSession creates a session with the given identity, 32-byte key, role and
// inbound dedup-window width (the span of recent packet numbers remembered for
// replay suppression).
func NewSession(id protocol.SessionID, key []byte, role Role, dedupCapacity int) (*Session, error) {
	if id.IsZero() {
		return nil, errBadIdentity
	}
	s, err := newSealer(key)
	if err != nil {
		return nil, err
	}
	window, err := dedup.NewReplayWindow(dedupCapacity)
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
// a duplicate/stale copy (drop it). A datagram that fails authentication is
// rejected before it can touch the replay window, so a forged identity cannot
// suppress a real packet. path is retained in the signature for attribution and
// logging by callers; the number-keyed replay window does not use it.
func (s *Session) Inbound(wire []byte, path string) (payload []byte, first bool, err error) {
	payload, decision, err := s.InboundClassified(wire, path)
	if err != nil {
		return nil, false, err
	}
	return payload, decision == dedup.ReplayFirstCopy, nil
}

// InboundClassified is identical to Inbound but returns the full ReplayDecision
// so callers that need to distinguish stale replays from in-window duplicates
// can act accordingly (e.g. the gateway data plane may log stale packets
// separately for abuse-detection purposes).
func (s *Session) InboundClassified(wire []byte, path string) (payload []byte, decision dedup.ReplayDecision, err error) {
	id, dir, off, err := parseHeader(wire)
	if err != nil {
		return nil, dedup.ReplayInvalid, err
	}
	if dir != s.role.peerDir() {
		return nil, dedup.ReplayInvalid, errBadDirection
	}

	plaintext, err := s.sealer.open(dir, id, wire[:off], wire[off:])
	if err != nil {
		return nil, dedup.ReplayInvalid, err
	}

	// ReplayWindow is self-synchronised; no outer lock needed for Observe.
	d := s.window.Observe(id.Number)
	return plaintext, d, nil
}
