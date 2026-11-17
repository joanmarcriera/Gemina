package exit

import (
	"errors"
	"net/netip"
	"testing"
	"time"
)

// TestRouterLease pins that Lease is idempotent per session, gives distinct
// sessions distinct addresses, and surfaces ErrPoolExhausted once the pool is
// spent (a /30 has exactly one assignable host after the gateway's .1).
func TestRouterLease(t *testing.T) {
	alloc := newTestAllocator(t, "10.9.0.0/30")
	r := NewRouter(alloc, NewPathSet(time.Second), newFakeDevice(), fakeFramer{}, &fakeSink{}, nil)

	a1, err := r.Lease(testSessionID(1))
	if err != nil {
		t.Fatalf("Lease(1): %v", err)
	}
	if want := netip.MustParseAddr("10.9.0.2"); a1 != want {
		t.Fatalf("Lease(1) = %v, want %v", a1, want)
	}
	again, err := r.Lease(testSessionID(1))
	if err != nil || again != a1 {
		t.Fatalf("second Lease(1) = %v, %v; want %v, nil (idempotent)", again, err, a1)
	}
	if _, err := r.Lease(testSessionID(2)); !errors.Is(err, ErrPoolExhausted) {
		t.Fatalf("Lease(2) on spent /30 err = %v, want ErrPoolExhausted", err)
	}
	if got, ok := alloc.LeaseOf(testSessionID(1)); !ok || got != a1 {
		t.Fatalf("LeaseOf after Lease = %v, %v", got, ok)
	}
}

// TestRouterRecordPath pins that RecordPath feeds the return-path set, and that
// recording the same endpoint twice does not duplicate it.
func TestRouterRecordPath(t *testing.T) {
	r, _, ps, _, _, _ := newTestRouter(t, "10.0.0.0/24")
	id := testSessionID(3)
	a := netip.MustParseAddrPort("192.0.2.10:4000")
	b := netip.MustParseAddrPort("198.51.100.10:4001")

	r.RecordPath(id, a)
	r.RecordPath(id, a)
	r.RecordPath(id, b)

	fresh := ps.Fresh(id)
	if len(fresh) != 2 {
		t.Fatalf("Fresh = %v, want 2 distinct endpoints", fresh)
	}
	seen := map[netip.AddrPort]bool{}
	for _, ep := range fresh {
		seen[ep] = true
	}
	if !seen[a] || !seen[b] {
		t.Fatalf("Fresh = %v, want both %v and %v", fresh, a, b)
	}
	if other := ps.Fresh(testSessionID(4)); len(other) != 0 {
		t.Fatalf("unrelated session has paths %v", other)
	}
}

type closingDevice struct {
	*fakeDevice
	closed   int
	closeErr error
}

func (d *closingDevice) Close() error { d.closed++; return d.closeErr }

// TestRouterClose pins that Close forwards to a device implementing io.Closer
// (propagating its error) and is a harmless no-op for one that does not.
func TestRouterClose(t *testing.T) {
	alloc := newTestAllocator(t, "10.0.0.0/24")
	boom := errors.New("boom")
	tests := []struct {
		name    string
		dev     Device
		wantErr error
		closed  func() int
	}{
		{name: "non-closer", dev: newFakeDevice()},
		{name: "closer ok", dev: &closingDevice{fakeDevice: newFakeDevice()}},
		{name: "closer error", dev: &closingDevice{fakeDevice: newFakeDevice(), closeErr: boom}, wantErr: boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRouter(alloc, NewPathSet(time.Second), tc.dev, fakeFramer{}, &fakeSink{}, nil)
			if err := r.Close(); !errors.Is(err, tc.wantErr) {
				t.Fatalf("Close = %v, want %v", err, tc.wantErr)
			}
			if cd, ok := tc.dev.(*closingDevice); ok && cd.closed != 1 {
				t.Fatalf("device Close called %d times, want 1", cd.closed)
			}
		})
	}
}

// TestRouterNilMetricsUsesNoop pins that a nil Metrics is replaced by the no-op
// sink so every decision path (forward, drop) runs without a nil dereference.
func TestRouterNilMetricsUsesNoop(t *testing.T) {
	alloc := newTestAllocator(t, "10.0.0.0/24")
	dev := newFakeDevice()
	r := NewRouter(alloc, NewPathSet(time.Second), dev, fakeFramer{}, &fakeSink{}, nil)
	id := testSessionID(5)

	if err := r.Egress(id, buildIPv4(netip.MustParseAddr("10.0.0.2"), netip.MustParseAddr("203.0.113.1"))); !errors.Is(err, ErrNoLease) {
		t.Fatalf("Egress without lease = %v, want ErrNoLease", err)
	}
	leased, err := r.Lease(id)
	if err != nil {
		t.Fatalf("Lease: %v", err)
	}
	if err := r.Egress(id, []byte{0x45}); err == nil {
		t.Fatal("Egress of truncated packet succeeded")
	}
	if err := r.Egress(id, buildIPv4(netip.MustParseAddr("10.0.0.200"), netip.MustParseAddr("203.0.113.1"))); !errors.Is(err, ErrReversePath) {
		t.Fatalf("spoofed Egress = %v, want ErrReversePath", err)
	}
	if err := r.Egress(id, buildIPv4(leased, netip.MustParseAddr("203.0.113.1"))); err != nil {
		t.Fatalf("Egress: %v", err)
	}
	if n := len(dev.writtenPackets()); n != 1 {
		t.Fatalf("device got %d packets, want 1", n)
	}
	var m noopMetrics
	m.Forwarded(1)
	m.Dropped("x")
	m.Returned(1)
}
