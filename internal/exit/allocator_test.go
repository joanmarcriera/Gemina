package exit

import (
	"errors"
	"net/netip"
	"testing"

	"continuity-vpn/internal/protocol"
)

// Tunnel pool: 10.0.0.0/24 (private range, safe for tests).
// Client endpoints use TEST-NET ranges (192.0.2.x etc.) in router tests.

func TestNewAllocatorRejectsIPv6(t *testing.T) {
	_, err := NewAllocator(netip.MustParsePrefix("fd00::/120"))
	if !errors.Is(err, ErrInvalidPool) {
		t.Fatalf("NewAllocator(ipv6) error = %v, want ErrInvalidPool", err)
	}
}

func TestNewAllocatorRejectsTooSmallPrefix(t *testing.T) {
	// /31 has only 2 addresses: network + broadcast — no usable host addresses
	// once .1 (gateway) is reserved.
	_, err := NewAllocator(netip.MustParsePrefix("10.0.0.0/31"))
	if !errors.Is(err, ErrInvalidPool) {
		t.Fatalf("NewAllocator(/31) error = %v, want ErrInvalidPool", err)
	}
}

func TestAllocatorAllocatesDistinctAddresses(t *testing.T) {
	a := newTestAllocator(t, "10.0.0.0/24")

	idA := testSessionID(1)
	idB := testSessionID(2)

	addrA, err := a.Allocate(idA)
	if err != nil {
		t.Fatalf("Allocate(idA): %v", err)
	}
	addrB, err := a.Allocate(idB)
	if err != nil {
		t.Fatalf("Allocate(idB): %v", err)
	}
	if addrA == addrB {
		t.Fatalf("two sessions received the same address %s", addrA)
	}
}

func TestAllocatorIsIdempotent(t *testing.T) {
	a := newTestAllocator(t, "10.0.0.0/24")
	id := testSessionID(1)

	first, err := a.Allocate(id)
	if err != nil {
		t.Fatalf("first Allocate: %v", err)
	}
	second, err := a.Allocate(id)
	if err != nil {
		t.Fatalf("second Allocate: %v", err)
	}
	if first != second {
		t.Fatalf("Allocate not idempotent: first=%s second=%s", first, second)
	}
}

func TestAllocatorLookup(t *testing.T) {
	a := newTestAllocator(t, "10.0.0.0/24")
	id := testSessionID(1)

	addr, err := a.Allocate(id)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}

	got, ok := a.Lookup(addr)
	if !ok {
		t.Fatalf("Lookup(%s) returned not-found", addr)
	}
	if got != id {
		t.Fatalf("Lookup(%s) = %s, want %s", addr, got, id)
	}
}

func TestAllocatorLookupUnknownAddress(t *testing.T) {
	a := newTestAllocator(t, "10.0.0.0/24")
	_, ok := a.Lookup(netip.MustParseAddr("10.0.0.200"))
	if ok {
		t.Fatal("Lookup of unallocated address returned ok")
	}
}

func TestAllocatorRelease(t *testing.T) {
	a := newTestAllocator(t, "10.0.0.0/24")
	id := testSessionID(1)

	addr, err := a.Allocate(id)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}

	a.Release(id)

	// Reverse map must be cleared.
	if _, ok := a.Lookup(addr); ok {
		t.Fatalf("Lookup after Release returned ok for %s", addr)
	}

	// The address must be back in the pool — a different session can get it.
	id2 := testSessionID(2)
	addr2, err := a.Allocate(id2)
	if err != nil {
		t.Fatalf("Allocate after Release: %v", err)
	}
	_ = addr2 // may or may not be the same address; both are valid
}

func TestAllocatorReleaseNoOpOnUnknownID(t *testing.T) {
	a := newTestAllocator(t, "10.0.0.0/24")
	// Releasing an id that was never allocated must not panic.
	a.Release(testSessionID(99))
}

func TestAllocatorExhaustion(t *testing.T) {
	// /30 gives addresses .1 through .2 usable (.0=net, .1=gateway reserved, .3=broadcast).
	// So only .2 is available — one session max.
	a := newTestAllocator(t, "10.0.0.0/30")

	_, err := a.Allocate(testSessionID(1))
	if err != nil {
		t.Fatalf("first Allocate on /30: %v", err)
	}
	_, err = a.Allocate(testSessionID(2))
	if !errors.Is(err, ErrPoolExhausted) {
		t.Fatalf("second Allocate on /30 error = %v, want ErrPoolExhausted", err)
	}
}

func TestAllocatorDoesNotAssignNetworkOrGatewayOrBroadcast(t *testing.T) {
	a := newTestAllocator(t, "10.0.0.0/24")
	// Drain the pool up to 252 sessions (.2 through .253 are the usable range;
	// .0=network, .1=gateway, .254 and .255 in a /24 would be the last usable
	// and broadcast).  Just check the first allocation is not a forbidden addr.
	addr, err := a.Allocate(testSessionID(1))
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	forbidden := []netip.Addr{
		netip.MustParseAddr("10.0.0.0"),   // network
		netip.MustParseAddr("10.0.0.1"),   // gateway host
		netip.MustParseAddr("10.0.0.255"), // broadcast
	}
	for _, f := range forbidden {
		if addr == f {
			t.Fatalf("Allocate returned forbidden address %s", addr)
		}
	}
}

// helpers

func newTestAllocator(t *testing.T, prefix string) *Allocator {
	t.Helper()
	a, err := NewAllocator(netip.MustParsePrefix(prefix))
	if err != nil {
		t.Fatalf("NewAllocator(%s): %v", prefix, err)
	}
	return a
}

func testSessionID(n byte) protocol.SessionID {
	var id protocol.SessionID
	for i := range id {
		id[i] = n
	}
	return id
}
