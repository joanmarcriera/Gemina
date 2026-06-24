package main

import (
	"strings"
	"testing"
	"time"
)

func TestSummariseRTTs(t *testing.T) {
	rtts := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
	}
	s := summarise("hosted", 4, rtts) // 4 sent, 3 replied -> 25% loss

	if s.Sent != 4 || s.Received != 3 {
		t.Fatalf("counts = %d/%d", s.Received, s.Sent)
	}
	if s.LossPercent < 24.9 || s.LossPercent > 25.1 {
		t.Fatalf("loss = %.1f, want 25", s.LossPercent)
	}
	if s.Min != 10*time.Millisecond || s.Max != 30*time.Millisecond {
		t.Fatalf("min/max = %v/%v", s.Min, s.Max)
	}
	if s.Avg != 20*time.Millisecond {
		t.Fatalf("avg = %v, want 20ms", s.Avg)
	}
}

func TestSummariseAllLost(t *testing.T) {
	s := summarise("home", 5, nil)
	if s.Received != 0 || s.LossPercent != 100 {
		t.Fatalf("all-lost summary wrong: %+v", s)
	}
}

func TestFormatBenchmarkComparison(t *testing.T) {
	results := []benchmarkStats{
		summarise("hosted", 10, repeat(20*time.Millisecond, 10)),
		summarise("home", 10, repeat(80*time.Millisecond, 7)),
	}
	out := formatComparison(results)
	if !strings.Contains(out, "hosted") || !strings.Contains(out, "home") {
		t.Fatalf("comparison missing labels:\n%s", out)
	}
	// The faster, more reliable target should be recommended.
	if !strings.Contains(strings.ToLower(out), "hosted") {
		t.Fatalf("recommendation missing:\n%s", out)
	}
}

func repeat(d time.Duration, n int) []time.Duration {
	out := make([]time.Duration, n)
	for i := range out {
		out[i] = d
	}
	return out
}
