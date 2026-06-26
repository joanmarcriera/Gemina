package gateway

import (
	"regexp"
	"strings"
	"testing"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

func probeBytes(t *testing.T, number protocol.PacketNumber, path protocol.PathTag) []byte {
	t.Helper()
	var session protocol.SessionID
	copy(session[:], []byte("metrics-test-sess"))
	wire, err := protocol.ProbePacket{
		ID:   protocol.PacketID{Session: session, Number: number},
		Path: path,
	}.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal probe: %v", err)
	}
	return wire
}

func TestGatewayMetricsCountDecisionsByPath(t *testing.T) {
	s, err := NewServer(1024, nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	first := probeBytes(t, 1, protocol.PathWiFi)
	s.Handle(first)       // first-copy over wi-fi
	s.Handle(first)       // duplicate over wi-fi
	s.Handle([]byte("x")) // rejected (too short)

	out := s.Metrics().Render()
	for _, want := range []string{
		`gemina_packets_total{decision="first-copy",path="wi-fi"} 1`,
		`gemina_packets_total{decision="duplicate",path="wi-fi"} 1`,
		`gemina_rejected_total{reason="short"} 1`,
		"# TYPE gemina_packets_total counter",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("metrics missing %q in:\n%s", want, out)
		}
	}
}

func TestGatewayMetricsNeverLeakIdentifiers(t *testing.T) {
	s, _ := NewServer(1024, nil)
	s.Handle(probeBytes(t, 1, protocol.PathAndroidUSBTether))

	out := s.Metrics().Render()
	ipv4 := regexp.MustCompile(`\b(\d{1,3}\.){3}\d{1,3}\b`)
	mac := regexp.MustCompile(`([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}`)
	if ipv4.MatchString(out) || mac.MatchString(out) {
		t.Fatalf("metrics output leaked a host identifier:\n%s", out)
	}
	if strings.Contains(out, "metrics-test-sess") {
		t.Fatalf("metrics output leaked a session id:\n%s", out)
	}
}
