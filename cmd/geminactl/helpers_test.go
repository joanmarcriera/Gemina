package main

import (
	"flag"
	"testing"
)

// TestMultiFlag pins that a repeated flag accumulates every value in order and
// renders comma-joined.
func TestMultiFlag(t *testing.T) {
	var m multiFlag
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.Var(&m, "target", "repeatable")
	if err := fs.Parse([]string{"-target", "a:1", "-target", "b:2", "-target", ""}); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(m) != 3 || m[0] != "a:1" || m[1] != "b:2" || m[2] != "" {
		t.Fatalf("multiFlag = %#v", m)
	}
	if got := m.String(); got != "a:1,b:2," {
		t.Fatalf("String() = %q", got)
	}
	var empty multiFlag
	if got := empty.String(); got != "" {
		t.Fatalf("empty String() = %q", got)
	}
}

func TestDuplicateNote(t *testing.T) {
	if got := duplicateNote(true); got != " (each duplicated)" {
		t.Fatalf("duplicateNote(true) = %q", got)
	}
	if got := duplicateNote(false); got != "" {
		t.Fatalf("duplicateNote(false) = %q", got)
	}
}

// TestRandomSession pins that probe session ids are never the invalid zero id
// and are not repeated across calls.
func TestRandomSession(t *testing.T) {
	seen := map[[16]byte]bool{}
	for i := 0; i < 64; i++ {
		s, err := randomSession()
		if err != nil {
			t.Fatalf("randomSession: %v", err)
		}
		if s.IsZero() {
			t.Fatal("randomSession returned the zero id")
		}
		if seen[s] {
			t.Fatal("randomSession repeated an id")
		}
		seen[s] = true
	}
}
