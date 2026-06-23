package main

import (
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"

	"continuity-vpn/internal/protocol"
	"continuity-vpn/internal/transport"
)

var errUnknownPath = errors.New("unknown path tag")

type probeConfig struct {
	iface     string
	to        string
	path      protocol.PathTag
	count     int
	duplicate bool

	// Optional second path. When iface2 is set, each probe identity is sent over
	// both interfaces (a cross-path duplicate), demonstrating dual-path egress.
	iface2 string
	path2  protocol.PathTag
}

func (c probeConfig) dualPath() bool { return c.iface2 != "" }

// parseProbeConfig parses the `probe` subcommand flags. It is separated from the
// network send so the flag/validation logic is unit-testable.
func parseProbeConfig(args []string) (probeConfig, error) {
	fs := flag.NewFlagSet("probe", flag.ContinueOnError)
	iface := fs.String("interface", "", "BSD interface name to bind egress to (e.g. en0)")
	to := fs.String("to", "", "gateway address host:port")
	pathName := fs.String("path", "wifi", "path tag: wifi|usb")
	count := fs.Int("count", 1, "number of distinct probes to send")
	duplicate := fs.Bool("duplicate", false, "also send each probe a second time to exercise dedup")
	iface2 := fs.String("interface2", "", "optional second interface; sends each probe over both paths")
	path2Name := fs.String("path2", "usb", "path tag for the second interface")
	if err := fs.Parse(args); err != nil {
		return probeConfig{}, err
	}

	if *iface == "" {
		return probeConfig{}, errors.New("probe: -interface is required")
	}
	if *to == "" {
		return probeConfig{}, errors.New("probe: -to is required")
	}
	if *count < 1 {
		return probeConfig{}, errors.New("probe: -count must be at least 1")
	}

	tag, err := parsePathTag(*pathName)
	if err != nil {
		return probeConfig{}, err
	}

	cfg := probeConfig{iface: *iface, to: *to, path: tag, count: *count, duplicate: *duplicate}

	if *iface2 != "" {
		tag2, err := parsePathTag(*path2Name)
		if err != nil {
			return probeConfig{}, err
		}
		cfg.iface2 = *iface2
		cfg.path2 = tag2
	}

	return cfg, nil
}

func parsePathTag(name string) (protocol.PathTag, error) {
	switch name {
	case "wifi", "wi-fi":
		return protocol.PathWiFi, nil
	case "usb", "android-usb-tether", "android-usb-tethering":
		return protocol.PathAndroidUSBTether, nil
	default:
		return protocol.PathUnknown, fmt.Errorf("%w: %q", errUnknownPath, name)
	}
}

// runProbe sends count probes (optionally each duplicated) to the gateway over a
// socket bound to the chosen interface, then reports a redacted summary. It does
// not print the resolved gateway address.
func runProbe(args []string, out io.Writer) error {
	cfg, err := parseProbeConfig(args)
	if err != nil {
		return err
	}

	conn, err := transport.PathDialer{Interface: cfg.iface}.DialUDP(cfg.to)
	if err != nil {
		return err
	}
	defer conn.Close()

	var conn2 *net.UDPConn
	if cfg.dualPath() {
		conn2, err = transport.PathDialer{Interface: cfg.iface2}.DialUDP(cfg.to)
		if err != nil {
			return fmt.Errorf("second path: %w", err)
		}
		defer conn2.Close()
	}

	session, err := randomSession()
	if err != nil {
		return err
	}

	sent := 0
	for n := 1; n <= cfg.count; n++ {
		id := protocol.PacketID{Session: session, Number: protocol.PacketNumber(n)}

		wire, err := protocol.ProbePacket{ID: id, Path: cfg.path}.MarshalBinary()
		if err != nil {
			return err
		}
		copies := 1
		if cfg.duplicate {
			copies = 2
		}
		for i := 0; i < copies; i++ {
			if _, err := conn.Write(wire); err != nil {
				return fmt.Errorf("send probe %d over %s: %w", n, cfg.iface, err)
			}
			sent++
		}

		if conn2 != nil {
			// Same identity over the second path: a cross-path duplicate.
			wire2, err := protocol.ProbePacket{ID: id, Path: cfg.path2}.MarshalBinary()
			if err != nil {
				return err
			}
			if _, err := conn2.Write(wire2); err != nil {
				return fmt.Errorf("send probe %d over %s: %w", n, cfg.iface2, err)
			}
			sent++
		}
	}

	if cfg.dualPath() {
		fmt.Fprintf(out, "sent %d datagram(s): %d distinct probe(s) over %s (%q) and %s (%q)\n",
			sent, cfg.count, cfg.iface, cfg.path.String(), cfg.iface2, cfg.path2.String())
	} else {
		fmt.Fprintf(out, "sent %d datagram(s): %d distinct probe(s) over %s as path %q%s\n",
			sent, cfg.count, cfg.iface, cfg.path.String(), duplicateNote(cfg.duplicate))
	}
	return nil
}

func duplicateNote(dup bool) string {
	if dup {
		return " (each duplicated)"
	}
	return ""
}

func randomSession() (protocol.SessionID, error) {
	var s protocol.SessionID
	if _, err := rand.Read(s[:]); err != nil {
		return s, fmt.Errorf("generate session id: %w", err)
	}
	// A zero session is invalid; the chance is negligible, but guard anyway.
	if s.IsZero() {
		s[0] = 1
	}
	return s, nil
}
