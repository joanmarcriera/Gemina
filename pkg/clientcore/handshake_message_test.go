package clientcore

import (
	"bytes"
	"testing"
)

func TestClientHelloRoundTrip(t *testing.T) {
	id := sessionID(0x3C)
	_, ephPub, _ := GenerateKeyPair()
	token := "an.entitlement.token"

	hello, err := EncodeClientHello(id, 1717171717, ephPub, token)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	gotID, gotTS, gotEph, gotToken, err := DecodeClientHello(hello)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if gotID != id || gotTS != 1717171717 || !bytes.Equal(gotEph, ephPub) || gotToken != token {
		t.Fatalf("round-trip mismatch: id=%v ts=%d eph=%v token=%q", gotID == id, gotTS, bytes.Equal(gotEph, ephPub), gotToken)
	}
}

func TestServerHelloRoundTrip(t *testing.T) {
	id := sessionID(0x3D)
	_, ephPub, _ := GenerateKeyPair()
	sig := bytes.Repeat([]byte{0x5A}, 64)
	assignedIP := [4]byte{10, 99, 0, 2}

	hello, err := EncodeServerHello(id, ephPub, sig, assignedIP)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	gotID, gotEph, gotSig, gotIP, err := DecodeServerHello(hello)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if gotID != id || !bytes.Equal(gotEph, ephPub) || !bytes.Equal(gotSig, sig) {
		t.Fatal("server hello round-trip mismatch")
	}
	if gotIP != assignedIP {
		t.Fatalf("assigned IP round-trip mismatch: got %v want %v", gotIP, assignedIP)
	}
}

// A zero AssignedIPv4 means "unassigned" (e.g. the gateway exit is off); it must
// still round-trip cleanly and be distinguishable from a real lease.
func TestServerHelloUnassignedIPRoundTrips(t *testing.T) {
	id := sessionID(0x3E)
	_, ephPub, _ := GenerateKeyPair()
	sig := bytes.Repeat([]byte{0x5A}, 64)

	hello, err := EncodeServerHello(id, ephPub, sig, [4]byte{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, _, _, gotIP, err := DecodeServerHello(hello)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if gotIP != ([4]byte{}) {
		t.Fatalf("unassigned IP should be zero, got %v", gotIP)
	}
}

func TestDecodeRejectsMalformed(t *testing.T) {
	if _, _, _, _, err := DecodeClientHello([]byte("short")); err == nil {
		t.Fatal("accepted short client hello")
	}
	if _, _, _, _, err := DecodeServerHello([]byte("short")); err == nil {
		t.Fatal("accepted short server hello")
	}
	id := sessionID(1)
	_, eph, _ := GenerateKeyPair()
	good, _ := EncodeClientHello(id, 0, eph, "t")
	bad := append([]byte(nil), good...)
	bad[0] ^= 0xFF // corrupt magic
	if _, _, _, _, err := DecodeClientHello(bad); err == nil {
		t.Fatal("accepted bad magic")
	}
}

// The client side: begin a handshake, have a (simulated) gateway answer, and
// complete into a working session whose key matches the gateway's.
func TestClientHandshakeCompletesAndKeysMatch(t *testing.T) {
	idPriv, idPub, _ := GenerateIdentity()

	hello, hs, err := BeginClientHandshake(idPub, "token")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	id, _, clientEph, _, err := DecodeClientHello(hello)
	if err != nil {
		t.Fatalf("decode hello: %v", err)
	}

	// Simulate the gateway: derive the shared key, sign its ephemeral key.
	gwEphPriv, gwEphPub, _ := GenerateKeyPair()
	gwKey, err := DeriveSessionKey(gwEphPriv, clientEph, id)
	if err != nil {
		t.Fatalf("gateway derive: %v", err)
	}
	sig := SignHandshake(idPriv, gwEphPub, id)
	serverHello, _ := EncodeServerHello(id, gwEphPub, sig, [4]byte{})

	session, err := hs.Complete(serverHello, 64)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	// Prove the keys match: a responder built with the gateway's key decrypts the
	// client session's traffic.
	gateway, err := NewSession(id, gwKey, RoleResponder, 64)
	if err != nil {
		t.Fatalf("gateway session: %v", err)
	}
	wire, _ := session.Outbound([]byte("after handshake"))
	payload, first, err := gateway.Inbound(wire, "wifi")
	if err != nil || !first || !bytes.Equal(payload, []byte("after handshake")) {
		t.Fatalf("post-handshake traffic failed: first=%v err=%v", first, err)
	}
}

func TestClientHandshakeRejectsForgedGateway(t *testing.T) {
	_, idPub, _ := GenerateIdentity() // client pins this real identity
	attackerPriv, _, _ := GenerateIdentity()

	hello, hs, _ := BeginClientHandshake(idPub, "token")
	id, _, clientEph, _, _ := DecodeClientHello(hello)

	gwEphPriv, gwEphPub, _ := GenerateKeyPair()
	_ = gwEphPriv
	_, _ = DeriveSessionKey(gwEphPriv, clientEph, id)
	forgedSig := SignHandshake(attackerPriv, gwEphPub, id) // signed by the attacker
	serverHello, _ := EncodeServerHello(id, gwEphPub, forgedSig, [4]byte{})

	if _, err := hs.Complete(serverHello, 64); err == nil {
		t.Fatal("client completed a handshake with a forged gateway signature")
	}
}

func TestClientHandshakeRejectsSessionMismatch(t *testing.T) {
	idPriv, idPub, _ := GenerateIdentity()
	hello, hs, _ := BeginClientHandshake(idPub, "token")
	_, _, clientEph, _, _ := DecodeClientHello(hello)

	other := sessionID(0xEE) // not the session the client started
	gwEphPriv, gwEphPub, _ := GenerateKeyPair()
	_, _ = DeriveSessionKey(gwEphPriv, clientEph, other)
	sig := SignHandshake(idPriv, gwEphPub, other)
	serverHello, _ := EncodeServerHello(other, gwEphPub, sig, [4]byte{})

	if _, err := hs.Complete(serverHello, 64); err == nil {
		t.Fatal("client accepted a server hello for a different session")
	}
}
