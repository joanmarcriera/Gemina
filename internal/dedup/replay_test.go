package dedup

import (
	"errors"
	"math"
	"math/bits"
	"sync"
	"testing"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

// --- unit tests ---------------------------------------------------------------

func TestNewReplayWindowRequiresPositiveWidth(t *testing.T) {
	if _, err := NewReplayWindow(0); !errors.Is(err, ErrInvalidCapacity) {
		t.Fatalf("NewReplayWindow(0) error = %v, want ErrInvalidCapacity", err)
	}
	if _, err := NewReplayWindow(-1); !errors.Is(err, ErrInvalidCapacity) {
		t.Fatalf("NewReplayWindow(-1) error = %v, want ErrInvalidCapacity", err)
	}
}

func TestReplayWindowFirstCopy(t *testing.T) {
	rw := newTestReplayWindow(t, 64)

	if got := rw.Observe(1); got != ReplayFirstCopy {
		t.Fatalf("first observation of n=1: got %s, want first-copy", got)
	}
}

func TestReplayWindowCrossPathDuplicate(t *testing.T) {
	// Same packet number arriving over two paths must yield first-copy then duplicate.
	rw := newTestReplayWindow(t, 64)

	if got := rw.Observe(42); got != ReplayFirstCopy {
		t.Fatalf("first copy of n=42: got %s", got)
	}
	if got := rw.Observe(42); got != ReplayDuplicate {
		t.Fatalf("second copy of n=42 (different path): got %s, want duplicate", got)
	}
}

func TestReplayWindowInWindowOutOfOrderAccepted(t *testing.T) {
	rw := newTestReplayWindow(t, 64)

	// Advance the window by delivering packets 1..10 in order.
	for n := protocol.PacketNumber(1); n <= 10; n++ {
		if got := rw.Observe(n); got != ReplayFirstCopy {
			t.Fatalf("setup: n=%d got %s", n, got)
		}
	}

	// n=5 is within the window (10-5=5 < 64) and was already seen.
	if got := rw.Observe(5); got != ReplayDuplicate {
		t.Fatalf("in-window duplicate of n=5: got %s, want duplicate", got)
	}

	// Deliver a packet that arrived out of order and was never seen.
	// Send 1..10 then artificially skip delivering n=11, send n=12.
	// After n=12 arrives, n=11 is still in-window and unseen.
	if got := rw.Observe(12); got != ReplayFirstCopy {
		t.Fatalf("out-of-order n=12: got %s", got)
	}
	// n=11 is between last-width..last: in-window, not yet seen.
	if got := rw.Observe(11); got != ReplayFirstCopy {
		t.Fatalf("out-of-order in-window n=11: got %s, want first-copy", got)
	}
	// Now n=11 is a duplicate.
	if got := rw.Observe(11); got != ReplayDuplicate {
		t.Fatalf("re-observe n=11: got %s, want duplicate", got)
	}
}

func TestReplayWindowStaleRejection(t *testing.T) {
	const width = 64
	rw := newTestReplayWindow(t, width)

	// Advance the window well past n=1 so that n=1 is stale.
	if got := rw.Observe(1); got != ReplayFirstCopy {
		t.Fatalf("n=1 first: %s", got)
	}
	// Advance by exactly width to push n=1 just out of the window.
	target := protocol.PacketNumber(1 + width)
	if got := rw.Observe(target); got != ReplayFirstCopy {
		t.Fatalf("advance to n=%d: got %s", target, got)
	}
	// n=1 is now at distance width from last — stale.
	if got := rw.Observe(1); got != ReplayStale {
		t.Fatalf("n=1 after advancing window by width: got %s, want stale", got)
	}
}

func TestReplayWindowInvalidZero(t *testing.T) {
	rw := newTestReplayWindow(t, 64)
	if got := rw.Observe(0); got != ReplayInvalid {
		t.Fatalf("n=0: got %s, want invalid", got)
	}
}

func TestReplayWindowDecisionStrings(t *testing.T) {
	cases := []struct {
		d    ReplayDecision
		want string
	}{
		{ReplayInvalid, "invalid"},
		{ReplayFirstCopy, "first-copy"},
		{ReplayDuplicate, "duplicate"},
		{ReplayStale, "stale"},
		{ReplayDecision(99), "invalid"}, // unknown sentinel
	}
	for _, tc := range cases {
		if got := tc.d.String(); got != tc.want {
			t.Errorf("ReplayDecision(%d).String() = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestReplayWindowWidthRoundedUp(t *testing.T) {
	// A width of 1 must be rounded up to 64 (one full uint64 word).
	rw := newTestReplayWindow(t, 1)
	if rw.Width() < 64 || rw.Width()%64 != 0 {
		t.Fatalf("Width() = %d, want multiple of 64 >= 64", rw.Width())
	}
}

func TestReplayWindowLargeJumpClearsWindow(t *testing.T) {
	const width = 64
	rw := newTestReplayWindow(t, width)

	// Fill the window.
	for n := protocol.PacketNumber(1); n <= 64; n++ {
		rw.Observe(n)
	}
	// Jump more than width ahead — all existing bits must be cleared.
	rw.Observe(200)
	// Any number in [1..64] is now stale (200-64=136 > 64 from 200).
	if got := rw.Observe(1); got != ReplayStale {
		t.Fatalf("after large jump, n=1: got %s, want stale", got)
	}
}

func TestReplayWindowConcurrentOneFirstCopyPerPacket(t *testing.T) {
	rw := newTestReplayWindow(t, 1024)
	const copies = 64
	const n = protocol.PacketNumber(7)

	results := make(chan ReplayDecision, copies)
	var wg sync.WaitGroup
	for i := 0; i < copies; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- rw.Observe(n)
		}()
	}
	wg.Wait()
	close(results)

	firstCopies, duplicates := 0, 0
	for d := range results {
		switch d {
		case ReplayFirstCopy:
			firstCopies++
		case ReplayDuplicate:
			duplicates++
		default:
			t.Fatalf("unexpected decision %s", d)
		}
	}
	if firstCopies != 1 {
		t.Fatalf("first-copy count = %d, want 1", firstCopies)
	}
	if duplicates != copies-1 {
		t.Fatalf("duplicate count = %d, want %d", duplicates, copies-1)
	}
}

// TestReplayWindowRollover confirms that packet numbers near the uint64 maximum
// do not cause panics or incorrect decisions.
func TestReplayWindowRollover(t *testing.T) {
	rw := newTestReplayWindow(t, 128)

	// Seed a high packet number, then check that numbers just below it behave
	// correctly as in-window out-of-order arrivals.
	const base = protocol.PacketNumber(math.MaxUint64 - 1000)

	if got := rw.Observe(base); got != ReplayFirstCopy {
		t.Fatalf("base n: got %s", got)
	}
	// One ahead: still valid (no rollover arithmetic should panic).
	next := base + 1
	if got := rw.Observe(next); got != ReplayFirstCopy {
		t.Fatalf("next n: got %s", got)
	}
	// Duplicate of base.
	if got := rw.Observe(base); got != ReplayDuplicate {
		t.Fatalf("duplicate of base: got %s", got)
	}
	// Far behind base: stale.
	if got := rw.Observe(1); got != ReplayStale {
		t.Fatalf("n=1 near max: got %s, want stale", got)
	}
}

// --- fuzz test ----------------------------------------------------------------

// FuzzReplayWindow drives a sequence of PacketNumbers through both ReplayWindow
// and a brute-force reference model, asserting identical decisions at every
// step. Run with: go test -run x -fuzz FuzzReplayWindow -fuzztime 20s ./internal/dedup/
func FuzzReplayWindow(f *testing.F) {
	// Seed corpus: width seed + byte stream of 8-byte packet numbers.
	f.Add(uint8(1), []byte{0, 0, 0, 0, 0, 0, 0, 1})
	f.Add(uint8(4), []byte{0, 0, 0, 0, 0, 0, 0, 1,
		0, 0, 0, 0, 0, 0, 0, 1, // duplicate
		0, 0, 0, 0, 0, 0, 0, 2})
	f.Add(uint8(8), make([]byte, 64))
	f.Add(uint8(15), []byte{
		0, 0, 0, 0, 0, 0, 0, 100,
		0, 0, 0, 0, 0, 0, 0, 1, // stale after 100
		0, 0, 0, 0, 0, 0, 0, 200,
		0, 0, 0, 0, 0, 0, 0, 50, // in-window out-of-order
	})

	f.Fuzz(func(t *testing.T, widthSeed uint8, ops []byte) {
		// Width 1..16 forces small windows so stale/duplicate paths are hit often.
		width := int(widthSeed%16) + 1

		rw, err := NewReplayWindow(width)
		if err != nil {
			t.Fatalf("NewReplayWindow(%d): %v", width, err)
		}
		ref := newReplayModel(rw.Width()) // use effective width for the model

		// Consume ops in 8-byte chunks; a zero PacketNumber is intentionally
		// exercised to cover the invalid branch.
		for i := 0; i+8 <= len(ops); i += 8 {
			var raw uint64
			for j := 0; j < 8; j++ {
				raw = raw<<8 | uint64(ops[i+j])
			}
			n := protocol.PacketNumber(raw)

			got := rw.Observe(n)
			want := ref.observe(n)

			if got != want {
				t.Fatalf("Observe(%d) = %s, model says %s (last=%d width=%d)",
					n, got, want, rw.last, rw.width)
			}
		}
	})
}

// --- benchmark ----------------------------------------------------------------

// BenchmarkReplayWindowObserve measures steady-state first-copy throughput for
// sequential packet numbers (the common case). It must show 0 allocs/op.
func BenchmarkReplayWindowObserve(b *testing.B) {
	const width = 4096
	rw, err := NewReplayWindow(width)
	if err != nil {
		b.Fatalf("NewReplayWindow(%d): %v", width, err)
	}
	// Pre-warm: fill the window so we're in steady state.
	for n := protocol.PacketNumber(1); n <= width; n++ {
		rw.Observe(n)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rw.Observe(protocol.PacketNumber(width + 1 + i))
	}
}

// --- helpers ------------------------------------------------------------------

func newTestReplayWindow(t *testing.T, width int) *ReplayWindow {
	t.Helper()
	rw, err := NewReplayWindow(width)
	if err != nil {
		t.Fatalf("NewReplayWindow(%d): %v", width, err)
	}
	return rw
}

// replayModel is a reference implementation: it remembers the highest number
// seen and a set of all seen numbers within the current window. It is O(width)
// memory and O(1) per observation — simple enough to trust, slow enough not to
// ship. The width here is the effective (rounded-up) width from NewReplayWindow.
type replayModel struct {
	width   int
	highest protocol.PacketNumber
	seen    map[protocol.PacketNumber]bool
}

func newReplayModel(effectiveWidth int) *replayModel {
	return &replayModel{
		width: effectiveWidth,
		seen:  make(map[protocol.PacketNumber]bool),
	}
}

func (m *replayModel) observe(n protocol.PacketNumber) ReplayDecision {
	if n == 0 {
		return ReplayInvalid
	}
	if m.seen[n] {
		return ReplayDuplicate
	}
	// Check stale before updating highest.
	if m.highest >= protocol.PacketNumber(m.width) && n <= m.highest-protocol.PacketNumber(m.width) {
		return ReplayStale
	}
	// First copy: mark seen and update highest.
	m.seen[n] = true
	if n > m.highest {
		// Evict entries that have fallen below the new window floor.
		if n >= protocol.PacketNumber(m.width) {
			floor := n - protocol.PacketNumber(m.width) + 1
			for k := range m.seen {
				if k < floor {
					delete(m.seen, k)
				}
			}
		}
		m.highest = n
	}
	return ReplayFirstCopy
}

// bitsSet returns the count of set bits across all words; used in tests to
// verify that the window never leaks or double-counts positions.
func (w *ReplayWindow) bitsSet() int {
	var total int
	for _, wd := range w.words {
		total += bits.OnesCount64(wd)
	}
	return total
}
