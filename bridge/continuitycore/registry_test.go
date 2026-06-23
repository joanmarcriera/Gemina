package main

import (
	"bytes"
	"testing"

	"continuity-vpn/pkg/clientcore"
)

// testKey is a fixed 32-byte key; the two endpoints in a session share it.
func testKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i + 1)
	}
	return k
}

// testID is a fixed, non-zero 16-byte session identity.
func testID() []byte {
	id := make([]byte, 16)
	for i := range id {
		id[i] = byte(0xA0 + i)
	}
	return id
}

func TestCreateSessionRejectsBadArgs(t *testing.T) {
	if h := createSession(make([]byte, 15), testKey(), clientcore.RoleInitiator, 64); h != 0 {
		t.Fatalf("short session id: want handle 0, got %d", h)
	}
	if h := createSession(make([]byte, 16), testKey(), clientcore.RoleInitiator, 64); h != 0 {
		t.Fatalf("zero session id: want handle 0, got %d", h)
	}
	if h := createSession(testID(), make([]byte, 31), clientcore.RoleInitiator, 64); h != 0 {
		t.Fatalf("short key: want handle 0, got %d", h)
	}
}

func TestCreateSessionReturnsDistinctHandles(t *testing.T) {
	h1 := createSession(testID(), testKey(), clientcore.RoleInitiator, 64)
	h2 := createSession(testID(), testKey(), clientcore.RoleResponder, 64)
	if h1 == 0 || h2 == 0 {
		t.Fatalf("want non-zero handles, got %d and %d", h1, h2)
	}
	if h1 == h2 {
		t.Fatalf("want distinct handles, both were %d", h1)
	}
	t.Cleanup(func() { reg.remove(h1); reg.remove(h2) })
}

// TestRoundTripAndDuplicateSuppression drives a full initiator->responder
// exchange through the registry-level helpers: the payload must survive, the
// first copy must deliver, and a second identical wire must be suppressed.
func TestRoundTripAndDuplicateSuppression(t *testing.T) {
	id, key := testID(), testKey()
	client := createSession(id, key, clientcore.RoleInitiator, 64)
	server := createSession(id, key, clientcore.RoleResponder, 64)
	if client == 0 || server == 0 {
		t.Fatalf("session creation failed: client=%d server=%d", client, server)
	}
	t.Cleanup(func() { reg.remove(client); reg.remove(server) })

	payload := []byte("the quick brown fox jumps over the lazy dog")

	// Frame on the initiator.
	wire := make([]byte, 2048)
	n := outboundInto(client, payload, wire)
	if n <= 0 {
		t.Fatalf("outbound: want positive length, got %d", n)
	}
	wire = wire[:n]

	// First inbound on the responder: must deliver the exact payload.
	out := make([]byte, 2048)
	var deliver bool
	m := inboundInto(server, wire, "wifi", out, &deliver)
	if m < 0 {
		t.Fatalf("inbound (first): error code %d", m)
	}
	if !deliver {
		t.Fatalf("inbound (first): want deliver=true")
	}
	if !bytes.Equal(out[:m], payload) {
		t.Fatalf("payload corrupted: want %q, got %q", payload, out[:m])
	}

	// Same framed bytes arriving on a second path: must be suppressed.
	out2 := make([]byte, 2048)
	var deliver2 bool
	m2 := inboundInto(server, wire, "cellular", out2, &deliver2)
	if m2 < 0 {
		t.Fatalf("inbound (duplicate): error code %d", m2)
	}
	if deliver2 {
		t.Fatalf("inbound (duplicate): want deliver=false, payload should be dropped")
	}
}

func TestOutboundBadHandle(t *testing.T) {
	out := make([]byte, 64)
	if got := outboundInto(999999, []byte("x"), out); got != -errCodeBadHandle {
		t.Fatalf("want %d (bad handle), got %d", -errCodeBadHandle, got)
	}
}

func TestInboundBadHandle(t *testing.T) {
	out := make([]byte, 64)
	var deliver bool
	if got := inboundInto(999999, []byte("x"), "wifi", out, &deliver); got != -errCodeBadHandle {
		t.Fatalf("want %d (bad handle), got %d", -errCodeBadHandle, got)
	}
	if deliver {
		t.Fatalf("deliver must be false on error")
	}
}

func TestOutboundBufferTooSmall(t *testing.T) {
	h := createSession(testID(), testKey(), clientcore.RoleInitiator, 64)
	if h == 0 {
		t.Fatal("session creation failed")
	}
	t.Cleanup(func() { reg.remove(h) })

	// One byte cannot hold the framed header, never mind the ciphertext.
	tiny := make([]byte, 1)
	if got := outboundInto(h, []byte("hello"), tiny); got != -errCodeBufferSize {
		t.Fatalf("want %d (buffer too small), got %d", -errCodeBufferSize, got)
	}
}

func TestInboundBufferTooSmall(t *testing.T) {
	id, key := testID(), testKey()
	client := createSession(id, key, clientcore.RoleInitiator, 64)
	server := createSession(id, key, clientcore.RoleResponder, 64)
	if client == 0 || server == 0 {
		t.Fatal("session creation failed")
	}
	t.Cleanup(func() { reg.remove(client); reg.remove(server) })

	payload := []byte("a moderately sized payload that will not fit a tiny buffer")
	wire := make([]byte, 2048)
	n := outboundInto(client, payload, wire)
	if n <= 0 {
		t.Fatalf("outbound failed: %d", n)
	}

	tiny := make([]byte, len(payload)-1)
	var deliver bool
	if got := inboundInto(server, wire[:n], "wifi", tiny, &deliver); got != -errCodeBufferSize {
		t.Fatalf("want %d (buffer too small), got %d", -errCodeBufferSize, got)
	}
	if deliver {
		t.Fatalf("deliver must be false on error")
	}
}

func TestInboundCoreErrorOnGarbage(t *testing.T) {
	h := createSession(testID(), testKey(), clientcore.RoleResponder, 64)
	if h == 0 {
		t.Fatal("session creation failed")
	}
	t.Cleanup(func() { reg.remove(h) })

	out := make([]byte, 64)
	var deliver bool
	if got := inboundInto(h, []byte("not a valid frame"), "wifi", out, &deliver); got != -errCodeCore {
		t.Fatalf("want %d (core error), got %d", -errCodeCore, got)
	}
}

func TestRemoveHandle(t *testing.T) {
	h := createSession(testID(), testKey(), clientcore.RoleInitiator, 64)
	if h == 0 {
		t.Fatal("session creation failed")
	}
	reg.remove(h)
	out := make([]byte, 64)
	if got := outboundInto(h, []byte("x"), out); got != -errCodeBadHandle {
		t.Fatalf("after free, want %d (bad handle), got %d", -errCodeBadHandle, got)
	}
}
