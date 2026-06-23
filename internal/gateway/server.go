// Package gateway implements the Stage-1 probe gateway: a UDP listener that
// deduplicates probe copies arriving over multiple client paths and logs each
// decision as a redacted structured record. It is a feasibility server for the
// dual-path probe, not production VPN transport.
package gateway

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync/atomic"

	"continuity-vpn/internal/dedup"
	"continuity-vpn/internal/protocol"
)

// maxDatagram bounds a single read. Probes are ProbeWireSize; the slack absorbs
// trailing bytes without unbounded allocation on a public listener.
const maxDatagram = 1500

// summaryEvery is how many processed datagrams between Info-level summary logs.
// Per-packet detail is logged at Debug; under a high probe rate this keeps the
// default (Info) hot path free of per-packet logging while still reporting
// progress.
const summaryEvery = 1000

// Decision is the outcome of handling one datagram.
type Decision uint8

const (
	DecisionRejected Decision = iota // not a valid probe
	DecisionFirstCopy
	DecisionDuplicate
)

func (d Decision) String() string {
	switch d {
	case DecisionFirstCopy:
		return "first-copy"
	case DecisionDuplicate:
		return "duplicate"
	default:
		return "rejected"
	}
}

// Record is what Handle decided about one datagram. It carries only coarse,
// non-identifying fields — never a source address.
type Record struct {
	Decision  Decision
	Path      protocol.PathTag
	FirstPath protocol.PathTag
	CopyCount int
}

// Stats is a snapshot of cumulative counters.
type Stats struct {
	FirstCopies uint64
	Duplicates  uint64
	Rejected    uint64
}

// Server deduplicates probe datagrams and logs redacted decisions.
type Server struct {
	window *dedup.Window
	logger *slog.Logger

	firstCopies atomic.Uint64
	duplicates  atomic.Uint64
	rejected    atomic.Uint64
}

// NewServer builds a gateway with a dedup window of the given capacity. A nil
// logger discards logs.
func NewServer(capacity int, logger *slog.Logger) (*Server, error) {
	window, err := dedup.NewWindow(capacity)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Server{window: window, logger: logger}, nil
}

// Handle processes one datagram: decode, deduplicate, count, and log. It never
// receives or logs the source address, so a source identifier cannot leak by
// construction. Malformed datagrams are rejected without affecting the window.
func (s *Server) Handle(datagram []byte) Record {
	probe, err := protocol.UnmarshalProbe(datagram)
	if err != nil {
		s.rejected.Add(1)
		if s.logger.Enabled(context.Background(), slog.LevelDebug) {
			s.logger.Debug("probe rejected", "decision", DecisionRejected.String(), "reason", err.Error())
		}
		return Record{Decision: DecisionRejected}
	}

	result := s.window.Observe(probe.ID, dedup.PathID(probe.Path.String()))

	rec := Record{
		Path:      probe.Path,
		FirstPath: probe.Path,
		CopyCount: result.CopyCount,
	}
	switch result.Decision {
	case dedup.DecisionFirstCopy:
		s.firstCopies.Add(1)
		rec.Decision = DecisionFirstCopy
	case dedup.DecisionDuplicate:
		s.duplicates.Add(1)
		rec.Decision = DecisionDuplicate
		rec.FirstPath = firstPathTag(result.FirstPath)
	default:
		// Observe only returns invalid for an invalid id/path, which a decoded
		// probe cannot produce; treat defensively as rejected.
		s.rejected.Add(1)
		return Record{Decision: DecisionRejected}
	}

	// Per-packet detail at Debug, guarded so the attribute strings are not even
	// built when Debug is disabled (the common, high-throughput case).
	if s.logger.Enabled(context.Background(), slog.LevelDebug) {
		s.logger.Debug("probe",
			"decision", rec.Decision.String(),
			"path", rec.Path.String(),
			"first_path", rec.FirstPath.String(),
			"copy_count", rec.CopyCount,
		)
	}
	return rec
}

// logSummary emits the cumulative counters at Info. Serve calls it periodically
// so progress is visible without per-packet Info logging.
func (s *Server) logSummary(reason string) {
	st := s.Stats()
	s.logger.Info("probe summary",
		"reason", reason,
		"first_copies", st.FirstCopies,
		"duplicates", st.Duplicates,
		"rejected", st.Rejected,
	)
}

// Serve reads datagrams from conn until ctx is cancelled, handling each one. It
// deliberately discards the source address returned by ReadFrom so no source
// identifier reaches logging.
func (s *Server) Serve(ctx context.Context, conn net.PacketConn) error {
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	buf := make([]byte, maxDatagram)
	var processed uint64
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		s.Handle(buf[:n])
		processed++
		if processed%summaryEvery == 0 {
			s.logSummary("interval")
		}
	}
}

// Stats returns a snapshot of the cumulative counters.
func (s *Server) Stats() Stats {
	return Stats{
		FirstCopies: s.firstCopies.Load(),
		Duplicates:  s.duplicates.Load(),
		Rejected:    s.rejected.Load(),
	}
}

// firstPathTag maps the dedup window's recorded first path back to a PathTag.
func firstPathTag(path dedup.PathID) protocol.PathTag {
	for _, tag := range []protocol.PathTag{protocol.PathWiFi, protocol.PathAndroidUSBTether} {
		if dedup.PathID(tag.String()) == path {
			return tag
		}
	}
	return protocol.PathUnknown
}
