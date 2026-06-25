package exit

import (
	"net/netip"
	"testing"
	"time"
)

func TestPathSetRecordAndFresh(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ps := newTestPathSet(30*time.Second, func() time.Time { return now })

	id := testSessionID(1)
	ep := netip.MustParseAddrPort("192.0.2.1:51820")

	ps.Record(id, ep)

	fresh := ps.Fresh(id)
	if len(fresh) != 1 {
		t.Fatalf("Fresh returned %d paths, want 1", len(fresh))
	}
	if fresh[0] != ep {
		t.Fatalf("Fresh[0] = %s, want %s", fresh[0], ep)
	}
}

func TestPathSetMultipleEndpoints(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ps := newTestPathSet(30*time.Second, func() time.Time { return now })

	id := testSessionID(1)
	ep1 := netip.MustParseAddrPort("192.0.2.1:51820")   // TEST-NET-1 (Wi-Fi path)
	ep2 := netip.MustParseAddrPort("198.51.100.1:4500") // TEST-NET-2 (cellular path)

	ps.Record(id, ep1)
	ps.Record(id, ep2)

	fresh := ps.Fresh(id)
	if len(fresh) != 2 {
		t.Fatalf("Fresh returned %d paths, want 2", len(fresh))
	}
}

func TestPathSetExpiry(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ttl := 30 * time.Second
	ps := newTestPathSet(ttl, func() time.Time { return now })

	id := testSessionID(1)
	ep := netip.MustParseAddrPort("192.0.2.1:51820")
	ps.Record(id, ep)

	// Advance the clock past the TTL.
	now = now.Add(ttl + time.Second)

	fresh := ps.Fresh(id)
	if len(fresh) != 0 {
		t.Fatalf("Fresh returned %d paths after expiry, want 0", len(fresh))
	}
}

func TestPathSetPartialExpiry(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ttl := 30 * time.Second
	ps := newTestPathSet(ttl, func() time.Time { return now })

	id := testSessionID(1)
	ep1 := netip.MustParseAddrPort("192.0.2.1:51820")
	ep2 := netip.MustParseAddrPort("198.51.100.1:4500")

	ps.Record(id, ep1)

	// Advance half the TTL, then record ep2.
	now = now.Add(20 * time.Second)
	ps.Record(id, ep2)

	// Advance past ep1's TTL but not ep2's.
	now = now.Add(15 * time.Second) // total 35s from ep1, 15s from ep2

	fresh := ps.Fresh(id)
	if len(fresh) != 1 {
		t.Fatalf("Fresh returned %d paths, want 1 (only ep2)", len(fresh))
	}
	if fresh[0] != ep2 {
		t.Fatalf("Fresh[0] = %s, want %s", fresh[0], ep2)
	}
}

func TestPathSetRefreshKeepsEndpointAlive(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ttl := 30 * time.Second
	ps := newTestPathSet(ttl, func() time.Time { return now })

	id := testSessionID(1)
	ep := netip.MustParseAddrPort("192.0.2.1:51820")
	ps.Record(id, ep)

	// Advance 25 seconds, then refresh the endpoint.
	now = now.Add(25 * time.Second)
	ps.Record(id, ep) // refresh

	// Advance another 20 seconds: 45s from first Record, 20s from refresh.
	// Without refresh the entry would be expired; with refresh it must be fresh.
	now = now.Add(20 * time.Second)

	fresh := ps.Fresh(id)
	if len(fresh) != 1 {
		t.Fatalf("Fresh returned %d paths after refresh, want 1", len(fresh))
	}
}

func TestPathSetUnknownSessionReturnsNil(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ps := newTestPathSet(30*time.Second, func() time.Time { return now })

	fresh := ps.Fresh(testSessionID(99))
	if fresh != nil {
		t.Fatalf("Fresh for unknown session returned %v, want nil", fresh)
	}
}

func newTestPathSet(ttl time.Duration, clock func() time.Time) *PathSet {
	ps := NewPathSet(ttl)
	ps.now = clock
	return ps
}
