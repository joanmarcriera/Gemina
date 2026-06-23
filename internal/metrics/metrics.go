// Package metrics is a small, dependency-free metrics registry that renders the
// Prometheus text exposition format. It carries counters and gauges with labels
// — enough for the gateway's failover signal — without pulling in a Prometheus
// client library.
//
// Redaction is the caller's responsibility and the whole point of the design:
// label *values* must be fixed coarse tokens (e.g. "wi-fi", "first-copy"), never
// a session id, IP, MAC or other host identifier. The registry escapes label
// values for the exposition format but does not otherwise constrain them, so
// call sites must only pass vocabulary tokens (see observability/METRICS.md).
package metrics

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Registry holds the counters and gauges for one process and renders them.
type Registry struct {
	mu       sync.Mutex
	counters []*CounterVec
	gauges   []*GaugeVec
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Counter registers a counter metric with the given label names and returns its
// vector. Call it once per metric at start-up.
func (r *Registry) Counter(name, help string, labelNames ...string) *CounterVec {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := &CounterVec{metric: metric{name: name, help: help, labelNames: labelNames}}
	r.counters = append(r.counters, c)
	return c
}

// Gauge registers a gauge metric and returns its vector.
func (r *Registry) Gauge(name, help string, labelNames ...string) *GaugeVec {
	r.mu.Lock()
	defer r.mu.Unlock()
	g := &GaugeVec{metric: metric{name: name, help: help, labelNames: labelNames}}
	r.gauges = append(r.gauges, g)
	return g
}

// Render produces the Prometheus text exposition format, deterministically
// ordered so output diffs are stable.
func (r *Registry) Render() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var b strings.Builder
	for _, c := range r.counters {
		c.render(&b, "counter")
	}
	for _, g := range r.gauges {
		g.render(&b, "gauge")
	}
	return b.String()
}

// metric is the shared state of a counter or gauge vector.
type metric struct {
	name       string
	help       string
	labelNames []string

	mu     sync.Mutex
	series map[string]*sample // keyed by the rendered label set
}

type sample struct {
	labelValues []string
	value       atomic.Int64
}

func (m *metric) sampleFor(labelValues []string) *sample {
	if len(labelValues) != len(m.labelNames) {
		panic("metrics: label value count does not match label names for " + m.name)
	}
	key := strings.Join(labelValues, "\x00")

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.series == nil {
		m.series = make(map[string]*sample)
	}
	s, ok := m.series[key]
	if !ok {
		s = &sample{labelValues: append([]string(nil), labelValues...)}
		m.series[key] = s
	}
	return s
}

func (m *metric) render(b *strings.Builder, typ string) {
	b.WriteString("# HELP " + m.name + " " + m.help + "\n")
	b.WriteString("# TYPE " + m.name + " " + typ + "\n")

	m.mu.Lock()
	samples := make([]*sample, 0, len(m.series))
	for _, s := range m.series {
		samples = append(samples, s)
	}
	m.mu.Unlock()

	sort.Slice(samples, func(i, j int) bool {
		return less(samples[i].labelValues, samples[j].labelValues)
	})

	for _, s := range samples {
		b.WriteString(m.name)
		writeLabels(b, m.labelNames, s.labelValues)
		b.WriteByte(' ')
		b.WriteString(strconv.FormatInt(s.value.Load(), 10))
		b.WriteByte('\n')
	}
}

// CounterVec is a monotonically increasing metric, optionally labelled.
type CounterVec struct{ metric }

// Inc adds 1 to the series identified by labelValues.
func (c *CounterVec) Inc(labelValues ...string) { c.Add(1, labelValues...) }

// Add adds delta (which should be non-negative for a counter) to the series.
func (c *CounterVec) Add(delta int64, labelValues ...string) {
	c.sampleFor(labelValues).value.Add(delta)
}

// GaugeVec is a metric that can go up and down, optionally labelled.
type GaugeVec struct{ metric }

// Set sets the series value.
func (g *GaugeVec) Set(v int64, labelValues ...string) {
	g.sampleFor(labelValues).value.Store(v)
}

// Inc adds 1 to the series.
func (g *GaugeVec) Inc(labelValues ...string) { g.sampleFor(labelValues).value.Add(1) }

// Dec subtracts 1 from the series.
func (g *GaugeVec) Dec(labelValues ...string) { g.sampleFor(labelValues).value.Add(-1) }

func writeLabels(b *strings.Builder, names, values []string) {
	if len(names) == 0 {
		return
	}
	b.WriteByte('{')
	for i, name := range names {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(name)
		b.WriteString(`="`)
		b.WriteString(escape(values[i]))
		b.WriteByte('"')
	}
	b.WriteByte('}')
}

// escape applies the Prometheus label-value escaping rules: backslash, double
// quote and newline.
func escape(v string) string {
	if !strings.ContainsAny(v, "\\\"\n") {
		return v
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return r.Replace(v)
}

func less(a, b []string) bool {
	for i := range a {
		if i >= len(b) {
			return false
		}
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}
