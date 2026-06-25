package gateway

import (
	"crypto/ed25519"
	"errors"
	"sync"
	"time"

	"continuity-vpn/internal/entitlement"
	"continuity-vpn/internal/protocol"
	"continuity-vpn/pkg/clientcore"
)

// ErrSessionReused is returned when admission is attempted for a SessionID the
// gateway has already admitted. A SessionID/key pair is single-use for the
// gateway's lifetime: a fresh handshake mints a fresh random SessionID and
// derives a fresh key, so a repeated SessionID is either a duplicate handshake
// or an attempt to rebind a live session to an attacker-chosen key. Re-admission
// is refused fail-closed, so a reused id can never be decrypted into a fresh
// session. See docs/security/stage-2-session-security.md.
var ErrSessionReused = errors.New("session id already admitted")

// SessionStore holds the keys of admitted sessions and resolves them for the
// DataPlane. It implements KeyResolver, so the data plane only ever decrypts
// traffic for a session that admission has registered.
type SessionStore struct {
	mu   sync.RWMutex
	keys map[protocol.SessionID][]byte
}

// NewSessionStore returns an empty store.
func NewSessionStore() *SessionStore {
	return &SessionStore{keys: make(map[protocol.SessionID][]byte)}
}

// SessionKey implements KeyResolver.
func (s *SessionStore) SessionKey(id protocol.SessionID) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.keys[id]
	return k, ok
}

// register stores key under id for a newly admitted session and reports whether
// the id was free. It refuses to overwrite an existing id, enforcing that a
// SessionID/key pair is single-use for the gateway's lifetime: a reused id is
// never rebound to a fresh key. The test-and-set is atomic under the lock, so
// two concurrent handshakes for the same id cannot both register.
func (s *SessionStore) register(id protocol.SessionID, key []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.keys[id]; exists {
		return false
	}
	s.keys[id] = append([]byte(nil), key...)
	return true
}

// Forget removes a session's key, e.g. when it ends or its entitlement lapses.
func (s *SessionStore) Forget(id protocol.SessionID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, id)
}

// len reports the number of admitted sessions.
func (s *SessionStore) len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.keys)
}

// Admitter gates session admission by entitlement. A self-hosted gateway runs the
// service in ModeOpen and admits every session; the paid hosted gateway runs in
// ModeHosted and admits a session only on a valid entitlement token. This is
// where payments connect to the data path: an unadmitted session's key is never
// registered, so its packets are rejected by the DataPlane as unknown.
type Admitter struct {
	service   *entitlement.Service
	store     *SessionStore
	now       func() time.Time // injectable for tests
	tolerance time.Duration
}

// NewAdmitter ties an entitlement service to a session store.
func NewAdmitter(service *entitlement.Service, store *SessionStore) *Admitter {
	return &Admitter{
		service:   service,
		store:     store,
		now:       time.Now,
		tolerance: clientcore.DefaultHandshakeTolerance,
	}
}

// Admit checks the entitlement and, on success, registers the session key so the
// DataPlane will accept the session's packets. It returns the admitted claims.
// On any error the key is NOT registered, so admission is fail-closed. Admission
// of an already-admitted SessionID is refused with ErrSessionReused, keeping a
// SessionID/key pair single-use for the gateway's lifetime.
func (a *Admitter) Admit(token string, id protocol.SessionID, key []byte) (entitlement.Claims, error) {
	claims, err := a.service.Admit(token)
	if err != nil {
		return entitlement.Claims{}, err
	}
	if !a.store.register(id, key) {
		return entitlement.Claims{}, ErrSessionReused
	}
	return claims, nil
}

// Handshake is the gateway side of the on-wire handshake. It decodes the
// client's ClientHello, derives the session key from a fresh gateway ephemeral
// key and the client's ephemeral key, admits the client by its entitlement token
// (registering the key only if admitted — fail-closed), and returns a ServerHello
// signed with the gateway's Ed25519 identity for the client to authenticate. The
// admitted session id is returned so the caller can allocate a tunnel-IP lease
// for the exit path.
func (a *Admitter) Handshake(clientHello []byte, identityPriv ed25519.PrivateKey, dedupCapacity int) (serverHello []byte, claims entitlement.Claims, id protocol.SessionID, err error) {
	id, timestamp, clientEph, token, err := clientcore.DecodeClientHello(clientHello)
	if err != nil {
		return nil, entitlement.Claims{}, protocol.SessionID{}, err
	}
	if err := clientcore.CheckHandshakeFresh(timestamp, a.now(), a.tolerance); err != nil {
		return nil, entitlement.Claims{}, protocol.SessionID{}, err
	}

	gatewayEphPriv, gatewayEphPub, err := clientcore.GenerateKeyPair()
	if err != nil {
		return nil, entitlement.Claims{}, protocol.SessionID{}, err
	}
	key, err := clientcore.DeriveSessionKey(gatewayEphPriv, clientEph, id)
	if err != nil {
		return nil, entitlement.Claims{}, protocol.SessionID{}, err
	}

	// Admit before revealing the signed ServerHello; on rejection the key is not
	// registered and the DataPlane will refuse the session's packets.
	claims, err = a.Admit(token, id, key)
	if err != nil {
		return nil, entitlement.Claims{}, protocol.SessionID{}, err
	}

	sig := clientcore.SignHandshake(identityPriv, gatewayEphPub, id)
	serverHello, err = clientcore.EncodeServerHello(id, gatewayEphPub, sig)
	if err != nil {
		a.store.Forget(id) // unwind the registration if we cannot answer
		return nil, entitlement.Claims{}, protocol.SessionID{}, err
	}
	return serverHello, claims, id, nil
}
