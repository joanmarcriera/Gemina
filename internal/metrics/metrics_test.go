package metrics

import (
	"strings"
	"sync"
	"testing"
)

func TestCounterIncAndRender(t *testing.T) {
	reg := NewRegistry()
	packets := reg.Counter("continuity_packets_total", "probe decisions", "decision", "path")
	packets.Inc("first-copy", "wi-fi")
	packets.Inc("first-copy", "wi-fi")
	packets.Add(3, "duplicate", "android-usb-tether")

	out := reg.Render()
	wantLines := []string{
		"# HELP continuity_packets_total probe decisions",
		"# TYPE continuity_packets_total counter",
		`continuity_packets_total{decision="first-copy",path="wi-fi"} 2`,
		`continuity_packets_total{decision="duplicate",path="android-usb-tether"} 3`,
	}
	for _, line := range wantLines {
		if !strings.Contains(out, line) {
			t.Errorf("render missing line %q in:\n%s", line, out)
		}
	}
}

func TestGaugeSet(t *testing.T) {
	reg := NewRegistry()
	g := reg.Gauge("continuity_active_sessions", "live sessions")
	g.Set(5)
	g.Inc()
	g.Dec()
	g.Dec()

	out := reg.Render()
	if !strings.Contains(out, "# TYPE continuity_active_sessions gauge") {
		t.Errorf("missing gauge TYPE: %s", out)
	}
	if !strings.Contains(out, "continuity_active_sessions 4") {
		t.Errorf("gauge value wrong: %s", out)
	}
}

func TestRenderIsStable(t *testing.T) {
	reg := NewRegistry()
	c := reg.Counter("m_total", "h", "k")
	c.Inc("b")
	c.Inc("a")
	// Series must render in a deterministic (sorted) order so diffs are stable.
	out1 := reg.Render()
	out2 := reg.Render()
	if out1 != out2 {
		t.Fatal("render is not deterministic")
	}
	ai := strings.Index(out1, `m_total{k="a"}`)
	bi := strings.Index(out1, `m_total{k="b"}`)
	if ai < 0 || bi < 0 || ai > bi {
		t.Fatalf("series not sorted by label value:\n%s", out1)
	}
}

func TestCounterPanicsOnLabelArityMismatch(t *testing.T) {
	reg := NewRegistry()
	c := reg.Counter("m_total", "h", "decision", "path")
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on wrong label arity")
		}
	}()
	c.Inc("only-one")
}

func TestLabelValuesAreEscaped(t *testing.T) {
	reg := NewRegistry()
	c := reg.Counter("m_total", "h", "reason")
	c.Inc(`a"b\c`)
	out := reg.Render()
	if !strings.Contains(out, `reason="a\"b\\c"`) {
		t.Fatalf("label value not escaped: %s", out)
	}
}

func TestConcurrentIncIsRaceFree(t *testing.T) {
	reg := NewRegistry()
	c := reg.Counter("hits_total", "h", "path")

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				c.Inc("wi-fi")
			}
		}()
	}
	wg.Wait()

	if !strings.Contains(reg.Render(), `hits_total{path="wi-fi"} 8000`) {
		t.Fatalf("concurrent count wrong:\n%s", reg.Render())
	}
}
