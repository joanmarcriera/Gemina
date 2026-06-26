package dedup

import (
	"sync"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

// ReplayDecision is the outcome of a ReplayWindow.Observe call. It is specific
// to RFC 6479 / WireGuard-style sequence-number anti-replay and adds
// ReplayStale (packet number has fallen below the window floor) — a concept
// that neither dedup.Decision (FIFO dedup, no ordering) nor gateway.Decision
// (datagram-pipeline outcome) express.
//
// Intentionally distinct from dedup.Decision and gateway.Decision. All three
// operate at different abstraction levels and have different zero values; a
// shared type would couple unrelated layers.
type ReplayDecision uint8

const (
	// ReplayInvalid means n == 0, which the protocol prohibits.
	ReplayInvalid ReplayDecision = iota
	// ReplayFirstCopy means the packet number has not been seen before and is
	// within or ahead of the current window — deliver the payload.
	ReplayFirstCopy
	// ReplayDuplicate means the packet number arrived before and is still within
	// the window — the copy must be dropped.
	ReplayDuplicate
	// ReplayStale means the packet number is so far behind the leading edge that
	// it has fallen out of the window — treat as a replay attack and drop.
	ReplayStale
)

func (d ReplayDecision) String() string {
	switch d {
	case ReplayFirstCopy:
		return "first-copy"
	case ReplayDuplicate:
		return "duplicate"
	case ReplayStale:
		return "stale"
	default:
		return "invalid"
	}
}

// ReplayWindow implements RFC 6479 / WireGuard-style sliding-window anti-replay
// for a single session. The window keeps the highest packet number seen (last)
// and a bitmap of nw×64 bits arranged as a ring. Bit position for packet number
// n is (n % width), distributed across words as word = (n % width) / 64.
// When the window advances, only the words that now cover new positions are
// cleared — all others retain their bits.
//
// Memory is O(width) — stored as uint64 words with width rounded up to the
// nearest multiple of 64. Observe is O(1) amortised: advancing clears at most
// width/64 words per call, which is bounded. This matches the implementation
// strategy of WireGuard's replay.h and RFC 6479 §2.
//
// All exported methods are safe for concurrent use.
type ReplayWindow struct {
	mu    sync.Mutex
	last  protocol.PacketNumber // highest packet number observed; 0 = not yet set
	words []uint64              // ring bitmap; position for n is n % width
	width int                   // effective window width (multiple of 64)
	nw    int                   // len(words) == width/64
}

// NewReplayWindow creates a replay window wide enough to track at least width
// distinct packet numbers. width must be positive; ErrInvalidCapacity is
// returned otherwise. The effective internal width is rounded up to the nearest
// multiple of 64 to align with uint64 word boundaries.
func NewReplayWindow(width int) (*ReplayWindow, error) {
	if width <= 0 {
		return nil, ErrInvalidCapacity
	}
	nw := (width + 63) / 64
	actual := nw * 64
	return &ReplayWindow{
		words: make([]uint64, nw),
		width: actual,
		nw:    nw,
	}, nil
}

// Width returns the effective window width (may be larger than the value passed
// to NewReplayWindow due to 64-bit word alignment).
func (w *ReplayWindow) Width() int {
	return w.width
}

// Observe classifies one inbound packet number:
//
//   - n == 0:                   ReplayInvalid  (protocol prohibits zero)
//   - n > last:                 ReplayFirstCopy (advance window, mark seen)
//   - last-n < width:           ReplayFirstCopy or ReplayDuplicate (in window)
//   - last-n >= width (n<last): ReplayStale    (below window floor)
//
// It is safe for concurrent use.
func (w *ReplayWindow) Observe(n protocol.PacketNumber) ReplayDecision {
	if n == 0 {
		return ReplayInvalid
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.last == 0 {
		// First packet ever received.
		w.last = n
		w.setBit(n)
		return ReplayFirstCopy
	}

	if n > w.last {
		// Packet number is ahead of the window — advance and mark.
		w.advance(n)
		return ReplayFirstCopy
	}

	// n <= w.last: check distance. Unsigned subtraction is safe since n <= last.
	diff := w.last - n
	if diff >= protocol.PacketNumber(w.width) {
		return ReplayStale
	}

	// Within the window — accept first copy, reject duplicate.
	if w.testBit(n) {
		return ReplayDuplicate
	}
	w.setBit(n)
	return ReplayFirstCopy
}

// advance slides the window so that n becomes the new last. The ring bitmap
// maps packet number p to position p%width; when the window advances we must
// clear the positions [old_last+1 .. n] (mod width) because those slots will
// now be reused by new inbound packet numbers and must not carry stale bits.
// The caller must hold w.mu.
func (w *ReplayWindow) advance(n protocol.PacketNumber) {
	diff := n - w.last // always >= 1 because n > w.last

	if diff >= protocol.PacketNumber(w.width) {
		// The jump is wider than the window — every existing bit is stale.
		for i := range w.words {
			w.words[i] = 0
		}
	} else {
		// Clear ring positions [startPos .. endPos] going forward around the ring.
		// These are the positions that will be reused and must not retain old bits.
		startPos := int(uint64(w.last+1) % uint64(w.width))
		endPos := int(uint64(n) % uint64(w.width))

		if startPos <= endPos {
			// Contiguous range: clear positions [startPos .. endPos].
			w.clearRange(startPos, endPos)
		} else {
			// Wrapped range: clear [startPos .. width-1] then [0 .. endPos].
			w.clearRange(startPos, w.width-1)
			w.clearRange(0, endPos)
		}
	}

	w.last = n
	w.setBit(n)
}

// clearRange clears all bits in the closed position range [lo, hi] where both
// lo and hi are absolute positions in [0, width). lo must be <= hi. The caller
// must hold w.mu.
func (w *ReplayWindow) clearRange(lo, hi int) {
	loWord := lo / 64
	hiWord := hi / 64
	loBit := uint(lo % 64)
	hiBit := uint(hi % 64)

	if loWord == hiWord {
		// Single-word range.
		w.words[loWord] &^= wordMask(loBit, hiBit)
		return
	}

	// Partial low word: bits [loBit .. 63].
	w.words[loWord] &^= ^uint64(0) << loBit

	// Complete middle words.
	for wd := loWord + 1; wd < hiWord; wd++ {
		w.words[wd] = 0
	}

	// Partial high word: bits [0 .. hiBit].
	w.words[hiWord] &^= wordMask(0, hiBit)
}

// wordMask returns a uint64 with bits [lo..hi] set (both inclusive).
func wordMask(lo, hi uint) uint64 {
	if lo > hi {
		return 0
	}
	n := hi - lo + 1
	if n == 64 {
		return ^uint64(0)
	}
	return ((uint64(1) << n) - 1) << lo
}

// setBit marks packet number n as seen in the ring bitmap.
// The caller must hold w.mu.
func (w *ReplayWindow) setBit(n protocol.PacketNumber) {
	pos := uint64(n) % uint64(w.width)
	w.words[pos/64] |= uint64(1) << (pos % 64)
}

// testBit reports whether packet number n is set in the ring bitmap.
// The caller must hold w.mu.
func (w *ReplayWindow) testBit(n protocol.PacketNumber) bool {
	pos := uint64(n) % uint64(w.width)
	return w.words[pos/64]&(uint64(1)<<(pos%64)) != 0
}
