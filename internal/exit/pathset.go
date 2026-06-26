package exit

import (
	"net/netip"
	"sync"
	"time"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

// defaultPathTTL is how long a source endpoint stays "fresh" after its last
// observation. Short enough to stop sending to stale paths, long enough to
// survive transient interface flaps.
const defaultPathTTL = 30 * time.Second

// pathEntry holds one source endpoint and the last time we saw a datagram from
// it for a given session.
type pathEntry struct {
	addr     netip.AddrPort
	lastSeen time.Time
}

// PathSet tracks the set of source endpoints seen for each session, expiring
// entries that have not been observed within a configurable TTL. The return
// path duplicates to every fresh endpoint, so this set must stay minimal.
//
// PathSet is safe for concurrent use.
type PathSet struct {
	mu  sync.Mutex
	ttl time.Duration
	// now is the clock source; injectable so tests use a fake clock without
	// real time.Sleep calls.
	now   func() time.Time
	paths map[protocol.SessionID][]pathEntry
}

// NewPathSet creates a PathSet with the given TTL. A zero TTL uses the default
// of 30 seconds. Callers may replace ps.now with a fake clock in tests.
func NewPathSet(ttl time.Duration) *PathSet {
	if ttl <= 0 {
		ttl = defaultPathTTL
	}
	return &PathSet{
		ttl:   ttl,
		now:   time.Now,
		paths: make(map[protocol.SessionID][]pathEntry),
	}
}

// Record marks addr as a recently-seen source endpoint for id. If addr is
// already known for id its timestamp is refreshed; otherwise it is appended.
func (ps *PathSet) Record(id protocol.SessionID, addr netip.AddrPort) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := ps.now()
	entries := ps.paths[id]
	for i := range entries {
		if entries[i].addr == addr {
			entries[i].lastSeen = now
			ps.paths[id] = entries
			return
		}
	}
	ps.paths[id] = append(entries, pathEntry{addr: addr, lastSeen: now})
}

// Fresh returns all source endpoints for id that have been seen within the TTL.
// Expired entries are removed in-place so the slice stays compact over time.
// Returns nil if no fresh paths exist.
func (ps *PathSet) Fresh(id protocol.SessionID) []netip.AddrPort {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	entries := ps.paths[id]
	if len(entries) == 0 {
		return nil
	}

	cutoff := ps.now().Add(-ps.ttl)
	// Compact in-place: keep entries that are still fresh.
	live := entries[:0]
	for _, e := range entries {
		if e.lastSeen.After(cutoff) {
			live = append(live, e)
		}
	}
	if len(live) == 0 {
		delete(ps.paths, id)
		return nil
	}
	ps.paths[id] = live

	out := make([]netip.AddrPort, len(live))
	for i, e := range live {
		out[i] = e.addr
	}
	return out
}
