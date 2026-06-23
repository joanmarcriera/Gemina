package clientcore

import (
	"bytes"
	"testing"
)

func TestHandshakeBothSidesDeriveSameKey(t *testing.T) {
	clientPriv, clientPub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("client keypair: %v", err)
	}
	gatewayPriv, gatewayPub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("gateway keypair: %v", err)
	}
	id := sessionID(0x44)

	clientKey, err := DeriveSessionKey(clientPriv, gatewayPub, id)
	if err != nil {
		t.Fatalf("client derive: %v", err)
	}
	gatewayKey, err := DeriveSessionKey(gatewayPriv, clientPub, id)
	if err != nil {
		t.Fatalf("gateway derive: %v", err)
	}

	if len(clientKey) != keySize {
		t.Fatalf("derived key length = %d, want %d", len(clientKey), keySize)
	}
	if !bytes.Equal(clientKey, gatewayKey) {
		t.Fatal("the two endpoints derived different session keys")
	}
}

func TestHandshakeKeyDiffersBySession(t *testing.T) {
	cPriv, cPub, _ := GenerateKeyPair()
	gPriv, gPub, _ := GenerateKeyPair()

	k1, _ := DeriveSessionKey(cPriv, gPub, sessionID(0x01))
	k2, _ := DeriveSessionKey(gPriv, cPub, sessionID(0x02))
	// Different session ids must not collide on a key even with the same parties.
	if bytes.Equal(k1, k2) {
		t.Fatal("different session ids produced the same key")
	}
}

func TestHandshakeKeyDiffersByPeer(t *testing.T) {
	cPriv, _, _ := GenerateKeyPair()
	_, gPub, _ := GenerateKeyPair()
	_, attackerPub, _ := GenerateKeyPair()
	id := sessionID(0x55)

	withGateway, _ := DeriveSessionKey(cPriv, gPub, id)
	withAttacker, _ := DeriveSessionKey(cPriv, attackerPub, id)
	if bytes.Equal(withGateway, withAttacker) {
		t.Fatal("a different peer public key produced the same session key")
	}
}

func TestDeriveSessionKeyRejectsBadInputs(t *testing.T) {
	_, pub, _ := GenerateKeyPair()
	if _, err := DeriveSessionKey([]byte("short"), pub, sessionID(1)); err == nil {
		t.Fatal("accepted a short private key")
	}
	priv, _, _ := GenerateKeyPair()
	if _, err := DeriveSessionKey(priv, []byte("short"), sessionID(1)); err == nil {
		t.Fatal("accepted a short public key")
	}
}

func TestHandshakeKeyEncryptsEndToEnd(t *testing.T) {
	cPriv, cPub, _ := GenerateKeyPair()
	gPriv, gPub, _ := GenerateKeyPair()
	id := sessionID(0x66)

	cKey, _ := DeriveSessionKey(cPriv, gPub, id)
	gKey, _ := DeriveSessionKey(gPriv, cPub, id)

	client, err := NewSession(id, cKey, RoleInitiator, 64)
	if err != nil {
		t.Fatalf("client session: %v", err)
	}
	gateway, err := NewSession(id, gKey, RoleResponder, 64)
	if err != nil {
		t.Fatalf("gateway session: %v", err)
	}

	wire, err := client.Outbound([]byte("handshake-keyed payload"))
	if err != nil {
		t.Fatalf("outbound: %v", err)
	}
	got, first, err := gateway.Inbound(wire, "wifi")
	if err != nil || !first {
		t.Fatalf("inbound first=%v err=%v", first, err)
	}
	if !bytes.Equal(got, []byte("handshake-keyed payload")) {
		t.Fatalf("payload = %q", got)
	}
}
