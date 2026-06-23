package gateway

import (
	"errors"
	"sync"

	"continuity-vpn/internal/protocol"
	"continuity-vpn/pkg/clientcore"
)

// ErrUnknownSession is returned for a datagram whose session has no key — the
// gateway will not decrypt traffic it has no key for.
var ErrUnknownSession = errors.New("no key for session")

// KeyResolver supplies the pre-shared key for a session. The gateway holds one
// key per admitted client; distribution is the job of the future handshake /
// entitlement path (the self-hosted gateway may load keys from config).
type KeyResolver interface {
	SessionKey(id protocol.SessionID) ([]byte, bool)
}

// DataPlane decrypts and deduplicates CVD1 data packets on the gateway. It keeps
// one responder session per client session, so a logical packet duplicated over
// several paths is decrypted and delivered once. It is safe for concurrent use
// across paths and sessions.
type DataPlane struct {
	resolver KeyResolver
	capacity int

	mu       sync.Mutex
	sessions map[protocol.SessionID]*clientcore.Session
}

// NewDataPlane creates a data plane resolving keys via resolver and giving each
// session a dedup window of the given capacity.
func NewDataPlane(resolver KeyResolver, dedupCapacity int) *DataPlane {
	return &DataPlane{
		resolver: resolver,
		capacity: dedupCapacity,
		sessions: make(map[protocol.SessionID]*clientcore.Session),
	}
}

// Handle authenticates, decrypts and deduplicates one received datagram. It
// returns the decrypted payload and whether this is the first copy to forward
// (false means it was a duplicate, or — with a nil error never paired with a
// true — it was rejected). A datagram for an unknown session, or one that fails
// authentication, returns an error and never reaches the dedup window.
func (d *DataPlane) Handle(datagram []byte, path string) (payload []byte, first bool, err error) {
	id, err := clientcore.SessionIDFromDatagram(datagram)
	if err != nil {
		return nil, false, err
	}
	session, err := d.sessionFor(id)
	if err != nil {
		return nil, false, err
	}
	return session.Inbound(datagram, path)
}

func (d *DataPlane) sessionFor(id protocol.SessionID) (*clientcore.Session, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if s, ok := d.sessions[id]; ok {
		return s, nil
	}
	key, ok := d.resolver.SessionKey(id)
	if !ok {
		return nil, ErrUnknownSession
	}
	s, err := clientcore.NewSession(id, key, clientcore.RoleResponder, d.capacity)
	if err != nil {
		return nil, err
	}
	d.sessions[id] = s
	return s, nil
}
