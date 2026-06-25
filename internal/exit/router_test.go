package exit

import (
	"context"
	"errors"
	"net/netip"
	"sync"
	"testing"
	"time"

	"continuity-vpn/internal/protocol"
)

// fakeDevice is an in-memory Device whose Write side queues packets for
// inspection and whose Read side is fed by the test via inject().
type fakeDevice struct {
	mu      sync.Mutex
	written [][]byte
	reads   chan []byte
}

func newFakeDevice() *fakeDevice {
	return &fakeDevice{reads: make(chan []byte, 16)}
}

func (d *fakeDevice) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	copy(cp, p)
	d.mu.Lock()
	d.written = append(d.written, cp)
	d.mu.Unlock()
	return len(p), nil
}

// Read blocks until inject() delivers a packet or the channel is closed.
func (d *fakeDevice) Read(p []byte) (int, error) {
	pkt, ok := <-d.reads
	if !ok {
		return 0, errors.New("device closed")
	}
	n := copy(p, pkt)
	return n, nil
}

// inject queues a packet to be returned by the next Read call.
func (d *fakeDevice) inject(pkt []byte) {
	cp := make([]byte, len(pkt))
	copy(cp, pkt)
	d.reads <- cp
}

// close signals EOF to any blocked Read.
func (d *fakeDevice) close() { close(d.reads) }

func (d *fakeDevice) writtenPackets() [][]byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([][]byte, len(d.written))
	copy(out, d.written)
	return out
}

// fakeFramer returns a trivially framed datagram: the literal bytes
// "FRAMED:" + payload, keyed to id. Good enough for round-trip testing.
type fakeFramer struct{}

func (fakeFramer) FrameReturn(id protocol.SessionID, payload []byte) ([]byte, error) {
	prefix := append([]byte("FRAMED:"), id[0]) // one id byte is enough to identify
	return append(prefix, payload...), nil
}

// fakeSink records every SendTo call.
type fakeSink struct {
	mu   sync.Mutex
	sent []sinkCall
}

type sinkCall struct {
	datagram []byte
	dst      netip.AddrPort
}

func (s *fakeSink) SendTo(datagram []byte, dst netip.AddrPort) error {
	cp := make([]byte, len(datagram))
	copy(cp, datagram)
	s.mu.Lock()
	s.sent = append(s.sent, sinkCall{datagram: cp, dst: dst})
	s.mu.Unlock()
	return nil
}

func (s *fakeSink) calls() []sinkCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]sinkCall, len(s.sent))
	copy(out, s.sent)
	return out
}

// fakeMetrics records every Dropped/Forwarded/Returned call for assertions.
type fakeMetrics struct {
	mu        sync.Mutex
	forwarded int
	returned  int
	dropped   []string
}

func (m *fakeMetrics) Forwarded(bytes int) {
	m.mu.Lock()
	m.forwarded += bytes
	m.mu.Unlock()
}

func (m *fakeMetrics) Returned(bytes int) {
	m.mu.Lock()
	m.returned += bytes
	m.mu.Unlock()
}

func (m *fakeMetrics) Dropped(reason string) {
	m.mu.Lock()
	m.dropped = append(m.dropped, reason)
	m.mu.Unlock()
}

func (m *fakeMetrics) droppedReasons() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.dropped))
	copy(out, m.dropped)
	return out
}

// ---- helpers ----------------------------------------------------------------

func newTestRouter(t *testing.T, poolPrefix string) (*Router, *Allocator, *PathSet, *fakeDevice, *fakeSink, *fakeMetrics) {
	t.Helper()
	alloc := newTestAllocator(t, poolPrefix)
	ps := NewPathSet(30 * time.Second)
	dev := newFakeDevice()
	sink := &fakeSink{}
	met := &fakeMetrics{}
	r := NewRouter(alloc, ps, dev, fakeFramer{}, sink, met)
	return r, alloc, ps, dev, sink, met
}

// buildIPv4 constructs a minimal 20-byte IPv4 packet with the given src/dst.
func buildIPv4(src, dst netip.Addr) []byte {
	pkt := make([]byte, 20)
	pkt[0] = 0x45 // version=4, IHL=5
	pkt[3] = 20   // total length
	s4 := src.As4()
	d4 := dst.As4()
	copy(pkt[12:16], s4[:])
	copy(pkt[16:20], d4[:])
	return pkt
}

// ---- Egress tests -----------------------------------------------------------

func TestEgressForwardsWhenSourceMatchesLease(t *testing.T) {
	r, alloc, _, dev, _, met := newTestRouter(t, "10.0.0.0/24")
	id := testSessionID(1)

	leased, err := alloc.Allocate(id)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}

	// Inner packet: src = leased tunnel IP, dst = TEST-NET-3 server.
	pkt := buildIPv4(leased, netip.MustParseAddr("203.0.113.1"))

	if err := r.Egress(id, pkt); err != nil {
		t.Fatalf("Egress: %v", err)
	}

	written := dev.writtenPackets()
	if len(written) != 1 {
		t.Fatalf("device received %d packets, want 1", len(written))
	}
	if met.forwarded == 0 {
		t.Fatal("Forwarded metric not incremented")
	}
	if len(met.droppedReasons()) != 0 {
		t.Fatalf("unexpected drops: %v", met.droppedReasons())
	}
}

func TestEgressDropsOnReversePathMismatch(t *testing.T) {
	r, alloc, _, dev, _, met := newTestRouter(t, "10.0.0.0/24")
	id := testSessionID(1)

	_, err := alloc.Allocate(id)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}

	// Inner packet claims a source that is NOT the session's leased address.
	spoofedSrc := netip.MustParseAddr("10.0.0.200")
	pkt := buildIPv4(spoofedSrc, netip.MustParseAddr("203.0.113.1"))

	err = r.Egress(id, pkt)
	if !errors.Is(err, ErrReversePath) {
		t.Fatalf("Egress error = %v, want ErrReversePath", err)
	}

	if len(dev.writtenPackets()) != 0 {
		t.Fatal("device received a packet despite reverse-path mismatch")
	}

	reasons := met.droppedReasons()
	if len(reasons) == 0 || reasons[0] != "reverse-path" {
		t.Fatalf("Dropped reasons = %v, want [reverse-path]", reasons)
	}
}

func TestEgressDropsBadInnerPacket(t *testing.T) {
	r, alloc, _, _, _, _ := newTestRouter(t, "10.0.0.0/24")
	id := testSessionID(1)
	if _, err := alloc.Allocate(id); err != nil {
		t.Fatalf("Allocate: %v", err)
	}

	err := r.Egress(id, []byte{0x40}) // too short to be a valid IPv4 packet
	if err == nil {
		t.Fatal("Egress with malformed inner packet returned nil error")
	}
}

// ---- ServeReturn tests ------------------------------------------------------

func TestServeReturnRoutesSinglePath(t *testing.T) {
	r, alloc, ps, dev, sink, _ := newTestRouter(t, "10.0.0.0/24")
	id := testSessionID(1)

	leased, err := alloc.Allocate(id)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}

	// Record one source endpoint for the session (TEST-NET-1 client).
	ep := netip.MustParseAddrPort("192.0.2.1:51820")
	ps.Record(id, ep)

	// Build a return packet: src = public internet, dst = session's tunnel IP.
	returnPkt := buildIPv4(netip.MustParseAddr("203.0.113.1"), leased)
	dev.inject(returnPkt)
	dev.close() // trigger EOF so ServeReturn exits

	ctx := context.Background()
	_ = r.ServeReturn(ctx) // device-closed error is returned but we don't assert it here

	calls := sink.calls()
	if len(calls) != 1 {
		t.Fatalf("sink received %d calls, want 1", len(calls))
	}
	if calls[0].dst != ep {
		t.Fatalf("sink dst = %s, want %s", calls[0].dst, ep)
	}
}

func TestServeReturnDuplicatesToAllFreshPaths(t *testing.T) {
	r, alloc, ps, dev, sink, _ := newTestRouter(t, "10.0.0.0/24")
	id := testSessionID(1)

	leased, err := alloc.Allocate(id)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}

	// Two source endpoints (Wi-Fi + cellular simulation).
	ep1 := netip.MustParseAddrPort("192.0.2.1:51820")
	ep2 := netip.MustParseAddrPort("198.51.100.1:4500")
	ps.Record(id, ep1)
	ps.Record(id, ep2)

	returnPkt := buildIPv4(netip.MustParseAddr("203.0.113.1"), leased)
	dev.inject(returnPkt)
	dev.close()

	_ = r.ServeReturn(context.Background())

	calls := sink.calls()
	if len(calls) != 2 {
		t.Fatalf("sink received %d calls, want 2 (one per path)", len(calls))
	}
	dsts := map[netip.AddrPort]bool{calls[0].dst: true, calls[1].dst: true}
	if !dsts[ep1] || !dsts[ep2] {
		t.Fatalf("sink dsts = %v, want both %s and %s", dsts, ep1, ep2)
	}
}

func TestServeReturnDropsUnknownDst(t *testing.T) {
	r, _, _, dev, sink, met := newTestRouter(t, "10.0.0.0/24")

	// Packet destined for an address that has no allocation.
	unknownDst := netip.MustParseAddr("10.0.0.200")
	returnPkt := buildIPv4(netip.MustParseAddr("203.0.113.1"), unknownDst)
	dev.inject(returnPkt)
	dev.close()

	_ = r.ServeReturn(context.Background())

	if len(sink.calls()) != 0 {
		t.Fatal("sink received a call for an unknown destination")
	}
	reasons := met.droppedReasons()
	found := false
	for _, r := range reasons {
		if r == "unknown-dst" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Dropped reasons = %v, want to contain unknown-dst", reasons)
	}
}

func TestServeReturnExitsOnContextCancel(t *testing.T) {
	r, _, _, _, _, _ := newTestRouter(t, "10.0.0.0/24")
	// dev has no packets; ServeReturn will block on Read until ctx is cancelled.
	// We use a separate goroutine and cancel quickly.
	dev := newFakeDevice()
	alloc := newTestAllocator(t, "10.0.0.0/24")
	ps := NewPathSet(30 * time.Second)
	router := NewRouter(alloc, ps, dev, fakeFramer{}, &fakeSink{}, nil)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- router.ServeReturn(ctx) }()

	cancel()
	// Unblock the Read call so the goroutine notices the cancelled context.
	dev.close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ServeReturn after cancel returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeReturn did not exit after context cancel")
	}

	_ = r // suppress unused warning from the outer router
}
