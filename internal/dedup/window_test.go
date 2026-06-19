package dedup

import (
	"errors"
	"sync"
	"testing"

	"continuity-vpn/internal/protocol"
)

func TestNewWindowRequiresPositiveCapacity(t *testing.T) {
	if _, err := NewWindow(0); !errors.Is(err, ErrInvalidCapacity) {
		t.Fatalf("NewWindow(0) error = %v, want ErrInvalidCapacity", err)
	}
}

func TestWindowAcceptsFirstCopyAndRejectsDuplicates(t *testing.T) {
	window := newTestWindow(t, 8)
	id := testPacketID(1)

	first := window.Observe(id, "wifi")
	if first.Decision != DecisionFirstCopy {
		t.Fatalf("first decision = %s", first.Decision)
	}
	if first.FirstPath != "wifi" || first.CopyCount != 1 {
		t.Fatalf("first result = %+v", first)
	}

	duplicate := window.Observe(id, "usb-tether")
	if duplicate.Decision != DecisionDuplicate {
		t.Fatalf("duplicate decision = %s", duplicate.Decision)
	}
	if duplicate.FirstPath != "wifi" {
		t.Fatalf("duplicate first path = %q", duplicate.FirstPath)
	}
	if duplicate.CopyCount != 2 {
		t.Fatalf("duplicate copy count = %d", duplicate.CopyCount)
	}
}

func TestWindowRejectsInvalidObservations(t *testing.T) {
	window := newTestWindow(t, 8)

	tests := []struct {
		name string
		id   protocol.PacketID
		path PathID
	}{
		{name: "invalid packet", id: protocol.PacketID{}, path: "wifi"},
		{name: "empty path", id: testPacketID(1), path: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := window.Observe(tt.id, tt.path)
			if got.Decision != DecisionInvalid {
				t.Fatalf("Observe() decision = %s", got.Decision)
			}
			if window.Len() != 0 {
				t.Fatalf("invalid observation changed window length to %d", window.Len())
			}
		})
	}
}

func TestWindowEvictsOldestPacketIDWhenFull(t *testing.T) {
	window := newTestWindow(t, 2)
	firstID := testPacketID(1)

	if got := window.Observe(firstID, "wifi"); got.Decision != DecisionFirstCopy {
		t.Fatalf("first observe = %s", got.Decision)
	}
	window.Observe(testPacketID(2), "wifi")
	window.Observe(testPacketID(3), "usb-tether")

	if window.Len() != 2 {
		t.Fatalf("window length = %d", window.Len())
	}

	acceptedAgain := window.Observe(firstID, "usb-tether")
	if acceptedAgain.Decision != DecisionFirstCopy {
		t.Fatalf("evicted packet decision = %s, want first-copy", acceptedAgain.Decision)
	}
}

func TestWindowConcurrentDuplicatesHaveOneFirstCopy(t *testing.T) {
	window := newTestWindow(t, 32)
	id := testPacketID(9)
	const copies = 64

	results := make(chan Decision, copies)
	var wg sync.WaitGroup
	for i := 0; i < copies; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- window.Observe(id, "wifi").Decision
		}()
	}
	wg.Wait()
	close(results)

	firstCopies := 0
	duplicates := 0
	for decision := range results {
		switch decision {
		case DecisionFirstCopy:
			firstCopies++
		case DecisionDuplicate:
			duplicates++
		default:
			t.Fatalf("unexpected decision %s", decision)
		}
	}

	if firstCopies != 1 {
		t.Fatalf("first copies = %d, want 1", firstCopies)
	}
	if duplicates != copies-1 {
		t.Fatalf("duplicates = %d, want %d", duplicates, copies-1)
	}
}

func newTestWindow(t *testing.T, capacity int) *Window {
	t.Helper()
	window, err := NewWindow(capacity)
	if err != nil {
		t.Fatalf("NewWindow(%d): %v", capacity, err)
	}
	return window
}

func testPacketID(number protocol.PacketNumber) protocol.PacketID {
	var session protocol.SessionID
	for i := range session {
		session[i] = 0xa5
	}
	return protocol.PacketID{Session: session, Number: number}
}
