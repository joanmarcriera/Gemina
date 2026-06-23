package gateway

import (
	"io"
	"log/slog"
	"testing"

	"continuity-vpn/internal/protocol"
)

func benchProbeWires(n int) [][]byte {
	wires := make([][]byte, n)
	var session protocol.SessionID
	for i := range session {
		session[i] = 0xc3
	}
	for i := 0; i < n; i++ {
		w, _ := protocol.ProbePacket{
			ID:   protocol.PacketID{Session: session, Number: protocol.PacketNumber(i + 1)},
			Path: protocol.PathWiFi,
		}.MarshalBinary()
		wires[i] = w
	}
	return wires
}

// BenchmarkHandleNoLog measures the decode + dedup + counter hot path with
// logging disabled (the discard handler), i.e. the irreducible per-packet cost.
func BenchmarkHandleNoLog(b *testing.B) {
	const distinct = 4096
	wires := benchProbeWires(distinct)
	server, _ := NewServer(distinct, nil) // nil logger -> discard

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.Handle(wires[i%distinct])
	}
}

// BenchmarkHandleInfoLevel measures the default production mode: a JSON logger
// at Info level, where per-packet logging is suppressed (guarded at Debug), so
// the hot path stays close to the no-log cost.
func BenchmarkHandleInfoLevel(b *testing.B) {
	const distinct = 4096
	wires := benchProbeWires(distinct)
	logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	server, _ := NewServer(distinct, logger)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.Handle(wires[i%distinct])
	}
}

// BenchmarkHandleDebugLevel measures full per-packet structured logging (Debug
// enabled), i.e. diagnostic mode.
func BenchmarkHandleDebugLevel(b *testing.B) {
	const distinct = 4096
	wires := benchProbeWires(distinct)
	logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	server, _ := NewServer(distinct, logger)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.Handle(wires[i%distinct])
	}
}
