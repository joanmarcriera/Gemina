package gateway

import (
	"sync"

	"continuity-vpn/internal/entitlement"
	"continuity-vpn/internal/protocol"
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

// Admitter gates session admission by entitlement. A self-hosted gateway runs the
// service in ModeOpen and admits every session; the paid hosted gateway runs in
// ModeHosted and admits a session only on a valid entitlement token. This is
// where payments connect to the data path: an unadmitted session's key is never
// registered, so its packets are rejected by the DataPlane as unknown.
type Admitter struct {
	service *entitlement.Service
	store   *SessionStore
}

// NewAdmitter ties an entitlement service to a session store.
func NewAdmitter(service *entitlement.Service, store *SessionStore) *Admitter {
	return &Admitter{service: service, store: store}
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
