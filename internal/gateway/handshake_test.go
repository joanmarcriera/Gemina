package gateway

import (
	"bytes"
	"testing"
	"time"

	"continuity-vpn/pkg/clientcore"
)

func TestHandshakeRejectsStaleClientHello(t *testing.T) {
	idPriv, idPub, _ := clientcore.GenerateIdentity()
	service, key := hostedService(t)
	admitter := NewAdmitter(service, NewSessionStore())

	// Pin the gateway's clock; craft a ClientHello stamped well outside tolerance.
	gatewayNow := time.Unix(2_000_000_000, 0)
	admitter.now = func() time.Time { return gatewayNow }
	staleTS := gatewayNow.Add(-10 * time.Minute).Unix()

	_, eph, _ := clientcore.GenerateKeyPair()
	_ = idPub
	stale, err := clientcore.EncodeClientHello(sessionID(0x5E), staleTS, eph, hostedToken(t, key))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if _, _, err := admitter.Handshake(stale, idPriv, 64); err == nil {
		t.Fatal("gateway accepted a stale ClientHello (replay window not enforced)")
	}
}

func TestHandshakeEndToEndAdmitsAndCarriesTraffic(t *testing.T) {
	idPriv, idPub, err := clientcore.GenerateIdentity()
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	service, key := hostedService(t)
	store := NewSessionStore()
	admitter := NewAdmitter(service, store)
	dp := NewDataPlane(store, 64)

	// Client begins, pinning the real gateway identity and presenting its token.
	hello, hs, err := clientcore.BeginClientHandshake(idPub, hostedToken(t, key))
	if err != nil {
		t.Fatalf("begin: %v", err)
	}

	// Gateway admits and answers.
	serverHello, claims, err := admitter.Handshake(hello, idPriv, 64)
	if err != nil {
		t.Fatalf("gateway handshake: %v", err)
	}
	if claims.Tier == "" {
		t.Fatal("admitted claims empty")
	}

	// Client completes into a ready session.
	session, err := hs.Complete(serverHello, 64)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	// Real encrypted traffic now flows and is decrypted+dedup'd by the gateway.
	wire, _ := session.Outbound([]byte("end-to-end after handshake"))
	payload, first, err := dp.Handle(wire, "wifi")
	if err != nil || !first || !bytes.Equal(payload, []byte("end-to-end after handshake")) {
		t.Fatalf("post-handshake traffic: first=%v err=%v payload=%q", first, err, payload)
	}
}

func TestHandshakeRejectsUnentitledClient(t *testing.T) {
	idPriv, idPub, _ := clientcore.GenerateIdentity()
	service, _ := hostedService(t)
	store := NewSessionStore()
	admitter := NewAdmitter(service, store)

	// No token in hosted mode -> rejected; nothing registered.
	hello, _, _ := clientcore.BeginClientHandshake(idPub, "")
	if _, _, err := admitter.Handshake(hello, idPriv, 64); err == nil {
		t.Fatal("gateway admitted a client with no entitlement token")
	}
	if got := len(store.keys); got != 0 {
		t.Fatalf("rejected handshake registered %d session(s)", got)
	}
}

func TestHandshakeClientRejectsWrongGatewayIdentity(t *testing.T) {
	idPriv, _, _ := clientcore.GenerateIdentity()
	_, wrongPub, _ := clientcore.GenerateIdentity() // client pins the WRONG identity
	service, key := hostedService(t)
	admitter := NewAdmitter(service, NewSessionStore())

	hello, hs, _ := clientcore.BeginClientHandshake(wrongPub, hostedToken(t, key))
	serverHello, _, err := admitter.Handshake(hello, idPriv, 64)
	if err != nil {
		t.Fatalf("gateway handshake: %v", err)
	}
	if _, err := hs.Complete(serverHello, 64); err == nil {
		t.Fatal("client completed against a gateway it could not authenticate")
	}
}
