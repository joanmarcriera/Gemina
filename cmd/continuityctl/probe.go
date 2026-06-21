package main

import (
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"

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
}

// parseProbeConfig parses the `probe` subcommand flags. It is separated from the
// network send so the flag/validation logic is unit-testable.
func parseProbeConfig(args []string) (probeConfig, error) {
	fs := flag.NewFlagSet("probe", flag.ContinueOnError)
	iface := fs.String("interface", "", "BSD interface name to bind egress to (e.g. en0)")
	to := fs.String("to", "", "gateway address host:port")
	pathName := fs.String("path", "wifi", "path tag: wifi|usb")
	count := fs.Int("count", 1, "number of distinct probes to send")
	duplicate := fs.Bool("duplicate", false, "also send each probe a second time to exercise dedup")
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

	return probeConfig{iface: *iface, to: *to, path: tag, count: *count, duplicate: *duplicate}, nil
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

	session, err := randomSession()
	if err != nil {
		return err
	}

	sent := 0
	for n := 1; n <= cfg.count; n++ {
		packet := protocol.ProbePacket{
			ID:   protocol.PacketID{Session: session, Number: protocol.PacketNumber(n)},
			Path: cfg.path,
		}
		wire, err := packet.MarshalBinary()
		if err != nil {
			return err
		}
		copies := 1
		if cfg.duplicate {
			copies = 2
		}
		for i := 0; i < copies; i++ {
			if _, err := conn.Write(wire); err != nil {
				return fmt.Errorf("send probe %d: %w", n, err)
			}
			sent++
		}
	}

	fmt.Fprintf(out, "sent %d datagram(s): %d distinct probe(s) over %s as path %q%s\n",
		sent, cfg.count, cfg.iface, cfg.path.String(), duplicateNote(cfg.duplicate))
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
