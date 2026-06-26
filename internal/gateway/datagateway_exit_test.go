package gateway

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/joanmarcriera/gemina/internal/entitlement"
	"github.com/joanmarcriera/gemina/internal/exit"
	"github.com/joanmarcriera/gemina/pkg/clientcore"
)

// fakeDevice stands in for the TUN device: Write captures egressed inner packets,
// Read yields injected "return from internet" packets. It lets the exit path be
// exercised in-process without a real /dev/net/tun.
type fakeDevice struct {
	egress chan []byte
	inject chan []byte
}

func newFakeDevice() *fakeDevice {
	return &fakeDevice{egress: make(chan []byte, 8), inject: make(chan []byte, 8)}
}

func (d *fakeDevice) Write(p []byte) (int, error) {
	d.egress <- append([]byte(nil), p...)
	return len(p), nil
}

func (d *fakeDevice) Read(p []byte) (int, error) {
	pkt, ok := <-d.inject
	if !ok {
		return 0, io.EOF
	}
	return copy(p, pkt), nil
}

// testSink writes framed return datagrams back over the gateway's UDP socket,
// mirroring the production connSink in cmd/gateway.
type testSink struct{ conn net.PacketConn }

func (s testSink) SendTo(datagram []byte, dst netip.AddrPort) error {
	_, err := s.conn.WriteTo(datagram, net.UDPAddrFromAddrPort(dst))
	return err
}

// craftIPv4 builds a minimal valid IPv4 packet (20-byte header + payload) with
// the given src/dst, enough for the exit router's parser and reverse-path filter.
func craftIPv4(src, dst netip.Addr, payload []byte) []byte {
	pkt := make([]byte, 20+len(payload))
	pkt[0] = 0x45 // version 4, IHL 5 (no options)
	binary.BigEndian.PutUint16(pkt[2:], uint16(20+len(payload)))
	pkt[8] = 64 // TTL
	pkt[9] = 17 // protocol UDP (arbitrary; the router does not inspect it)
	s4, d4 := src.As4(), dst.As4()
	copy(pkt[12:16], s4[:])
	copy(pkt[16:20], d4[:])
	copy(pkt[20:], payload)
	return pkt
}

// TestDataGatewayExitForwardsAndReturns drives the whole exit path over real
// loopback UDP: a client handshakes, sends an encrypted inner packet that the
// gateway decrypts and forwards to the (fake) TUN, then a return packet injected
// at the TUN is framed back to the client and decrypts cleanly. This is the
// Stage-2 server-exit exit criterion in miniature.
func TestDataGatewayExitForwardsAndReturns(t *testing.T) {
	idPriv, idPub, err := clientcore.GenerateIdentity()
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	dg := NewDataGateway(idPriv, &entitlement.Service{Mode: entitlement.ModeOpen}, 64, nil)

	pool := netip.MustParsePrefix("10.99.0.0/16")
	alloc, err := exit.NewAllocator(pool)
	if err != nil {
		t.Fatalf("allocator: %v", err)
	}
	dev := newFakeDevice()

	gwConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("gateway listen: %v", err)
	}
	defer gwConn.Close()

	router := exit.NewRouter(alloc, exit.NewPathSet(time.Minute), dev, dg, testSink{gwConn}, dg)
	dg.EnableExit(router)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() { cancel(); close(dev.inject) }()
	go func() { _ = dg.Serve(ctx, gwConn) }()

	gwAddr := gwConn.LocalAddr()
	clConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("client listen: %v", err)
	}
	defer clConn.Close()

	// Handshake.
	hello, hs, err := clientcore.BeginClientHandshake(idPub, "open")
	if err != nil {
		t.Fatalf("begin handshake: %v", err)
	}
	if _, err := clConn.WriteTo(hello, gwAddr); err != nil {
		t.Fatalf("send ClientHello: %v", err)
	}
	rbuf := make([]byte, 2048)
	_ = clConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := clConn.ReadFrom(rbuf)
	if err != nil {
		t.Fatalf("read ServerHello: %v", err)
	}
	session, err := hs.Complete(rbuf[:n], 64)
	if err != nil {
		t.Fatalf("complete handshake: %v", err)
	}

	// The client's assigned tunnel IP (idempotent with the gateway's admit-time lease).
	id, _, _, _, err := clientcore.DecodeClientHello(hello)
	if err != nil {
		t.Fatalf("decode hello: %v", err)
	}
	tunnelIP, err := alloc.Allocate(id)
	if err != nil {
		t.Fatalf("learn tunnel IP: %v", err)
	}
	peer := netip.MustParseAddr("198.51.100.7") // TEST-NET-2 documentation address

	// Client sends an encrypted inner packet sourced from its tunnel IP.
	inner := craftIPv4(tunnelIP, peer, []byte("hello internet"))
	wire, err := session.Outbound(inner)
	if err != nil {
		t.Fatalf("outbound: %v", err)
	}
	if _, err := clConn.WriteTo(wire, gwAddr); err != nil {
		t.Fatalf("send data: %v", err)
	}

	// The gateway decrypts and forwards the exact inner packet to the TUN.
	select {
	case got := <-dev.egress:
		if !bytes.Equal(got, inner) {
			t.Fatalf("forwarded packet mismatch: got %d bytes, want %d", len(got), len(inner))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("inner packet was not forwarded to the exit device")
	}

	// A return packet from the internet, addressed to the client's tunnel IP, is
	// framed back over the client's path and decrypts to the original payload.
	ret := craftIPv4(peer, tunnelIP, []byte("hello client"))
	dev.inject <- ret

	_ = clConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err = clConn.ReadFrom(rbuf)
	if err != nil {
		t.Fatalf("read return datagram: %v", err)
	}
	payload, first, err := session.Inbound(rbuf[:n], "remote")
	if err != nil || !first {
		t.Fatalf("return decrypt: first=%v err=%v", first, err)
	}
	if !bytes.Equal(payload, ret) {
		t.Fatalf("return payload mismatch: got %d bytes, want %d", len(payload), len(ret))
	}
}
