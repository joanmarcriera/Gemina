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

type Window struct {
	mu       sync.Mutex
	capacity int
	seen     map[protocol.PacketID]record
	order    []protocol.PacketID
}

func NewWindow(capacity int) (*Window, error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}
	return &Window{
		capacity: capacity,
		seen:     make(map[protocol.PacketID]record, capacity),
		order:    make([]protocol.PacketID, 0, capacity),
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
	window.order = append(window.order, id)

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

func (window *Window) evictIfFull() {
	if len(window.seen) < window.capacity {
		return
	}

	oldest := window.order[0]
	delete(window.seen, oldest)
	copy(window.order, window.order[1:])
	window.order = window.order[:len(window.order)-1]
}
