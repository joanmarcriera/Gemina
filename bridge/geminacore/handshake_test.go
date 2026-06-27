package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/joanmarcriera/gemina/pkg/clientcore"
)

// simulateGateway plays the gateway's half of the handshake using only clientcore
// primitives: it decodes the ClientHello, mints an ephemeral key, signs it under
// the long-term identity, and returns the ServerHello bytes plus the session key
// it derived. A correct client Complete must derive the same key.
func simulateGateway(t *testing.T, identityPriv ed25519.PrivateKey, clientHello []byte) (serverHello, gatewayKey []byte, id []byte) {
	t.Helper()
	sid, _, clientEph, _, err := clientcore.DecodeClientHello(clientHello)
	if err != nil {
		t.Fatalf("gateway could not decode ClientHello: %v", err)
	}
	gwPriv, gwPub, err := clientcore.GenerateKeyPair()
	if err != nil {
		t.Fatalf("gateway key pair: %v", err)
	}
	sig := clientcore.SignHandshake(identityPriv, gwPub, sid)
	hello, err := clientcore.EncodeServerHello(sid, gwPub, sig, [4]byte{})
	if err != nil {
		t.Fatalf("gateway EncodeServerHello: %v", err)
	}
	key, err := clientcore.DeriveSessionKey(gwPriv, clientEph, sid)
	if err != nil {
		t.Fatalf("gateway DeriveSessionKey: %v", err)
	}
	return hello, key, sid[:]
}

// TestHandshakeBeginCompleteEstablishesWorkingSession drives the full client
// handshake through the bridge helpers and proves the resulting session both
// matches the gateway's independently-derived key and carries traffic: an
// outbound frame from the client session decrypts cleanly on a responder session
// built from the gateway's key.
func TestHandshakeBeginCompleteEstablishesWorkingSession(t *testing.T) {
	gwPub, gwPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("identity: %v", err)
	}

	out := make([]byte, 4096)
	helloLen, hs := beginHandshake(gwPub, "valid-token", out)
	if helloLen <= 0 {
		t.Fatalf("beginHandshake: want positive hello length, got %d", helloLen)
	}
	if hs == 0 {
		t.Fatalf("beginHandshake: want non-zero handshake handle")
	}
	clientHello := out[:helloLen]

	serverHello, gatewayKey, id := simulateGateway(t, gwPriv, clientHello)

	client := completeHandshake(hs, serverHello, 64, nil)
	if client == 0 {
		t.Fatalf("completeHandshake: want non-zero session handle")
	}
	t.Cleanup(func() { reg.remove(client) })

	// Build the responder side from the gateway's derived key; if Complete derived
	// the same key, a client frame decrypts and delivers on the responder.
	server := createSession(id, gatewayKey, clientcore.RoleResponder, 64)
	if server == 0 {
		t.Fatalf("responder session creation failed (keys likely diverged)")
	}
	t.Cleanup(func() { reg.remove(server) })

	payload := []byte("post-handshake traffic")
	wire := make([]byte, 2048)
	n := outboundInto(client, payload, wire)
	if n <= 0 {
		t.Fatalf("outbound on handshaken session: %d", n)
	}
	recovered := make([]byte, 2048)
	var deliver bool
	m := inboundInto(server, wire[:n], "wifi", recovered, &deliver)
	if m < 0 || !deliver {
		t.Fatalf("inbound on responder: code %d deliver %v (keys diverged?)", m, deliver)
	}
	if !bytes.Equal(recovered[:m], payload) {
		t.Fatalf("payload corrupted: want %q got %q", payload, recovered[:m])
	}
}

// TestCompleteHandshakeReturnsAssignedIP proves the bridge surfaces the
// gateway-assigned tunnel IPv4 (carried in-band in the ServerHello) into the
// caller's 4-byte out buffer.
func TestCompleteHandshakeReturnsAssignedIP(t *testing.T) {
	gwPub, gwPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("identity: %v", err)
	}

	out := make([]byte, 4096)
	helloLen, hs := beginHandshake(gwPub, "tok", out)
	if helloLen <= 0 || hs == 0 {
		t.Fatalf("beginHandshake failed: len %d handle %d", helloLen, hs)
	}

	// The gateway builds a ServerHello carrying a known assigned tunnel IP.
	sid, _, clientEph, _, err := clientcore.DecodeClientHello(out[:helloLen])
	if err != nil {
		t.Fatalf("decode client hello: %v", err)
	}
	gwEphPriv, gwEphPub, err := clientcore.GenerateKeyPair()
	if err != nil {
		t.Fatalf("gateway key pair: %v", err)
	}
	if _, err = clientcore.DeriveSessionKey(gwEphPriv, clientEph, sid); err != nil {
		t.Fatalf("gateway derive: %v", err)
	}
	sig := clientcore.SignHandshake(gwPriv, gwEphPub, sid)
	wantIP := [4]byte{10, 99, 0, 42}
	serverHello, err := clientcore.EncodeServerHello(sid, gwEphPub, sig, wantIP)
	if err != nil {
		t.Fatalf("encode server hello: %v", err)
	}

	gotIP := make([]byte, 4)
	handle := completeHandshake(hs, serverHello, 64, gotIP)
	if handle == 0 {
		t.Fatalf("completeHandshake returned 0")
	}
	t.Cleanup(func() { reg.remove(handle) })
	if !bytes.Equal(gotIP, wantIP[:]) {
		t.Fatalf("assigned IP out: got %v want %v", gotIP, wantIP)
	}
}

func TestBeginHandshakeRejectsBadIdentity(t *testing.T) {
	out := make([]byte, 4096)
	// A 31-byte identity is not a valid Ed25519 public key.
	if helloLen, hs := beginHandshake(make([]byte, 31), "tok", out); helloLen != -errCodeBadArgs || hs != 0 {
		t.Fatalf("bad identity: want (%d, 0), got (%d, %d)", -errCodeBadArgs, helloLen, hs)
	}
}

func TestBeginHandshakeBufferTooSmall(t *testing.T) {
	gwPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	tiny := make([]byte, 8) // cannot hold a ClientHello
	if helloLen, hs := beginHandshake(gwPub, "tok", tiny); helloLen != -errCodeBufferSize || hs != 0 {
		t.Fatalf("tiny buffer: want (%d, 0), got (%d, %d)", -errCodeBufferSize, helloLen, hs)
	}
}

func TestCompleteHandshakeBadHandle(t *testing.T) {
	if h := completeHandshake(999999, make([]byte, serverHelloSize), 64, nil); h != 0 {
		t.Fatalf("unknown handshake handle: want 0, got %d", h)
	}
}

// serverHelloSize mirrors clientcore's ServerHello frame length so the bad-handle
// test does not silently encode a stale wire size. The handle check fires before
// the wire bytes are parsed, so the exact value only documents intent.
const serverHelloSize = 122

// TestCancelHandshakeConsumesHandle proves cancelHandshake discards an in-flight
// handshake so it cannot leak and the handle cannot be reused: a subsequent
// completeHandshake on the cancelled handle returns 0.
func TestCancelHandshakeConsumesHandle(t *testing.T) {
	gwPub, gwPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	out := make([]byte, 4096)
	helloLen, hs := beginHandshake(gwPub, "tok", out)
	if helloLen <= 0 || hs == 0 {
		t.Fatalf("beginHandshake failed: len %d handle %d", helloLen, hs)
	}

	cancelHandshake(hs)

	// The handle is consumed: completing it now (even with a valid ServerHello)
	// must fail, proving the in-flight state was freed and cannot be reused.
	serverHello, _, _ := simulateGateway(t, gwPriv, out[:helloLen])
	if h := completeHandshake(hs, serverHello, 64, nil); h != 0 {
		reg.remove(h)
		t.Fatalf("completing a cancelled handshake returned a session: %d", h)
	}
}

// TestCompleteHandshakeIsOneShot proves a handshake handle is consumed by the
// first Complete call (success or failure), so a captured handle cannot be reused
// to mint extra sessions.
func TestCompleteHandshakeIsOneShot(t *testing.T) {
	gwPub, gwPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	out := make([]byte, 4096)
	helloLen, hs := beginHandshake(gwPub, "tok", out)
	if helloLen <= 0 || hs == 0 {
		t.Fatalf("beginHandshake failed: len %d handle %d", helloLen, hs)
	}
	serverHello, _, _ := simulateGateway(t, gwPriv, out[:helloLen])

	first := completeHandshake(hs, serverHello, 64, nil)
	if first == 0 {
		t.Fatalf("first complete should succeed")
	}
	t.Cleanup(func() { reg.remove(first) })

	if second := completeHandshake(hs, serverHello, 64, nil); second != 0 {
		t.Fatalf("handshake handle should be one-shot: second complete returned %d", second)
		reg.remove(second)
	}
}

// TestCompleteHandshakeRejectsForgedSignature proves the pinned-identity check
// fires: a ServerHello signed by a DIFFERENT identity must not yield a session,
// and the handle is still consumed.
func TestCompleteHandshakeRejectsForgedSignature(t *testing.T) {
	gwPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	_, attackerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("attacker identity: %v", err)
	}

	out := make([]byte, 4096)
	helloLen, hs := beginHandshake(gwPub, "tok", out)
	if helloLen <= 0 || hs == 0 {
		t.Fatalf("beginHandshake failed: len %d handle %d", helloLen, hs)
	}
	// The attacker signs with its own identity, not the pinned gwPub.
	forged, _, _ := simulateGateway(t, attackerPriv, out[:helloLen])

	if h := completeHandshake(hs, forged, 64, nil); h != 0 {
		reg.remove(h)
		t.Fatalf("forged signature must be rejected, got session handle %d", h)
	}
}
