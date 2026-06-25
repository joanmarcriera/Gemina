package gateway

import (
	"context"
	"crypto/ed25519"
	"errors"
	"log/slog"
	"net"
	"net/netip"

	"continuity-vpn/internal/dedup"
	"continuity-vpn/internal/entitlement"
	"continuity-vpn/internal/exit"
	"continuity-vpn/internal/metrics"
	"continuity-vpn/internal/protocol"
	"continuity-vpn/pkg/clientcore"
)

// dataPath is the opaque path label the gateway uses for dedup bookkeeping. The
// duplicated copies of one logical packet are byte-identical, so the gateway
// cannot (and must not) attribute a datagram to Wi-Fi vs cellular from the packet
// alone — per-path delivery is a client-side signal. The gateway counts the
// decision (first-copy / duplicate / stale / rejected) only.
const dataPath = "remote"

// DataRecord is what the DataGateway decided about one datagram, for the caller
// to act on (forward a delivered payload) and for logging.
type DataRecord struct {
	Kind      string             // client-hello | data | ping | unknown
	Admitted  bool               // for a client-hello: was the session admitted
	Deliver   bool               // for data: is this the first copy to forward
	Payload   []byte             // for data: the decrypted payload when Deliver is true
	SessionID protocol.SessionID // the session this datagram belongs to (zero if unknown)
	TunnelIP  netip.Addr         // for an admitted client-hello with exit on: the leased tunnel IP
}

// DataGateway is the real gateway: it terminates the authenticated handshake,
// admits sessions by entitlement, decrypts+deduplicates the data plane, and —
// when an exit Router is enabled — forwards delivered packets to the internet and
// routes return traffic back. It exposes redacted Prometheus metrics and never
// logs or stores a source address.
type DataGateway struct {
	identityPriv ed25519.PrivateKey
	admitter     *Admitter
	store        *SessionStore
	dataPlane    *DataPlane
	capacity     int
	logger       *slog.Logger

	// exit is the optional internet exit path. When nil the gateway decrypts and
	// dedups but drops the payload (the Stage-1 probe behaviour).
	exit *exit.Router

	metrics       *metrics.Registry
	handshakes    *metrics.CounterVec // continuity_handshakes_total{result}
	dataPackets   *metrics.CounterVec // continuity_data_packets_total{decision}
	activeSession *metrics.GaugeVec   // continuity_active_sessions

	exitForwarded      *metrics.CounterVec // continuity_exit_forwarded_total
	exitForwardedBytes *metrics.CounterVec // continuity_exit_forwarded_bytes_total
	exitReturned       *metrics.CounterVec // continuity_exit_returned_total
	exitReturnedBytes  *metrics.CounterVec // continuity_exit_returned_bytes_total
	exitDropped        *metrics.CounterVec // continuity_exit_dropped_total{reason}
}

// NewDataGateway builds a gateway with the given Ed25519 identity, entitlement
// service (ModeOpen for self-host, ModeHosted for the paid tier) and per-session
// dedup capacity. A nil logger discards logs. The exit path is off until
// EnableExit is called.
func NewDataGateway(identityPriv ed25519.PrivateKey, service *entitlement.Service, dedupCapacity int, logger *slog.Logger) *DataGateway {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	store := NewSessionStore()
	reg := metrics.NewRegistry()
	return &DataGateway{
		identityPriv: identityPriv,
		admitter:     NewAdmitter(service, store),
		store:        store,
		dataPlane:    NewDataPlane(store, dedupCapacity),
		capacity:     dedupCapacity,
		logger:       logger,
		metrics:      reg,
		handshakes:   reg.Counter("continuity_handshakes_total", "Handshakes by result.", "result"),
		dataPackets:  reg.Counter("continuity_data_packets_total", "Data datagrams by decision.", "decision"),
		activeSession: reg.Gauge("continuity_active_sessions",
			"Sessions currently admitted on the gateway."),
		exitForwarded:      reg.Counter("continuity_exit_forwarded_total", "Inner packets forwarded to the internet."),
		exitForwardedBytes: reg.Counter("continuity_exit_forwarded_bytes_total", "Inner bytes forwarded to the internet."),
		exitReturned:       reg.Counter("continuity_exit_returned_total", "Return datagrams sent back to clients."),
		exitReturnedBytes:  reg.Counter("continuity_exit_returned_bytes_total", "Return bytes sent back to clients."),
		exitDropped:        reg.Counter("continuity_exit_dropped_total", "Exit-path drops by reason.", "reason"),
	}
}

// EnableExit turns on the internet exit path, routing delivered packets through r
// and serving return traffic when Serve runs. Call before Serve.
func (g *DataGateway) EnableExit(r *exit.Router) { g.exit = r }

// FrameReturn implements exit.Framer: it re-encapsulates a return inner-IP
// payload into a CVD1 datagram for the session so the client can decrypt it.
func (g *DataGateway) FrameReturn(id protocol.SessionID, payload []byte) ([]byte, error) {
	return g.dataPlane.FrameReturn(id, payload)
}

// Forwarded, Dropped and Returned implement exit.Metrics, feeding the router's
// decisions into the gateway's redacted Prometheus counters.
func (g *DataGateway) Forwarded(bytes int) {
	g.exitForwarded.Inc()
	g.exitForwardedBytes.Add(int64(bytes))
}

func (g *DataGateway) Dropped(reason string) { g.exitDropped.Inc(reason) }

func (g *DataGateway) Returned(bytes int) {
	g.exitReturned.Inc()
	g.exitReturnedBytes.Add(int64(bytes))
}

// Metrics returns the registry rendering the gateway's redacted metrics.
func (g *DataGateway) Metrics() *metrics.Registry { return g.metrics }

// HandleDatagram processes one received datagram. For a ClientHello it returns
// the ServerHello to send back (or nil if the client was not admitted). For a
// data datagram it returns a nil reply and a record whose Payload should be
// forwarded when Deliver is true. It never returns or logs the source address.
func (g *DataGateway) HandleDatagram(datagram []byte) (reply []byte, rec DataRecord) {
	switch clientcore.ClassifyDatagram(datagram) {
	case clientcore.KindClientHello:
		return g.handleHandshake(datagram)
	case clientcore.KindData:
		return nil, g.handleData(datagram)
	case clientcore.KindPing:
		// Echo a pong for latency/loss measurement (continuityctl benchmark).
		// The pong is the same size as the ping, so there is no amplification.
		if isPong, nonce, err := clientcore.DecodePing(datagram); err == nil && !isPong {
			return clientcore.EncodePong(nonce), DataRecord{Kind: "ping"}
		}
		return nil, DataRecord{Kind: "ping"}
	default:
		g.dataPackets.Inc("rejected")
		return nil, DataRecord{Kind: "unknown"}
	}
}

func (g *DataGateway) handleHandshake(datagram []byte) ([]byte, DataRecord) {
	serverHello, _, id, err := g.admitter.Handshake(datagram, g.identityPriv, g.capacity)
	if err != nil {
		g.handshakes.Inc("rejected")
		if g.logger.Enabled(context.Background(), slog.LevelDebug) {
			g.logger.Debug("handshake rejected", "reason", err.Error())
		}
		return nil, DataRecord{Kind: "client-hello", Admitted: false}
	}
	g.handshakes.Inc("admitted")
	g.activeSession.Set(int64(g.store.len()))

	rec := DataRecord{Kind: "client-hello", Admitted: true, SessionID: id}
	if g.exit != nil {
		// Reserve the tunnel IP now so the reverse-path filter and the return-path
		// lookup work from the first packet. In-band delivery of the address to the
		// client is a separate wire step (see the TASKS list).
		if addr, leaseErr := g.exit.Lease(id); leaseErr == nil {
			rec.TunnelIP = addr
		} else if g.logger.Enabled(context.Background(), slog.LevelDebug) {
			g.logger.Debug("tunnel lease failed", "reason", leaseErr.Error())
		}
	}
	return serverHello, rec
}

func (g *DataGateway) handleData(datagram []byte) DataRecord {
	// Decode the session id first so callers (the Serve loop) can attribute the
	// source endpoint even for a duplicate; a malformed datagram leaves it zero.
	id, _ := clientcore.SessionIDFromDatagram(datagram)

	payload, decision, err := g.dataPlane.HandleClassified(datagram, dataPath)
	if err != nil {
		g.dataPackets.Inc("rejected")
		return DataRecord{Kind: "data", SessionID: id}
	}
	switch decision {
	case dedup.ReplayFirstCopy:
		g.dataPackets.Inc("first-copy")
		return DataRecord{Kind: "data", Deliver: true, Payload: payload, SessionID: id}
	case dedup.ReplayStale:
		g.dataPackets.Inc("stale")
		return DataRecord{Kind: "data", SessionID: id}
	default: // ReplayDuplicate
		g.dataPackets.Inc("duplicate")
		return DataRecord{Kind: "data", SessionID: id}
	}
}

// Serve reads datagrams until ctx is cancelled, handling each and sending any
// handshake reply back to its source. When the exit path is enabled it also runs
// the return loop and forwards each delivered packet to the internet, recording
// the (transient, never-logged) source endpoint so return traffic can find the
// client. The source address is used only to reply/route and is never logged.
func (g *DataGateway) Serve(ctx context.Context, conn net.PacketConn) error {
	go func() {
		<-ctx.Done()
		_ = conn.Close()
		// Close the exit device too, so the ServeReturn goroutine parked in a
		// blocking TUN read wakes and exits instead of leaking past shutdown.
		if g.exit != nil {
			_ = g.exit.Close()
		}
	}()

	if g.exit != nil {
		go func() {
			if err := g.exit.ServeReturn(ctx); err != nil {
				g.logger.Error("exit return loop", "error", err.Error())
			}
		}()
	}

	buf := make([]byte, 2048)
	for {
		n, src, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		reply, rec := g.HandleDatagram(buf[:n])
		if reply != nil {
			_, _ = conn.WriteTo(reply, src)
		}

		if g.exit != nil && rec.Kind == "data" && !rec.SessionID.IsZero() {
			// Record every arriving path (first copy or duplicate) so the return
			// path duplicates across all of a client's active uplinks.
			if ap, ok := udpAddrPort(src); ok {
				g.exit.RecordPath(rec.SessionID, ap)
			}
			if rec.Deliver {
				if err := g.exit.Egress(rec.SessionID, rec.Payload); err != nil &&
					g.logger.Enabled(ctx, slog.LevelDebug) {
					g.logger.Debug("egress dropped", "reason", err.Error())
				}
			}
		}
	}
}

// udpAddrPort extracts a netip.AddrPort from a UDP source address, returning
// false for any non-UDP address (the return path only handles UDP clients).
func udpAddrPort(a net.Addr) (netip.AddrPort, bool) {
	if ua, ok := a.(*net.UDPAddr); ok {
		return ua.AddrPort(), true
	}
	return netip.AddrPort{}, false
}
