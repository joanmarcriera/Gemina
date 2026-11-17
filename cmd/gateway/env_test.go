package main

import (
	"bytes"
	"log/slog"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestEnvOr(t *testing.T) {
	const key = "GEMINA_TEST_ENV_OR"
	tests := []struct {
		name, value, want string
	}{
		{name: "unset uses fallback", value: "", want: "fb"},
		{name: "set wins", value: "x", want: "x"},
		{name: "whitespace is a value", value: " ", want: " "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(key, tc.value)
			if got := envOr(key, "fb"); got != tc.want {
				t.Fatalf("envOr = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestEnvInt pins integer env parsing: only a positive base-10 int overrides
// the fallback; anything else is ignored with a warning naming the key.
func TestEnvInt(t *testing.T) {
	const key = "GEMINA_TEST_ENV_INT"
	tests := []struct {
		value    string
		want     int
		wantWarn bool
	}{
		{value: "", want: 42},
		{value: "1", want: 1},
		{value: "4194304", want: 4194304},
		{value: "0", want: 42, wantWarn: true},
		{value: "-5", want: 42, wantWarn: true},
		{value: "abc", want: 42, wantWarn: true},
		{value: "12k", want: 42, wantWarn: true},
		{value: " 7", want: 42, wantWarn: true},
		{value: "1.5", want: 42, wantWarn: true},
		{value: "99999999999999999999", want: 42, wantWarn: true},
	}
	for _, tc := range tests {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv(key, tc.value)
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			if got := envInt(logger, key, 42); got != tc.want {
				t.Fatalf("envInt(%q) = %d, want %d", tc.value, got, tc.want)
			}
			warned := strings.Contains(buf.String(), "ignoring invalid env value") && strings.Contains(buf.String(), key)
			if warned != tc.wantWarn {
				t.Fatalf("warned = %v, want %v; log: %s", warned, tc.wantWarn, buf.String())
			}
		})
	}
}

// TestConnSinkSendTo pins that the return-path sink writes the framed datagram
// verbatim to the requested endpoint and surfaces a closed-socket error.
func TestConnSinkSendTo(t *testing.T) {
	recv, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen recv: %v", err)
	}
	defer recv.Close()
	send, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen send: %v", err)
	}

	dst := recv.LocalAddr().(*net.UDPAddr).AddrPort()
	sink := connSink{conn: send}
	want := []byte("CVD1-framed-return")
	if err := sink.SendTo(want, dst); err != nil {
		t.Fatalf("SendTo: %v", err)
	}
	_ = recv.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 64)
	n, from, err := recv.ReadFrom(buf)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if !bytes.Equal(buf[:n], want) {
		t.Fatalf("received %q, want %q", buf[:n], want)
	}
	if from.(*net.UDPAddr).AddrPort() != send.LocalAddr().(*net.UDPAddr).AddrPort() {
		t.Fatalf("datagram came from unexpected socket")
	}

	_ = send.Close()
	if err := sink.SendTo(want, netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), dst.Port())); err == nil {
		t.Fatal("SendTo on closed socket succeeded")
	}
}
