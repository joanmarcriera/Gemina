package exit

import "net/netip"

// ipv4HeaderMinLen is the minimum byte length of a valid IPv4 header (no
// options). Packets shorter than this cannot be parsed.
const ipv4HeaderMinLen = 20

// parseIPv4 extracts the source and destination addresses from a raw IPv4
// packet. It returns ok=false for any packet that is too short, whose version
// nibble is not 4, or whose IHL field implies a header shorter than the
// minimum. Callers must validate ok before using the returned addresses.
func parseIPv4(packet []byte) (src, dst netip.Addr, ok bool) {
	if len(packet) < ipv4HeaderMinLen {
		return netip.Addr{}, netip.Addr{}, false
	}
	// The high nibble of the first byte is the IP version.
	if packet[0]>>4 != 4 {
		return netip.Addr{}, netip.Addr{}, false
	}
	// IHL (low nibble of byte 0) is in 32-bit words; minimum valid value is 5
	// (= 20 bytes). Packets claiming a smaller header are malformed.
	ihl := int(packet[0]&0x0f) * 4
	if ihl < ipv4HeaderMinLen || len(packet) < ihl {
		return netip.Addr{}, netip.Addr{}, false
	}
	// Source address is at bytes 12–15, destination at bytes 16–19.
	src = netip.AddrFrom4([4]byte(packet[12:16]))
	dst = netip.AddrFrom4([4]byte(packet[16:20]))
	return src, dst, true
}
