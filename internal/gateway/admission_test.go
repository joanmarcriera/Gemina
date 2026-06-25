package gateway

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"continuity-vpn/internal/entitlement"
	"continuity-vpn/pkg/clientcore"
)

func hostedService(t *testing.T) (*entitlement.Service, []byte) {
	t.Helper()
	key := bytes.Repeat([]byte{0x6B}, 32)
	return &entitlement.Service{Mode: entitlement.ModeHosted, Key: key}, key
}

func hostedToken(t *testing.T, key []byte) string {
	t.Helper()
	token, err := entitlement.Issue(entitlement.Claims{
		Subject: "acct-opaque-1",
		Tier:    entitlement.TierHosted,
		Expiry:  time.Now().Add(time.Hour).Unix(),
	}, key)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return token
}

func TestAdmitterOpenModeAdmitsWithoutToken(t *testing.T) {
	store := NewSessionStore()
	a := NewAdmitter(&entitlement.Service{Mode: entitlement.ModeOpen}, store)

	id := sessionID(0x01)
	sessKey := bytes.Repeat([]byte{0x11}, 32)
	if _, err := a.Admit("", id, sessKey); err != nil {
		t.Fatalf("open mode should admit without a token: %v", err)
	}
	if _, ok := store.SessionKey(id); !ok {
		t.Fatal("admitted session key not registered")
	}
}

func TestAdmitterHostedModeRequiresValidToken(t *testing.T) {
	service, key := hostedService(t)
	store := NewSessionStore()
	a := NewAdmitter(service, store)

	id := sessionID(0x02)
	sessKey := bytes.Repeat([]byte{0x22}, 32)

	// No token: rejected, nothing registered.
	if _, err := a.Admit("", id, sessKey); err == nil {
		t.Fatal("hosted mode admitted without a token")
	}
	if _, ok := store.SessionKey(id); ok {
		t.Fatal("rejected session must not be registered")
	}

	// Valid token: admitted and registered.
	if _, err := a.Admit(hostedToken(t, key), id, sessKey); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	got, ok := store.SessionKey(id)
	if !ok || !bytes.Equal(got, sessKey) {
		t.Fatal("valid admission did not register the session key")
	}
}

func TestAdmitterRejectsReusedSessionID(t *testing.T) {
	store := NewSessionStore()
	a := NewAdmitter(&entitlement.Service{Mode: entitlement.ModeOpen}, store)
	dp := NewDataPlane(store, 64)

	id := sessionID(0x05)
	firstKey := bytes.Repeat([]byte{0x55}, 32)
	if _, err := a.Admit("", id, firstKey); err != nil {
		t.Fatalf("first admit: %v", err)
	}

	// Re-admitting the same SessionID with a fresh key is refused fail-closed: a
	// SessionID/key pair is single-use for the gateway's lifetime.
	freshKey := bytes.Repeat([]byte{0x66}, 32)
	if _, err := a.Admit("", id, freshKey); !errors.Is(err, ErrSessionReused) {
		t.Fatalf("reused session id not refused: err=%v", err)
	}

	// The originally admitted key is untouched...
	got, ok := store.SessionKey(id)
	if !ok || !bytes.Equal(got, firstKey) {
		t.Fatal("reuse attempt overwrote the admitted session key")
	}

	// ...so traffic sealed under the rebind-rejected key cannot decrypt into the
	// session — the reused id is never admitted into a fresh session.
	attacker, err := clientcore.NewSession(id, freshKey, clientcore.RoleInitiator, 64)
	if err != nil {
		t.Fatalf("attacker session: %v", err)
	}
	wire, _ := attacker.Outbound([]byte("forged"))
	if _, _, err := dp.Handle(wire, "wifi"); err == nil {
		t.Fatal("data plane accepted a packet sealed under a rebind-rejected key")
	}
}

func TestDataPlaneRejectsUnadmittedSession(t *testing.T) {
	store := NewSessionStore()
	dp := NewDataPlane(store, 64)

	// A real client packet for a session the gateway never admitted.
	id := sessionID(0x03)
	clientKey := bytes.Repeat([]byte{0x33}, 32)
	client, err := clientcore.NewSession(id, clientKey, clientcore.RoleInitiator, 64)
	if err != nil {
		t.Fatalf("client session: %v", err)
	}
	wire, _ := client.Outbound([]byte("hi"))

	if _, _, err := dp.Handle(wire, "wifi"); err == nil {
		t.Fatal("data plane accepted a packet for an unadmitted session")
	}
}

func TestAdmittedSessionFlowsThroughDataPlane(t *testing.T) {
	service, key := hostedService(t)
	store := NewSessionStore()
	a := NewAdmitter(service, store)
	dp := NewDataPlane(store, 64)

	id := sessionID(0x04)
	sessKey := bytes.Repeat([]byte{0x44}, 32)
	if _, err := a.Admit(hostedToken(t, key), id, sessKey); err != nil {
		t.Fatalf("admit: %v", err)
	}

	client, _ := clientcore.NewSession(id, sessKey, clientcore.RoleInitiator, 64)
	wire, _ := client.Outbound([]byte("real traffic"))
	payload, first, err := dp.Handle(wire, "wifi")
	if err != nil || !first || !bytes.Equal(payload, []byte("real traffic")) {
		t.Fatalf("admitted session did not flow: first=%v err=%v payload=%q", first, err, payload)
	}
}
