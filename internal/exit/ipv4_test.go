package exit

import (
	"net/netip"
	"testing"
)

func TestParseIPv4ValidPacket(t *testing.T) {
	// Minimal IPv4 header: version=4, IHL=5 (20 bytes), no options.
	// Source: 192.0.2.1 (TEST-NET-1), Destination: 203.0.113.1 (TEST-NET-3).
	pkt := makeIPv4Header(
		[4]byte{192, 0, 2, 1},   // src
		[4]byte{203, 0, 113, 1}, // dst
	)

	src, dst, ok := parseIPv4(pkt)
	if !ok {
		t.Fatal("parseIPv4 returned ok=false for valid packet")
	}
	if want := netip.MustParseAddr("192.0.2.1"); src != want {
		t.Fatalf("src = %s, want %s", src, want)
	}
	if want := netip.MustParseAddr("203.0.113.1"); dst != want {
		t.Fatalf("dst = %s, want %s", dst, want)
	}
}

func TestParseIPv4TooShort(t *testing.T) {
	_, _, ok := parseIPv4([]byte{0x45, 0x00, 0x00}) // only 3 bytes
	if ok {
		t.Fatal("parseIPv4 returned ok=true for too-short packet")
	}
}

func TestParseIPv4EmptyPacket(t *testing.T) {
	_, _, ok := parseIPv4(nil)
	if ok {
		t.Fatal("parseIPv4 returned ok=true for nil packet")
	}
}

func TestParseIPv4WrongVersion(t *testing.T) {
	pkt := makeIPv4Header(
		[4]byte{192, 0, 2, 1},
		[4]byte{203, 0, 113, 1},
	)
	// Overwrite version nibble to 6 (IPv6).
	pkt[0] = (pkt[0] & 0x0f) | 0x60

	_, _, ok := parseIPv4(pkt)
	if ok {
		t.Fatal("parseIPv4 returned ok=true for IPv6 version nibble")
	}
}

func TestParseIPv4BadIHL(t *testing.T) {
	pkt := makeIPv4Header(
		[4]byte{192, 0, 2, 1},
		[4]byte{203, 0, 113, 1},
	)
	// Set IHL=1 (4 bytes) — below the 20-byte minimum.
	pkt[0] = (pkt[0] & 0xf0) | 0x01

	_, _, ok := parseIPv4(pkt)
	if ok {
		t.Fatal("parseIPv4 returned ok=true for IHL < 5")
	}
}

// makeIPv4Header builds a minimal 20-byte IPv4 header with the given src/dst.
// The packet payload is empty; other fields (TTL, proto, checksum) are zeroed
// because parseIPv4 does not inspect them.
func makeIPv4Header(src, dst [4]byte) []byte {
	pkt := make([]byte, 20)
	pkt[0] = 0x45 // version=4, IHL=5
	pkt[2] = 0x00 // total length high byte
	pkt[3] = 0x14 // total length low byte (20)
	copy(pkt[12:16], src[:])
	copy(pkt[16:20], dst[:])
	return pkt
}
