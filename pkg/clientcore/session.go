package clientcore

import (
	"sync"

	"continuity-vpn/internal/dedup"
	"continuity-vpn/internal/protocol"
)

// Session frames and deduplicates one logical connection's packets. It is safe
// for concurrent use: Outbound is typically driven by the tunnel read loop while
// Inbound is driven by several path receivers at once.
type Session struct {
	id protocol.SessionID

	mu     sync.Mutex
	next   protocol.PacketNumber
	window *dedup.Window
}

// NewSession creates a session with the given identity and inbound dedup-window
// capacity (the number of recent packet identities remembered for suppression).
func NewSession(id protocol.SessionID, dedupCapacity int) (*Session, error) {
	if id.IsZero() {
		return nil, errBadIdentity
	}
	window, err := dedup.NewWindow(dedupCapacity)
	if err != nil {
		return nil, err
	}
	return &Session{id: id, window: window}, nil
}

// Outbound assigns the next packet number and frames the payload for
// transmission. The caller sends the returned bytes over every active path; the
// peer's Inbound deduplicates them.
func (s *Session) Outbound(payload []byte) ([]byte, error) {
	if len(payload) > maxPayload {
		return nil, errOversize
	}
	s.mu.Lock()
	s.next++
	id := protocol.PacketID{Session: s.id, Number: s.next}
	s.mu.Unlock()
	return encodeData(id, payload)
}

// Inbound decodes a received datagram and reports whether this is the first copy
// of its logical packet (deliver it) or a duplicate (drop it). path is an opaque
// label used only for dedup bookkeeping/attribution.
func (s *Session) Inbound(wire []byte, path string) (payload []byte, first bool, err error) {
	payload, id, err := decodeData(wire)
	if err != nil {
		return nil, false, err
	}

	s.mu.Lock()
	result := s.window.Observe(id, dedup.PathID(path))
	s.mu.Unlock()

	return payload, result.Decision == dedup.DecisionFirstCopy, nil
}
