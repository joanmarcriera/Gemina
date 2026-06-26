package dedup

import (
	"errors"
	"sync"

	"continuity-vpn/internal/protocol"
)

var ErrInvalidCapacity = errors.New("dedup window capacity must be positive")

type PathID string

func (id PathID) Valid() bool {
	return id != ""
}

// Decision is the outcome of a Window.Observe call for one (PacketID, PathID)
// pair. It covers only the FIFO dedup layer — it has no knowledge of packet
// ordering or sequence numbers and therefore has no Stale value.
//
// Intentionally distinct from gateway.Decision (which adds a Rejected state for
// probe-parse failures at the datagram layer) and from ReplayDecision (which
// adds ReplayStale for RFC 6479 sequence-number anti-replay). The three types
// operate at different abstraction levels and have different zero values; a
// shared type would couple unrelated layers.
type Decision uint8

const (
	DecisionInvalid Decision = iota
	DecisionFirstCopy
	DecisionDuplicate
)

func (decision Decision) String() string {
	switch decision {
	case DecisionFirstCopy:
		return "first-copy"
	case DecisionDuplicate:
		return "duplicate"
	default:
		return "invalid"
	}
}

type Result struct {
	Decision  Decision
	ID        protocol.PacketID
	Path      PathID
	FirstPath PathID
	CopyCount int
}

type record struct {
	firstPath PathID
	copyCount int
}

// Window suppresses duplicate packet copies, keeping at most capacity packet
// IDs. Eviction is first-in, first-out using a fixed-size ring buffer so each
// observation is O(1); it is intentionally not a sequence-aware replay window.
type Window struct {
	mu       sync.Mutex
	capacity int
	seen     map[protocol.PacketID]record
	ring     []protocol.PacketID
	head     int // index of the oldest live entry
	count    int // number of live entries, always equal to len(seen)
}

func NewWindow(capacity int) (*Window, error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}
	return &Window{
		capacity: capacity,
		seen:     make(map[protocol.PacketID]record, capacity),
		ring:     make([]protocol.PacketID, capacity),
	}, nil
}

func (window *Window) Observe(id protocol.PacketID, path PathID) Result {
	if window == nil || !id.Valid() || !path.Valid() {
		return Result{Decision: DecisionInvalid, ID: id, Path: path}
	}

	window.mu.Lock()
	defer window.mu.Unlock()

	if existing, ok := window.seen[id]; ok {
		existing.copyCount++
		window.seen[id] = existing
		return Result{
			Decision:  DecisionDuplicate,
			ID:        id,
			Path:      path,
			FirstPath: existing.firstPath,
			CopyCount: existing.copyCount,
		}
	}

	window.evictIfFull()
	window.seen[id] = record{firstPath: path, copyCount: 1}
	window.pushID(id)

	return Result{
		Decision:  DecisionFirstCopy,
		ID:        id,
		Path:      path,
		FirstPath: path,
		CopyCount: 1,
	}
}

func (window *Window) Len() int {
	if window == nil {
		return 0
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	return len(window.seen)
}

func (window *Window) Capacity() int {
	if window == nil {
		return 0
	}
	return window.capacity
}

// pushID records id as the newest entry in the ring buffer. The caller must
// hold window.mu and must have evicted first if the window was full.
func (window *Window) pushID(id protocol.PacketID) {
	tail := (window.head + window.count) % window.capacity
	window.ring[tail] = id
	window.count++
}

// evictIfFull removes the oldest entry when the window is at capacity so a new
// packet ID can be inserted. The caller must hold window.mu.
func (window *Window) evictIfFull() {
	if window.count < window.capacity {
		return
	}

	oldest := window.ring[window.head]
	delete(window.seen, oldest)
	window.head = (window.head + 1) % window.capacity
	window.count--
}
