package clientcore

import (
	"bytes"
	"testing"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

func sessionID(b byte) protocol.SessionID {
	var id protocol.SessionID
	copy(id[:], bytes.Repeat([]byte{b}, protocol.SessionIDSize))
	return id
}

// newPair returns a client (initiator) and gateway (responder) sharing a key, as
// the two ends of one session.
func newPair(t *testing.T) (client, gateway *Session) {
	t.Helper()
	key := testKey()
	id := sessionID(0xAB)
	client, err := NewSession(id, key, RoleInitiator, 1024)
	if err != nil {
		t.Fatalf("new client session: %v", err)
	}
	gateway, err = NewSession(id, key, RoleResponder, 1024)
	if err != nil {
		t.Fatalf("new gateway session: %v", err)
	}
	return client, gateway
}

func TestOutboundIncrementsIdentityAndEncrypts(t *testing.T) {
	client, gateway := newPair(t)
	payload := []byte("the quick brown fox")

	first, err := client.Outbound(payload)
	if err != nil {
		t.Fatalf("outbound 1: %v", err)
	}
	second, err := client.Outbound(payload)
	if err != nil {
		t.Fatalf("outbound 2: %v", err)
	}

	// Distinct identities -> distinct ciphertext, and the payload is not in clear.
	if bytes.Equal(first, second) {
		t.Fatal("consecutive outbound packets share wire bytes; numbers must differ")
	}
	if bytes.Contains(first, payload) {
		t.Fatal("payload appears in cleartext on the wire")
	}

	got, deliver, err := gateway.Inbound(first, "wifi")
	if err != nil {
		t.Fatalf("inbound: %v", err)
	}
	if !deliver || !bytes.Equal(got, payload) {
		t.Fatalf("round-trip deliver=%v payload=%q want %q", deliver, got, payload)
	}
}

func TestInboundDeduplicatesSameLogicalPacketAcrossPaths(t *testing.T) {
	client, gateway := newPair(t)

	wire, err := client.Outbound([]byte("hello"))
	if err != nil {
		t.Fatalf("outbound: %v", err)
	}

	payload, first, err := gateway.Inbound(wire, "wifi")
	if err != nil {
		t.Fatalf("inbound path A: %v", err)
	}
	if !first || !bytes.Equal(payload, []byte("hello")) {
		t.Fatalf("first copy: first=%v payload=%q", first, payload)
	}

	_, firstAgain, err := gateway.Inbound(wire, "usb")
	if err != nil {
		t.Fatalf("inbound path B: %v", err)
	}
	if firstAgain {
		t.Fatal("the same logical packet arriving on the second path must be a duplicate")
	}
}

func TestInboundDistinctPacketsAreEachDelivered(t *testing.T) {
	client, gateway := newPair(t)

	w1, _ := client.Outbound([]byte("one"))
	w2, _ := client.Outbound([]byte("two"))

	_, f1, err := gateway.Inbound(w1, "wifi")
	if err != nil || !f1 {
		t.Fatalf("packet 1 first=%v err=%v", f1, err)
	}
	_, f2, err := gateway.Inbound(w2, "wifi")
	if err != nil || !f2 {
		t.Fatalf("packet 2 first=%v err=%v", f2, err)
	}
}

func TestInboundRejectsWrongDirection(t *testing.T) {
	client, _ := newPair(t)
	// A second client (initiator) cannot accept another initiator's packet: the
	// direction is wrong, which also guards against reflected traffic.
	other, err := NewSession(sessionID(0xAB), testKey(), RoleInitiator, 1024)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	wire, _ := client.Outbound([]byte("x"))
	if _, _, err := other.Inbound(wire, "wifi"); err == nil {
		t.Fatal("an initiator accepted another initiator's packet (wrong direction)")
	}
}

func TestInboundRejectsForgedPacketWithoutTouchingDedup(t *testing.T) {
	client, gateway := newPair(t)

	good, _ := client.Outbound([]byte("real"))
	// Forge a datagram with the same identity but garbage ciphertext.
	forged := append([]byte(nil), good...)
	forged[len(forged)-1] ^= 0xFF

	if _, _, err := gateway.Inbound(forged, "attacker"); err == nil {
		t.Fatal("gateway accepted a forged packet")
	}
	// The real packet must still be delivered as a first copy: the forgery did
	// not poison the dedup window.
	_, first, err := gateway.Inbound(good, "wifi")
	if err != nil || !first {
		t.Fatalf("real packet after forgery: first=%v err=%v", first, err)
	}
}

func TestInboundConcurrentPathsDeliverEachPacketOnce(t *testing.T) {
	client, gateway := newPair(t)

	const n = 500
	wires := make([][]byte, n)
	for i := range wires {
		w, err := client.Outbound([]byte{byte(i), byte(i >> 8)})
		if err != nil {
			t.Fatalf("outbound %d: %v", i, err)
		}
		wires[i] = w
	}

	firsts := make(chan int, 2)
	recv := func(path string) {
		count := 0
		for _, w := range wires {
			_, first, err := gateway.Inbound(w, path)
			if err != nil {
				t.Errorf("inbound on %s: %v", path, err)
				return
			}
			if first {
				count++
			}
		}
		firsts <- count
	}
	go recv("wifi")
	go recv("usb")
	total := <-firsts + <-firsts

	if total != n {
		t.Fatalf("delivered %d first copies across both paths, want exactly %d", total, n)
	}
}

func TestInboundRejectsMalformedWire(t *testing.T) {
	_, gateway := newPair(t)

	if _, _, err := gateway.Inbound([]byte("too short"), "wifi"); err == nil {
		t.Fatal("expected error for short datagram")
	}
	bad := make([]byte, dataHeaderSize+16)
	copy(bad, []byte("XXXX")) // wrong magic
	if _, _, err := gateway.Inbound(bad, "wifi"); err == nil {
		t.Fatal("expected error for bad magic")
	}
}

func TestOutboundRejectsOversizePayload(t *testing.T) {
	client, _ := newPair(t)
	if _, err := client.Outbound(make([]byte, maxPayload+1)); err == nil {
		t.Fatal("expected error for oversize payload")
	}
}

func TestNewSessionRejectsBadKey(t *testing.T) {
	if _, err := NewSession(sessionID(1), []byte("short"), RoleInitiator, 16); err == nil {
		t.Fatal("expected error for short key")
	}
}
