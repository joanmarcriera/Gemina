package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

func testProbe(t *testing.T, session byte, number protocol.PacketNumber, path protocol.PathTag) []byte {
	t.Helper()
	var s protocol.SessionID
	for i := range s {
		s[i] = session
	}
	wire, err := protocol.ProbePacket{ID: protocol.PacketID{Session: s, Number: number}, Path: path}.MarshalBinary()
	if err != nil {
		t.Fatalf("build probe: %v", err)
	}
	return wire
}

func newTestServer(t *testing.T, buf *bytes.Buffer) *Server {
	t.Helper()
	// Debug level so per-packet decision records are emitted for assertions
	// (in production these are Debug; Info carries periodic summaries).
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	server, err := NewServer(64, logger)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return server
}

func TestHandleFirstCopyThenDuplicate(t *testing.T) {
	server := newTestServer(t, &bytes.Buffer{})
	probeA := testProbe(t, 0x11, 7, protocol.PathWiFi)
	// Same identity, different path tag: the second copy is a duplicate.
	probeB := testProbe(t, 0x11, 7, protocol.PathAndroidUSBTether)

	if got := server.Handle(probeA); got.Decision != DecisionFirstCopy {
		t.Fatalf("first decision = %v, want first-copy", got.Decision)
	}
	dup := server.Handle(probeB)
	if dup.Decision != DecisionDuplicate {
		t.Fatalf("second decision = %v, want duplicate", dup.Decision)
	}
	if dup.FirstPath != protocol.PathWiFi {
		t.Fatalf("duplicate first path = %v, want wi-fi", dup.FirstPath)
	}

	stats := server.Stats()
	if stats.FirstCopies != 1 || stats.Duplicates != 1 || stats.Rejected != 0 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestHandleRejectsMalformedWithoutCrashing(t *testing.T) {
	server := newTestServer(t, &bytes.Buffer{})

	for _, bad := range [][]byte{nil, []byte("garbage"), bytes.Repeat([]byte{0}, protocol.ProbeWireSize)} {
		if got := server.Handle(bad); got.Decision != DecisionRejected {
			t.Fatalf("Handle(%q) decision = %v, want rejected", bad, got.Decision)
		}
	}
	if s := server.Stats(); s.Rejected != 3 || s.FirstCopies != 0 {
		t.Fatalf("stats = %+v, want 3 rejected", s)
	}
}

func TestServeDedupsOverLoopbackWithoutLeakingSource(t *testing.T) {
	var logs bytes.Buffer
	server := newTestServer(t, &logs)

	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, conn) }()

	client, err := net.Dial("udp", conn.LocalAddr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	// One logical packet sent over two paths (duplicate), plus a distinct one.
	send := [][]byte{
		testProbe(t, 0x22, 1, protocol.PathWiFi),
		testProbe(t, 0x22, 1, protocol.PathAndroidUSBTether),
		testProbe(t, 0x22, 2, protocol.PathWiFi),
	}
	for _, p := range send {
		if _, err := client.Write(p); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	// Wait until the server has processed all three (bounded, no fixed sleep).
	waitFor(t, func() bool {
		s := server.Stats()
		return s.FirstCopies+s.Duplicates == 3
	})

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	stats := server.Stats()
	if stats.FirstCopies != 2 || stats.Duplicates != 1 {
		t.Fatalf("stats = %+v, want 2 first-copies, 1 duplicate", stats)
	}

	// Redaction: the client source address must never appear in the logs.
	host, _, _ := net.SplitHostPort(client.LocalAddr().String())
	if bytes.Contains(logs.Bytes(), []byte(host)) {
		t.Fatalf("source host %q leaked into logs:\n%s", host, logs.String())
	}
	// And the logs must be structured records carrying a decision.
	if !bytes.Contains(logs.Bytes(), []byte(`"decision"`)) {
		t.Fatalf("logs missing decision field:\n%s", logs.String())
	}
	// Confirm log lines are valid JSON objects.
	for _, line := range bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatalf("log line not JSON: %s", line)
		}
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}
