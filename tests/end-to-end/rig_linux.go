//go:build linux && e2e

// Command rig is the Stage-2 on-hardware test harness (not a shipped product). It
// stands in for the macOS client so the server exit path can be demonstrated on
// Linux: it opens a TUN, performs the real handshake against a running data
// gateway, and pumps inner IP packets over one or more uplinks at once —
// duplicating each outbound packet over every path and relying on the gateway +
// the client session to deduplicate. Cutting one uplink mid-flow must not break
// an established flow (the Stage-2 exit criterion).
//
// Build (compile-check from any host):
//
//	GOOS=linux GOARCH=arm64 go build -tags e2e ./tests/end-to-end/...
//
// Run on a Linux box with root (CAP_NET_ADMIN for the TUN):
//
//	sudo GEMINA_RIG_GATEWAY=gw.example:51820 \
//	     GEMINA_RIG_IDENTITY=<base64 ed25519 pub from the gateway log> \
//	     GEMINA_RIG_TOKEN=<entitlement token> \
//	     GEMINA_RIG_TUNNEL_IP=10.99.0.2 \
//	     GEMINA_RIG_PATHS=eth0,wwan0 \
//	     ./rig
//
// The tunnel IP is learned out-of-band for now (the first admitted client gets
// <pool>.0.2); in-band delivery is a tracked follow-up.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joanmarcriera/gemina/internal/exit"
	"github.com/joanmarcriera/gemina/pkg/clientcore"
)

func main() {
	gateway := mustEnv("GEMINA_RIG_GATEWAY")
	identityB64 := mustEnv("GEMINA_RIG_IDENTITY")
	token := mustEnv("GEMINA_RIG_TOKEN")
	tunnelIP := envOr("GEMINA_RIG_TUNNEL_IP", "10.99.0.2")
	paths := splitPaths(envOr("GEMINA_RIG_PATHS", "")) // empty => default route, single path

	identity, err := base64.StdEncoding.DecodeString(identityB64)
	if err != nil || len(identity) != ed25519.PublicKeySize {
		log.Fatalf("GEMINA_RIG_IDENTITY must be a base64 ed25519 public key")
	}

	gwAddr, err := net.ResolveUDPAddr("udp", gateway)
	if err != nil {
		log.Fatalf("resolve gateway %q: %v", gateway, err)
	}

	// One UDP socket per uplink; an empty path list yields a single default socket.
	socks, err := openPaths(paths, gwAddr)
	if err != nil {
		log.Fatalf("open uplinks: %v", err)
	}
	defer func() {
		for _, s := range socks {
			_ = s.Close()
		}
	}()

	// Handshake over the first uplink.
	session, err := handshake(socks[0], gwAddr, identity, token)
	if err != nil {
		log.Fatalf("handshake: %v", err)
	}
	log.Printf("session established over %d path(s)", len(socks))

	dev, err := exit.OpenTUN("continuity-rig", 1280)
	if err != nil {
		log.Fatalf("open tun: %v", err)
	}
	defer dev.Close()
	if err := configureTUN("continuity-rig", tunnelIP); err != nil {
		log.Fatalf("configure tun: %v", err)
	}
	log.Printf("tun continuity-rig up with %s; routing demo traffic through the tunnel", tunnelIP)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Inbound: each path receiver decrypts and dedups via the session, writing the
	// first copy of each packet to the TUN.
	for i, s := range socks {
		go receiveLoop(ctx, s, session, dev, pathLabel(paths, i))
	}

	// Outbound: read inner packets from the TUN, frame once, send a copy over
	// every uplink. A cut uplink simply stops carrying copies; the others keep the
	// flow alive.
	buf := make([]byte, 65535)
	for {
		n, err := dev.Read(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("tun read: %v", err)
			return
		}
		framed, err := session.Outbound(buf[:n])
		if err != nil {
			continue
		}
		for _, s := range socks {
			_, _ = s.WriteToUDP(framed, gwAddr)
		}
	}
}

func handshake(sock *net.UDPConn, gw *net.UDPAddr, identity ed25519.PublicKey, token string) (*clientcore.Session, error) {
	hello, hs, err := clientcore.BeginClientHandshake(identity, token)
	if err != nil {
		return nil, err
	}
	if _, err := sock.WriteToUDP(hello, gw); err != nil {
		return nil, err
	}
	_ = sock.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 2048)
	n, _, err := sock.ReadFromUDP(buf)
	if err != nil {
		return nil, err
	}
	_ = sock.SetReadDeadline(time.Time{})
	return hs.Complete(buf[:n], 1024)
}

func receiveLoop(ctx context.Context, sock *net.UDPConn, session *clientcore.Session, dev *exit.TUN, path string) {
	buf := make([]byte, 65535)
	for {
		n, _, err := sock.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		payload, first, err := session.Inbound(buf[:n], path)
		if err != nil || !first {
			continue
		}
		_, _ = dev.Write(payload)
	}
}

// openPaths opens one UDP socket per named uplink (SO_BINDTODEVICE), or a single
// default-route socket when no paths are given.
func openPaths(paths []string, gw *net.UDPAddr) ([]*net.UDPConn, error) {
	if len(paths) == 0 {
		c, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			return nil, err
		}
		return []*net.UDPConn{c}, nil
	}

	var socks []*net.UDPConn
	for _, dev := range paths {
		lc := net.ListenConfig{Control: bindToDevice(dev)}
		pc, err := lc.ListenPacket(context.Background(), "udp", ":0")
		if err != nil {
			return nil, err
		}
		socks = append(socks, pc.(*net.UDPConn))
	}
	_ = gw
	return socks, nil
}

func bindToDevice(dev string) func(network, address string, c syscall.RawConn) error {
	return func(_, _ string, c syscall.RawConn) error {
		var serr error
		if err := c.Control(func(fd uintptr) {
			serr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, dev)
		}); err != nil {
			return err
		}
		return serr
	}
}

// configureTUN assigns the tunnel address and brings the interface up via the
// `ip` tool (the demo host has it; the production client uses NEPacketTunnel).
func configureTUN(name, addr string) error {
	prefix, err := netip.ParseAddr(addr)
	if err != nil {
		return err
	}
	for _, args := range [][]string{
		{"addr", "add", prefix.String() + "/16", "dev", name},
		{"link", "set", "dev", name, "up"},
	} {
		if out, err := runIP(args...); err != nil {
			return errors.New("ip " + strings.Join(args, " ") + ": " + err.Error() + ": " + out)
		}
	}
	return nil
}

func runIP(args ...string) (string, error) {
	out, err := exec.Command("ip", args...).CombinedOutput()
	return string(out), err
}

func pathLabel(paths []string, i int) string {
	if i < len(paths) {
		return paths[i]
	}
	return "default"
}

func splitPaths(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is required", key)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
