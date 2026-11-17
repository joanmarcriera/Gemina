package dedup

import "testing"

func TestDecisionString(t *testing.T) {
	tests := []struct {
		d    Decision
		want string
	}{
		{DecisionInvalid, "invalid"},
		{DecisionFirstCopy, "first-copy"},
		{DecisionDuplicate, "duplicate"},
		{Decision(255), "invalid"},
	}
	for _, tc := range tests {
		if got := tc.d.String(); got != tc.want {
			t.Errorf("Decision(%d).String() = %q, want %q", tc.d, got, tc.want)
		}
	}
}

// TestWindowCapacity pins that Capacity reports the configured bound and that a
// nil *Window is safe (returns 0), mirroring Len.
func TestWindowCapacity(t *testing.T) {
	for _, capacity := range []int{1, 2, 1024} {
		w, err := NewWindow(capacity)
		if err != nil {
			t.Fatalf("NewWindow(%d): %v", capacity, err)
		}
		if got := w.Capacity(); got != capacity {
			t.Errorf("Capacity() = %d, want %d", got, capacity)
		}
	}
	var nilWindow *Window
	if got := nilWindow.Capacity(); got != 0 {
		t.Errorf("nil Capacity() = %d, want 0", got)
	}
}
