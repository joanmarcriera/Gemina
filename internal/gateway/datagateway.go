package gateway

import (
	"context"
	"crypto/ed25519"
	"errors"
	"log/slog"
	"net"

	"continuity-vpn/internal/entitlement"
	"continuity-vpn/internal/metrics"
	"continuity-vpn/pkg/clientcore"
)

// dataPath is the opaque path label the gateway uses for dedup bookkeeping. The
// duplicated copies of one logical packet are byte-identical, so the gateway
// cannot (and must not) attribute a datagram to Wi-Fi vs cellular from the packet
// alone — per-path delivery is a client-side signal. The gateway counts the
// decision (first-copy / duplicate / rejected) only.
const dataPath = "remote"

// DataRecord is what the DataGateway decided about one datagram, for the caller
// to act on (forward a delivered payload) and for logging.
type DataRecord struct {
	Kind     string // client-hello | data | unknown
	Admitted bool   // for a client-hello: was the session admitted
	Deliver  bool   // for data: is this the first copy to forward
	Payload  []byte // for data: the decrypted payload when Deliver is true
}

// DataGateway is the real gateway: it terminates the authenticated handshake,
// admits sessions by entitlement, and decrypts+deduplicates the data plane,
// exposing redacted Prometheus metrics. It never logs or stores a source address.
type DataGateway struct {
	identityPriv ed25519.PrivateKey
	admitter     *Admitter
	store        *SessionStore
	dataPlane    *DataPlane
	capacity     int
	logger       *slog.Logger

	metrics       *metrics.Registry
	handshakes    *metrics.CounterVec // continuity_handshakes_total{result}
	dataPackets   *metrics.CounterVec // continuity_data_packets_total{decision}
	activeSession *metrics.GaugeVec   // continuity_active_sessions
}

// NewDataGateway builds a gateway with the given Ed25519 identity, entitlement
// service (ModeOpen for self-host, ModeHosted for the paid tier) and per-session
// dedup capacity. A nil logger discards logs.
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
	}
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
	serverHello, _, err := g.admitter.Handshake(datagram, g.identityPriv, g.capacity)
	if err != nil {
		g.handshakes.Inc("rejected")
		if g.logger.Enabled(context.Background(), slog.LevelDebug) {
			g.logger.Debug("handshake rejected", "reason", err.Error())
		}
		return nil, DataRecord{Kind: "client-hello", Admitted: false}
	}
	g.handshakes.Inc("admitted")
	g.activeSession.Set(int64(g.store.len()))
	return serverHello, DataRecord{Kind: "client-hello", Admitted: true}
}

func (g *DataGateway) handleData(datagram []byte) DataRecord {
	payload, first, err := g.dataPlane.Handle(datagram, dataPath)
	if err != nil {
		g.dataPackets.Inc("rejected")
		return DataRecord{Kind: "data"}
	}
	if first {
		g.dataPackets.Inc("first-copy")
		return DataRecord{Kind: "data", Deliver: true, Payload: payload}
	}
	g.dataPackets.Inc("duplicate")
	return DataRecord{Kind: "data"}
}

// Serve reads datagrams until ctx is cancelled, handling each and sending any
// handshake reply back to its source. The source address is used only to reply
// and is never logged.
func (g *DataGateway) Serve(ctx context.Context, conn net.PacketConn) error {
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

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
		_ = rec // forwarding a delivered payload to the internet is the exit path (future)
	}
}
