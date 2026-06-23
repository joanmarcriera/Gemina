package gateway

import (
	"bytes"
	"testing"

	"continuity-vpn/internal/protocol"
	"continuity-vpn/pkg/clientcore"
)

type staticKeys map[protocol.SessionID][]byte

func (s staticKeys) SessionKey(id protocol.SessionID) ([]byte, bool) {
	k, ok := s[id]
	return k, ok
}

func sessionID(b byte) protocol.SessionID {
	var id protocol.SessionID
	copy(id[:], bytes.Repeat([]byte{b}, protocol.SessionIDSize))
	return id
}

func TestDataPlaneDecryptsAndDeduplicatesAcrossPaths(t *testing.T) {
	id := sessionID(0x11)
	key := bytes.Repeat([]byte{0x42}, 32)
	client, err := clientcore.NewSession(id, key, clientcore.RoleInitiator, 1024)
	if err != nil {
		t.Fatalf("client session: %v", err)
	}

	dp := NewDataPlane(staticKeys{id: key}, 1024)

	payload := []byte("a real tunnelled IP packet")
	wire, err := client.Outbound(payload)
	if err != nil {
		t.Fatalf("outbound: %v", err)
	}

	// First copy over Wi-Fi: decrypted and delivered.
	got, first, err := dp.Handle(wire, "wifi")
	if err != nil {
		t.Fatalf("handle copy A: %v", err)
	}
	if !first || !bytes.Equal(got, payload) {
		t.Fatalf("copy A first=%v payload=%q want %q", first, got, payload)
	}

	// Same logical packet over the USB path: a duplicate to drop.
	_, first, err = dp.Handle(wire, "usb")
	if err != nil {
		t.Fatalf("handle copy B: %v", err)
	}
	if first {
		t.Fatal("duplicate over second path was not suppressed")
	}

	// A distinct packet is delivered.
	wire2, _ := client.Outbound([]byte("second packet"))
	_, first, err = dp.Handle(wire2, "wifi")
	if err != nil || !first {
		t.Fatalf("distinct packet first=%v err=%v", first, err)
	}
}

func TestDataPlaneRejectsUnknownSession(t *testing.T) {
	id := sessionID(0x22)
	key := bytes.Repeat([]byte{0x7E}, 32)
	client, _ := clientcore.NewSession(id, key, clientcore.RoleInitiator, 16)
	wire, _ := client.Outbound([]byte("x"))

	// Resolver has no key for this session.
	dp := NewDataPlane(staticKeys{}, 16)
	if _, _, err := dp.Handle(wire, "wifi"); err == nil {
		t.Fatal("expected error for a session with no key")
	}
}

func TestDataPlaneRejectsForgedPacket(t *testing.T) {
	id := sessionID(0x33)
	key := bytes.Repeat([]byte{0x5A}, 32)
	client, _ := clientcore.NewSession(id, key, clientcore.RoleInitiator, 16)
	dp := NewDataPlane(staticKeys{id: key}, 16)

	wire, _ := client.Outbound([]byte("genuine"))
	forged := append([]byte(nil), wire...)
	forged[len(forged)-1] ^= 0xFF
	if _, _, err := dp.Handle(forged, "attacker"); err == nil {
		t.Fatal("forged packet was accepted")
	}
}
