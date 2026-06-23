// Command gateway runs the Stage-1 probe gateway: a UDP listener that
// deduplicates probe copies arriving over multiple client paths and logs each
// decision as redacted JSON. It is a feasibility server, not VPN transport.
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"continuity-vpn/internal/gateway"
)

const (
	defaultAddr       = ":51820"
	defaultCapacity   = 8192
	defaultReadBuffer = 4 << 20 // 4 MiB: tolerate bursts of duplicate probes
)

func main() {
	level := slog.LevelInfo
	if envOr("CONTINUITY_GATEWAY_LOG_LEVEL", "info") == "debug" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	addr := envOr("CONTINUITY_GATEWAY_ADDR", defaultAddr)
	capacity := envInt(logger, "CONTINUITY_GATEWAY_DEDUP_CAPACITY", defaultCapacity)
	readBuffer := envInt(logger, "CONTINUITY_GATEWAY_READ_BUFFER", defaultReadBuffer)

	server, err := gateway.NewServer(capacity, logger)
	if err != nil {
		logger.Error("create gateway", "error", err.Error())
		os.Exit(1)
	}

	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		logger.Error("listen", "addr", addr, "error", err.Error())
		os.Exit(1)
	}

	// Enlarge the socket receive buffer so bursts are not silently dropped under
	// load. Non-fatal: the kernel may clamp to a lower maximum.
	if udp, ok := conn.(*net.UDPConn); ok {
		if err := udp.SetReadBuffer(readBuffer); err != nil {
			logger.Warn("set read buffer", "bytes", readBuffer, "error", err.Error())
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("gateway listening", "addr", addr, "dedup_capacity", capacity, "stage", "stage-1-probe")
	if err := server.Serve(ctx, conn); err != nil {
		logger.Error("serve", "error", err.Error())
		os.Exit(1)
	}

	stats := server.Stats()
	logger.Info("gateway stopped",
		"first_copies", stats.FirstCopies,
		"duplicates", stats.Duplicates,
		"rejected", stats.Rejected,
	)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(logger *slog.Logger, key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		logger.Warn("ignoring invalid env value", "key", key, "value", v)
		return fallback
	}
	return n
}
