package main

import (
	"bytes"
	"log/slog"
	"net"
	"strings"
	"testing"
)

// TestSetReadBufferLogsFailure pins that a failed SetReadBuffer is logged (the
// data gateway used to discard it silently while probe mode warned).
func TestSetReadBufferLogsFailure(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("no loopback UDP: %v", err)
	}
	_ = conn.Close() // a closed socket makes SetReadBuffer fail deterministically

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	setReadBuffer(logger, conn, 1<<20)

	if !strings.Contains(buf.String(), "set read buffer") {
		t.Fatalf("SetReadBuffer failure not logged; log = %q", buf.String())
	}
}

// TestEnvChoiceWarnsOnUnrecognisedValue pins that a mistyped mode/exit switch is
// logged instead of silently falling back (GEMINA_GATEWAY_EXIT=true used to
// leave the exit path off with no trace).
func TestEnvChoiceWarnsOnUnrecognisedValue(t *testing.T) {
	tests := []struct {
		value    string
		want     string
		wantWarn bool
	}{
		{value: "", want: "off"},
		{value: "off", want: "off"},
		{value: "on", want: "on"},
		{value: "true", want: "off", wantWarn: true},
		{value: "ON", want: "off", wantWarn: true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("GEMINA_TEST_CHOICE", tt.value)
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			got := envChoice(logger, "GEMINA_TEST_CHOICE", "off", "on")
			if got != tt.want {
				t.Errorf("envChoice = %q, want %q", got, tt.want)
			}
			if warned := strings.Contains(buf.String(), "unrecognised"); warned != tt.wantWarn {
				t.Errorf("warned = %v, want %v; log = %q", warned, tt.wantWarn, buf.String())
			}
		})
	}
}
