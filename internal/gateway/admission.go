package gateway

import (
	"crypto/ed25519"
	"sync"
	"time"

	"continuity-vpn/internal/entitlement"
	"continuity-vpn/internal/protocol"
	"continuity-vpn/pkg/clientcore"
)

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

func (s *SessionStore) put(id protocol.SessionID, key []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[id] = append([]byte(nil), key...)
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
// On any error the key is NOT registered, so admission is fail-closed.
func (a *Admitter) Admit(token string, id protocol.SessionID, key []byte) (entitlement.Claims, error) {
	claims, err := a.service.Admit(token)
	if err != nil {
		return entitlement.Claims{}, err
	}
	a.store.put(id, key)
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
