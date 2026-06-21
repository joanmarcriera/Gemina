//go:build darwin

package transport

import (
	"net"
	"testing"
	"time"

	"continuity-vpn/internal/protocol"
)

// TestDialUDPBindsLoopbackAndDelivers binds a UDP socket to the loopback
// interface via IP_BOUND_IF and confirms a probe sent over it is delivered to a
// loopback listener. This exercises the real socket option on the dev host
// without depending on Wi-Fi or the phone.
func TestDialUDPBindsLoopbackAndDelivers(t *testing.T) {
	listener, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	conn, err := PathDialer{Interface: "lo0"}.DialUDP(listener.LocalAddr().String())
	if err != nil {
		t.Fatalf("DialUDP bound to lo0: %v", err)
	}
	defer conn.Close()

	wire, err := protocol.ProbePacket{
		ID:   protocol.PacketID{Session: probeSession(0x5a), Number: 1},
		Path: protocol.PathWiFi,
	}.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal probe: %v", err)
	}
	if _, err := conn.Write(wire); err != nil {
		t.Fatalf("write over bound socket: %v", err)
	}

	buf := make([]byte, 64)
	_ = listener.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := listener.ReadFrom(buf)
	if err != nil {
		t.Fatalf("listener read: %v", err)
	}

	got, err := protocol.UnmarshalProbe(buf[:n])
	if err != nil {
		t.Fatalf("decode delivered probe: %v", err)
	}
	if got.ID.Number != 1 {
		t.Fatalf("delivered probe number = %d, want 1", got.ID.Number)
	}
}

func probeSession(b byte) protocol.SessionID {
	var s protocol.SessionID
	for i := range s {
		s[i] = b
	}
	return s
}
