package main

import (
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	"continuity-vpn/pkg/clientcore"
)

// benchmarkStats summarises the latency and loss to one gateway target.
type benchmarkStats struct {
	Label       string
	Sent        int
	Received    int
	LossPercent float64
	Min         time.Duration
	Avg         time.Duration
	Max         time.Duration
}

// summarise turns the round-trip times of replies (out of `sent` pings) into a
// stats summary.
func summarise(label string, sent int, rtts []time.Duration) benchmarkStats {
	s := benchmarkStats{Label: label, Sent: sent, Received: len(rtts)}
	if sent > 0 {
		s.LossPercent = float64(sent-len(rtts)) / float64(sent) * 100
	}
	if len(rtts) == 0 {
		if sent > 0 {
			s.LossPercent = 100
		}
		return s
	}
	sort.Slice(rtts, func(i, j int) bool { return rtts[i] < rtts[j] })
	s.Min = rtts[0]
	s.Max = rtts[len(rtts)-1]
	var total time.Duration
	for _, d := range rtts {
		total += d
	}
	s.Avg = total / time.Duration(len(rtts))
	return s
}

// formatComparison renders a human table and a recommendation. The best target
// is the one with the least loss, then the lowest average latency — which is why
// a well-connected hosted gateway typically beats a home box you call back to
// while travelling.
func formatComparison(results []benchmarkStats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-16s %8s %8s %8s %8s\n", "gateway", "loss", "min", "avg", "max")
	for _, r := range results {
		fmt.Fprintf(&b, "%-16s %7.0f%% %8s %8s %8s\n",
			r.Label, r.LossPercent, dur(r.Min), dur(r.Avg), dur(r.Max))
	}
	if best := recommend(results); best != "" {
		fmt.Fprintf(&b, "\nRecommended: %s (lowest loss, then lowest latency).\n", best)
	}
	return b.String()
}

func recommend(results []benchmarkStats) string {
	best := ""
	var bestLoss float64 = 101
	var bestAvg time.Duration
	for _, r := range results {
		if r.Received == 0 {
			continue
		}
		if r.LossPercent < bestLoss || (r.LossPercent == bestLoss && r.Avg < bestAvg) {
			best, bestLoss, bestAvg = r.Label, r.LossPercent, r.Avg
		}
	}
	return best
}

func dur(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	return d.Round(time.Microsecond * 100).String()
}

// runBenchmark pings each -to target and prints a latency/loss comparison. It is
// the conversion lever: it shows when a maintained hosted gateway is faster and
// more reliable than calling back to a home server while on the move.
func runBenchmark(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("benchmark", flag.ContinueOnError)
	var targets multiFlag
	fs.Var(&targets, "to", "gateway host:port to test (repeatable, e.g. -to home:51820 -to hosted:51820)")
	count := fs.Int("count", 10, "pings per target")
	timeout := fs.Duration("timeout", time.Second, "per-ping reply timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("give at least one -to host:port")
	}

	results := make([]benchmarkStats, 0, len(targets))
	for _, target := range targets {
		rtts, err := pingTarget(target, *count, *timeout)
		if err != nil {
			fmt.Fprintf(out, "%s: %v\n", target, err)
		}
		results = append(results, summarise(target, *count, rtts))
	}
	_, err := io.WriteString(out, formatComparison(results))
	return err
}

func pingTarget(target string, count int, timeout time.Duration) ([]time.Duration, error) {
	conn, err := net.Dial("udp", target)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var rtts []time.Duration
	buf := make([]byte, 64)
	for i := 0; i < count; i++ {
		var n8 [8]byte
		_, _ = rand.Read(n8[:])
		nonce := binary.BigEndian.Uint64(n8[:])

		start := time.Now()
		if _, err := conn.Write(clientcore.EncodePing(nonce)); err != nil {
			continue
		}
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		m, err := conn.Read(buf)
		if err != nil {
			continue // lost
		}
		if isPong, got, derr := clientcore.DecodePing(buf[:m]); derr == nil && isPong && got == nonce {
			rtts = append(rtts, time.Since(start))
		}
	}
	return rtts, nil
}

// multiFlag collects a repeated string flag.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}
