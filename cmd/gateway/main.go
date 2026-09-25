// Command gateway runs the Stage-1 probe gateway: a UDP listener that
// deduplicates probe copies arriving over multiple client paths and logs each
// decision as redacted JSON. It is a feasibility server, not VPN transport.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joanmarcriera/gemina/internal/entitlement"
	"github.com/joanmarcriera/gemina/internal/exit"
	"github.com/joanmarcriera/gemina/internal/gateway"
	"github.com/joanmarcriera/gemina/internal/metrics"
)

const (
	defaultAddr       = ":51820"
	defaultCapacity   = 8192
	defaultReadBuffer = 4 << 20 // 4 MiB: tolerate bursts of duplicate probes
)

func main() {
	level := slog.LevelInfo
	if envOr("GEMINA_GATEWAY_LOG_LEVEL", "info") == "debug" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	addr := envOr("GEMINA_GATEWAY_ADDR", defaultAddr)
	capacity := envInt(logger, "GEMINA_GATEWAY_DEDUP_CAPACITY", defaultCapacity)
	readBuffer := envInt(logger, "GEMINA_GATEWAY_READ_BUFFER", defaultReadBuffer)

	// "data" runs the real gateway (authenticated handshake + encrypted data plane
	// + admission); "probe" (default) runs the Stage-1 dedup probe server.
	if envOr("GEMINA_GATEWAY_MODE", "probe") == "data" {
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
	if metricsAddr := os.Getenv("GEMINA_GATEWAY_METRICS_ADDR"); metricsAddr != "" {
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
//
// Privilege boundary (issue #5): only the UDP listener and the TUN device are
// opened while privileged. dropToRunAs then switches to the unprivileged uid
// BEFORE the identity file is touched, the metrics server starts, or any
// untrusted datagram is read.
func runDataGateway(logger *slog.Logger, addr string, capacity, readBuffer int) {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		logger.Error("listen", "addr", addr, "error", err.Error())
		os.Exit(1)
	}
	if udp, ok := conn.(*net.UDPConn); ok {
		_ = udp.SetReadBuffer(readBuffer)
	}

	// Optional internet exit path (Stage 2). Off by default so the data gateway
	// stays a decrypt+dedup endpoint unless the operator provisions a TUN.
	var exitDev *exitDevice
	if envOr("GEMINA_GATEWAY_EXIT", "off") == "on" {
		if exitDev, err = openExitDevice(logger); err != nil {
			logger.Error("enable exit path", "error", err.Error())
			os.Exit(1)
		}
	}

	dropToRunAs(logger)

	identityPath := envOr("GEMINA_GATEWAY_IDENTITY", "gateway-identity.key")
	priv, created, err := gateway.LoadOrCreateIdentity(identityPath)
	if err != nil {
		logger.Error("gateway identity", "path", identityPath, "error", err.Error())
		os.Exit(1)
	}
	pub := base64.StdEncoding.EncodeToString(priv.Public().(ed25519.PublicKey))
	logger.Info("gateway identity", "path", identityPath, "created", created, "public_key", pub)

	service := &entitlement.Service{Mode: entitlement.ModeOpen}
	if envOr("GEMINA_GATEWAY_TIER", "open") == "hosted" {
		key := os.Getenv("GEMINA_GATEWAY_ENTITLEMENT_KEY")
		if key == "" {
			logger.Error("hosted mode needs GEMINA_GATEWAY_ENTITLEMENT_KEY")
			os.Exit(1)
		}
		service = &entitlement.Service{Mode: entitlement.ModeHosted, Key: []byte(key)}
	}

	dg := gateway.NewDataGateway(priv, service, capacity, logger)
	if exitDev != nil {
		exitDev.enable(dg, conn, logger)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if metricsAddr := os.Getenv("GEMINA_GATEWAY_METRICS_ADDR"); metricsAddr != "" {
		startMetricsServer(ctx, metricsAddr, dg.Metrics(), logger)
	}

	logger.Info("gateway listening", "addr", addr, "mode", "data", "tier", service.Mode)
	if err := dg.Serve(ctx, conn); err != nil {
		logger.Error("serve", "error", err.Error())
		os.Exit(1)
	}
}

// dropToRunAs drops root to GEMINA_GATEWAY_RUN_AS (default 65532:65532, the
// distroless :nonroot uid) and exits the process if that cannot be done and
// verified — running the packet path as root is never a silent fallback. The
// explicit value "root" keeps root (logged loudly) for operators who must.
func dropToRunAs(logger *slog.Logger) {
	spec := envOr("GEMINA_GATEWAY_RUN_AS", defaultRunAs)
	if spec == runAsKeepRoot {
		if os.Geteuid() == 0 {
			logger.Warn("GEMINA_GATEWAY_RUN_AS=root: NOT dropping privileges; the packet path runs as root")
		}
		return
	}
	uid, gid, err := parseRunAs(spec)
	if err != nil {
		logger.Error("GEMINA_GATEWAY_RUN_AS", "value", spec, "error", err.Error())
		os.Exit(1)
	}
	dropped, err := dropPrivileges(uid, gid)
	if err != nil {
		logger.Error("drop privileges", "uid", uid, "gid", gid, "error", err.Error())
		os.Exit(1)
	}
	if dropped {
		logger.Info("dropped privileges", "uid", uid, "gid", gid, "capabilities", "none")
	} else {
		logger.Info("running unprivileged", "uid", os.Geteuid(), "gid", os.Getegid())
	}
}

// connSink delivers framed return datagrams over the gateway's UDP socket. It
// implements exit.Sink so the return path can reach a client's source endpoints.
type connSink struct{ conn net.PacketConn }

func (s connSink) SendTo(datagram []byte, dst netip.AddrPort) error {
	_, err := s.conn.WriteTo(datagram, net.UDPAddrFromAddrPort(dst))
	return err
}

// exitDevice holds the privileged half of the exit path: the opened TUN device
// plus the parsed pool. It is created before the privilege drop and wired into
// the data gateway afterwards.
type exitDevice struct {
	dev     *exit.TUN
	alloc   *exit.Allocator
	tunName string
	mtu     int
	pool    string
}

// openExitDevice parses the pool and opens the TUN device. This is the only
// step that needs CAP_NET_ADMIN (TUNSETIFF on a TUN owned by another uid, and
// SIOCSIFMTU when the host has not already set the MTU). On other platforms
// OpenTUN returns a clear error.
func openExitDevice(logger *slog.Logger) (*exitDevice, error) {
	poolStr := envOr("GEMINA_GATEWAY_POOL", "10.99.0.0/16")
	pool, err := netip.ParsePrefix(poolStr)
	if err != nil {
		return nil, fmt.Errorf("GEMINA_GATEWAY_POOL %q: %w", poolStr, err)
	}
	alloc, err := exit.NewAllocator(pool)
	if err != nil {
		return nil, err
	}

	tunName := envOr("GEMINA_GATEWAY_TUN", "gemina0")
	mtu := envInt(logger, "GEMINA_GATEWAY_TUN_MTU", 1280)
	dev, err := exit.OpenTUN(tunName, mtu)
	if err != nil {
		return nil, fmt.Errorf("open tun %q: %w", tunName, err)
	}
	return &exitDevice{dev: dev, alloc: alloc, tunName: tunName, mtu: mtu, pool: poolStr}, nil
}

// enable builds the path set and router over the already-open TUN and turns on
// the exit path. It needs no privileges and runs after the drop.
func (e *exitDevice) enable(dg *gateway.DataGateway, conn net.PacketConn, logger *slog.Logger) {
	// The kernel does the NAT; we only health-assert it so a misconfigured host
	// surfaces a loud warning instead of silently dropping all egress.
	if err := exit.AssertIPForward(); err != nil {
		logger.Warn("ip forwarding not enabled; egress will not route until fixed", "error", err.Error())
	}

	paths := exit.NewPathSet(2 * time.Minute)
	router := exit.NewRouter(e.alloc, paths, e.dev, dg, connSink{conn}, dg)
	dg.EnableExit(router)
	logger.Info("exit path enabled", "tun", e.tunName, "mtu", e.mtu, "pool", e.pool)
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
