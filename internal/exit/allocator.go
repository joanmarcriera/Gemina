package exit

import (
	"errors"
	"net/netip"
	"sync"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

// ErrPoolExhausted is returned by Allocate when every usable address in the
// pool has already been leased to an active session.
var ErrPoolExhausted = errors.New("tunnel ip pool exhausted")

// ErrInvalidPool is returned by NewAllocator when the prefix is not usable as
// a tunnel address pool (not IPv4, or too small to hold any clients).
var ErrInvalidPool = errors.New("invalid tunnel ip pool: must be an ipv4 prefix with at least one usable host address")

// Allocator assigns tunnel IPv4 addresses from a pool to sessions. Each
// session always receives the same address (idempotent) until it is released.
// The network address, the gateway-host address (.1), and the broadcast address
// are never assigned to sessions.
//
// Allocator is safe for concurrent use.
type Allocator struct {
	mu     sync.Mutex
	prefix netip.Prefix
	// forward maps a session ID to its leased address.
	forward map[protocol.SessionID]netip.Addr
	// reverse maps a leased address back to its session, used to demux return
	// packets from the TUN device without a linear search.
	reverse map[netip.Addr]protocol.SessionID
	// free holds every address that is currently available. We use a slice to
	// keep allocation O(1) amortised; order does not matter for correctness.
	free []netip.Addr
}

// NewAllocator builds an Allocator over pool. pool must be an IPv4 /prefix;
// the network address, the first host (.1, reserved for the gateway), and the
// broadcast address are excluded from the usable set.
func NewAllocator(pool netip.Prefix) (*Allocator, error) {
	if !pool.IsValid() || pool.Addr().Is6() {
		return nil, ErrInvalidPool
	}
	// Normalise to the network address so Addr() is the base.
	base := pool.Masked()

	// Collect the usable host addresses: everything strictly between the gateway
	// host address (.1) and the broadcast address (the last address in the
	// prefix). We start at .1 and advance before use, so the first candidate is
	// .2; the broadcast is recognised because its successor leaves the prefix.
	var free []netip.Addr
	addr := base.Addr().Next() // .1 — reserved for the gateway host
	for {
		addr = addr.Next()
		if !base.Contains(addr) {
			break // advanced past the end of the prefix
		}
		if !base.Contains(addr.Next()) {
			break // addr is the broadcast (last) address; never assign it
		}
		free = append(free, addr)
	}

	if len(free) == 0 {
		return nil, ErrInvalidPool
	}

	return &Allocator{
		prefix:  base,
		forward: make(map[protocol.SessionID]netip.Addr),
		reverse: make(map[netip.Addr]protocol.SessionID),
		free:    free,
	}, nil
}

// Allocate returns the tunnel address leased to id. If id already has a lease
// the existing address is returned (idempotent). If no address is available,
// ErrPoolExhausted is returned.
func (a *Allocator) Allocate(id protocol.SessionID) (netip.Addr, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if addr, ok := a.forward[id]; ok {
		return addr, nil
	}
	if len(a.free) == 0 {
		return netip.Addr{}, ErrPoolExhausted
	}
	// Pop from the end — O(1), order irrelevant.
	addr := a.free[len(a.free)-1]
	a.free = a.free[:len(a.free)-1]

	a.forward[id] = addr
	a.reverse[addr] = id
	return addr, nil
}

// LeaseOf returns the address currently leased to id, without allocating one.
// It is the read-only lookup the egress reverse-path filter uses on the hot
// path: it never mints a new lease, so a session whose lease was released cannot
// silently acquire a different address mid-flow.
func (a *Allocator) LeaseOf(id protocol.SessionID) (netip.Addr, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	addr, ok := a.forward[id]
	return addr, ok
}

// Lookup returns the session that owns addr. Used by the return path to find
// which session an IP packet from the internet belongs to.
func (a *Allocator) Lookup(addr netip.Addr) (protocol.SessionID, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	id, ok := a.reverse[addr]
	return id, ok
}

// Release frees the lease held by id, returning its address to the pool.
// Releasing an id that has no lease is a no-op.
func (a *Allocator) Release(id protocol.SessionID) {
	a.mu.Lock()
	defer a.mu.Unlock()

	addr, ok := a.forward[id]
	if !ok {
		return
	}
	delete(a.forward, id)
	delete(a.reverse, addr)
	a.free = append(a.free, addr)
}
