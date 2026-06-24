// Command gateway runs the Stage-1 probe gateway: a UDP listener that
// deduplicates probe copies arriving over multiple client paths and logs each
// decision as redacted JSON. It is a feasibility server, not VPN transport.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"continuity-vpn/internal/entitlement"
	"continuity-vpn/internal/gateway"
	"continuity-vpn/internal/metrics"
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

	// "data" runs the real gateway (authenticated handshake + encrypted data plane
	// + admission); "probe" (default) runs the Stage-1 dedup probe server.
	if envOr("CONTINUITY_GATEWAY_MODE", "probe") == "data" {
		runDataGateway(logger, addr, capacity, readBuffer)
		return
	}

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

	// Optional Prometheus metrics endpoint. Off unless an address is configured,
	// so the default footprint is unchanged and self-hosters opt in.
	if metricsAddr := os.Getenv("CONTINUITY_GATEWAY_METRICS_ADDR"); metricsAddr != "" {
		startMetricsServer(ctx, metricsAddr, server.Metrics(), logger)
	}

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

// runDataGateway runs the real gateway: it loads (or creates) the Ed25519
// identity clients pin, builds the entitlement service (open for self-host,
// hosted for the paid tier), and serves the authenticated handshake + encrypted
// data plane, exposing the same redacted /metrics.
func runDataGateway(logger *slog.Logger, addr string, capacity, readBuffer int) {
	identityPath := envOr("CONTINUITY_GATEWAY_IDENTITY", "gateway-identity.key")
	priv, created, err := gateway.LoadOrCreateIdentity(identityPath)
	if err != nil {
		logger.Error("gateway identity", "path", identityPath, "error", err.Error())
		os.Exit(1)
	}
	pub := base64.StdEncoding.EncodeToString(priv.Public().(ed25519.PublicKey))
	logger.Info("gateway identity", "path", identityPath, "created", created, "public_key", pub)

	service := &entitlement.Service{Mode: entitlement.ModeOpen}
	if envOr("CONTINUITY_GATEWAY_TIER", "open") == "hosted" {
		key := os.Getenv("CONTINUITY_GATEWAY_ENTITLEMENT_KEY")
		if key == "" {
			logger.Error("hosted mode needs CONTINUITY_GATEWAY_ENTITLEMENT_KEY")
			os.Exit(1)
		}
		service = &entitlement.Service{Mode: entitlement.ModeHosted, Key: []byte(key)}
	}

	dg := gateway.NewDataGateway(priv, service, capacity, logger)

	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		logger.Error("listen", "addr", addr, "error", err.Error())
		os.Exit(1)
	}
	if udp, ok := conn.(*net.UDPConn); ok {
		_ = udp.SetReadBuffer(readBuffer)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if metricsAddr := os.Getenv("CONTINUITY_GATEWAY_METRICS_ADDR"); metricsAddr != "" {
		startMetricsServer(ctx, metricsAddr, dg.Metrics(), logger)
	}

	logger.Info("gateway listening", "addr", addr, "mode", "data", "tier", service.Mode)
	if err := dg.Serve(ctx, conn); err != nil {
		logger.Error("serve", "error", err.Error())
		os.Exit(1)
	}
}

// startMetricsServer serves GET /metrics (Prometheus text format) on addr until
// ctx is cancelled. The body is the gateway's redacted, coarse-token metrics.
func startMetricsServer(ctx context.Context, addr string, reg *metrics.Registry, logger *slog.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = io.WriteString(w, reg.Render())
	})
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	go func() {
		logger.Info("metrics listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics serve", "error", err.Error())
		}
	}()
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
