package clientcore

import (
	"bytes"
	"testing"

	"continuity-vpn/internal/protocol"
)

func testSession(t *testing.T) *Session {
	t.Helper()
	id, err := protocol.NewSessionID(bytes.Repeat([]byte{0xAB}, protocol.SessionIDSize))
	if err != nil {
		t.Fatalf("new session id: %v", err)
	}
	s, err := NewSession(id, 1024)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	return s
}

func TestOutboundFramesPayloadWithIncrementingIdentity(t *testing.T) {
	s := testSession(t)
	payload := []byte("the quick brown fox")

	first, err := s.Outbound(payload)
	if err != nil {
		t.Fatalf("outbound 1: %v", err)
	}
	second, err := s.Outbound(payload)
	if err != nil {
		t.Fatalf("outbound 2: %v", err)
	}

	// Each outbound packet must carry a distinct identity (so duplicates of the
	// SAME logical packet — not consecutive packets — are what dedups).
	if bytes.Equal(first, second) {
		t.Fatal("consecutive outbound packets share wire bytes; numbers must differ")
	}

	gotPayload, _, err := decodeData(first)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(gotPayload, payload) {
		t.Fatalf("payload round-trip = %q, want %q", gotPayload, payload)
	}
}

func TestInboundDeduplicatesSameLogicalPacketAcrossPaths(t *testing.T) {
	sender := testSession(t)
	receiver := testSession(t)

	// One logical packet, framed once, sent over two paths (identical bytes).
	wire, err := sender.Outbound([]byte("hello"))
	if err != nil {
		t.Fatalf("outbound: %v", err)
	}

	payload, first, err := receiver.Inbound(wire, "wifi")
	if err != nil {
		t.Fatalf("inbound path A: %v", err)
	}
	if !first {
		t.Fatal("first copy must be reported as first")
	}
	if !bytes.Equal(payload, []byte("hello")) {
		t.Fatalf("payload = %q", payload)
	}

	_, firstAgain, err := receiver.Inbound(wire, "usb")
	if err != nil {
		t.Fatalf("inbound path B: %v", err)
	}
	if firstAgain {
		t.Fatal("the same logical packet arriving on the second path must be a duplicate")
	}
}

func TestInboundDistinctPacketsAreEachDelivered(t *testing.T) {
	sender := testSession(t)
	receiver := testSession(t)

	w1, _ := sender.Outbound([]byte("one"))
	w2, _ := sender.Outbound([]byte("two"))

	_, f1, err := receiver.Inbound(w1, "wifi")
	if err != nil || !f1 {
		t.Fatalf("packet 1 first=%v err=%v", f1, err)
	}
	_, f2, err := receiver.Inbound(w2, "wifi")
	if err != nil || !f2 {
		t.Fatalf("packet 2 first=%v err=%v", f2, err)
	}
}

func TestInboundConcurrentPathsDeliverEachPacketOnce(t *testing.T) {
	sender := testSession(t)
	receiver := testSession(t)

	const n = 500
	wires := make([][]byte, n)
	for i := range wires {
		w, err := sender.Outbound([]byte{byte(i), byte(i >> 8)})
		if err != nil {
			t.Fatalf("outbound %d: %v", i, err)
		}
		wires[i] = w
	}

	// Two receive goroutines (Wi-Fi + USB) feed every packet over both paths
	// concurrently. Exactly n must be reported as first copies.
	firsts := make(chan int, 2)
	recv := func(path string) {
		count := 0
		for _, w := range wires {
			_, first, err := receiver.Inbound(w, path)
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
	receiver := testSession(t)

	if _, _, err := receiver.Inbound([]byte("too short"), "wifi"); err == nil {
		t.Fatal("expected error for short datagram")
	}
	bad := make([]byte, dataHeaderSize+1)
	copy(bad, []byte("XXXX")) // wrong magic
	if _, _, err := receiver.Inbound(bad, "wifi"); err == nil {
		t.Fatal("expected error for bad magic")
	}
}

func TestOutboundRejectsOversizePayload(t *testing.T) {
	s := testSession(t)
	if _, err := s.Outbound(make([]byte, maxPayload+1)); err == nil {
		t.Fatal("expected error for oversize payload")
	}
}
