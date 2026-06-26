package dedup

import (
	"testing"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

// FuzzWindowObserveModel drives Observe with an arbitrary sequence of
// operations derived from fuzz bytes and checks the Window against an
// independent reference model plus its own structural invariants. The id and
// path spaces are deliberately small so duplicates and evictions actually
// occur; some operations are intentionally invalid (zero session, zero packet
// number, empty path) to exercise the rejection path.
func FuzzWindowObserveModel(f *testing.F) {
	// Seed corpus: a capacity seed plus a byte stream of 3-byte operations
	// (session, number, path-selector).
	f.Add(uint8(0), []byte{1, 1, 1, 1, 1, 1}) // tiny window, repeats
	f.Add(uint8(3), []byte{1, 1, 1, 2, 2, 1, 3, 3, 2})
	f.Add(uint8(7), []byte{0, 1, 1, 1, 0, 1, 1, 1, 0}) // invalid ids/paths
	f.Add(uint8(15), make([]byte, 96))

	f.Fuzz(func(t *testing.T, capSeed uint8, ops []byte) {
		capacity := int(capSeed%16) + 1 // 1..16, small enough to force eviction

		window, err := NewWindow(capacity)
		if err != nil {
			t.Fatalf("NewWindow(%d): %v", capacity, err)
		}
		model := newWindowModel(capacity)

		paths := []PathID{"", "wifi", "usb-tether", "thunderbolt"}

		for i := 0; i+3 <= len(ops); i += 3 {
			id := fuzzPacketID(ops[i], ops[i+1])
			path := paths[int(ops[i+2])%len(paths)]
			valid := id.Valid() && path.Valid()

			got := window.Observe(id, path)

			if !valid {
				if got.Decision != DecisionInvalid {
					t.Fatalf("invalid op (id.valid=%v path=%q) decision = %s, want invalid",
						id.Valid(), path, got.Decision)
				}
				if window.Len() != len(model.entries) {
					t.Fatalf("invalid op changed window length to %d, model has %d",
						window.Len(), len(model.entries))
				}
				assertWindowInvariants(t, window, model)
				continue
			}

			wantDecision, wantFirstPath, wantCount := model.observe(id, path)

			if got.Decision != wantDecision {
				t.Fatalf("Observe(%s, %q) decision = %s, want %s",
					id, path, got.Decision, wantDecision)
			}
			if got.FirstPath != wantFirstPath {
				t.Fatalf("Observe(%s, %q) first path = %q, want %q",
					id, path, got.FirstPath, wantFirstPath)
			}
			if got.CopyCount != wantCount {
				t.Fatalf("Observe(%s, %q) copy count = %d, want %d",
					id, path, got.CopyCount, wantCount)
			}
			if got.ID != id || got.Path != path {
				t.Fatalf("Observe(%s, %q) echoed id/path = %s/%q", id, path, got.ID, got.Path)
			}

			assertWindowInvariants(t, window, model)
		}
	})
}

// assertWindowInvariants checks structural properties that must hold after every
// operation regardless of input.
func assertWindowInvariants(t *testing.T, window *Window, model *windowModel) {
	t.Helper()

	if window.Len() > window.capacity {
		t.Fatalf("window length %d exceeds capacity %d", window.Len(), window.capacity)
	}
	// The ring's live count must always track the map size (the code documents
	// this as an invariant); a drift means an eviction/insert bookkeeping bug.
	if window.count != len(window.seen) {
		t.Fatalf("ring count %d != len(seen) %d", window.count, len(window.seen))
	}
	if window.Len() != len(model.entries) {
		t.Fatalf("window length %d != model size %d", window.Len(), len(model.entries))
	}
}

// fuzzPacketID maps two fuzz bytes onto a packet id in a small space. A zero
// session byte yields a zero session (an invalid id), and a zero number byte
// yields packet number 0 (also invalid), so both rejection branches are reached.
func fuzzPacketID(sessionByte, numberByte byte) protocol.PacketID {
	var session protocol.SessionID
	for i := range session {
		session[i] = sessionByte
	}
	return protocol.PacketID{Session: session, Number: protocol.PacketNumber(numberByte)}
}

// windowModel is an independent reference implementation of the dedup window's
// observable behaviour: a FIFO of resident ids with their first path and copy
// count, evicting the oldest when full.
type windowModel struct {
	capacity int
	order    []protocol.PacketID // front = oldest resident id
	entries  map[protocol.PacketID]*windowModelEntry
}

type windowModelEntry struct {
	firstPath PathID
	copyCount int
}

func newWindowModel(capacity int) *windowModel {
	return &windowModel{
		capacity: capacity,
		entries:  make(map[protocol.PacketID]*windowModelEntry, capacity),
	}
}

// observe returns the decision, first path and copy count the Window must
// produce for a valid (id, path). The caller guarantees validity.
func (m *windowModel) observe(id protocol.PacketID, path PathID) (Decision, PathID, int) {
	if entry, ok := m.entries[id]; ok {
		entry.copyCount++
		return DecisionDuplicate, entry.firstPath, entry.copyCount
	}

	if len(m.order) == m.capacity {
		oldest := m.order[0]
		m.order = m.order[1:]
		delete(m.entries, oldest)
	}
	m.entries[id] = &windowModelEntry{firstPath: path, copyCount: 1}
	m.order = append(m.order, id)
	return DecisionFirstCopy, path, 1
}
