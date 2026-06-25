package exit

import (
	"context"
	"errors"
	"io"
	"net/netip"

	"continuity-vpn/internal/protocol"
)

// ErrReversePath is returned by Egress when the inner packet's source address
// does not match the tunnel IP leased to the sending session. This is the
// reverse-path filter: a client may only originate traffic from its own
// assigned address.
var ErrReversePath = errors.New("reverse-path filter: inner src does not match session lease")

// ErrNoLease is returned by Egress for a session that has no tunnel-IP lease.
// In normal operation the gateway leases the address at admission, so this only
// happens if a packet is delivered for a session that was never admitted (or was
// released), which must not be forwarded.
var ErrNoLease = errors.New("no tunnel ip lease for session")

// Device is the TUN interface abstraction. Read returns raw inner IP packets
// arriving from the internet; Write injects inner IP packets from clients into
// the kernel's IP stack.
type Device interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
}

// Framer re-encapsulates a return inner-IP payload into a CVD1 datagram
// addressed to the given session so the client can decrypt it.
type Framer interface {
	FrameReturn(id protocol.SessionID, payload []byte) ([]byte, error)
}

// Sink delivers an already-framed datagram to a specific remote endpoint. The
// gateway calls this once per fresh source path, duplicating the return packet
// across every known path for the session.
type Sink interface {
	SendTo(datagram []byte, dst netip.AddrPort) error
}

// Metrics is an optional seam for recording forwarding decisions. The no-op
// default (noopMetrics) is used when nil is passed to NewRouter so callers do
// not need to provide one.
type Metrics interface {
	// Forwarded records that a packet of the given byte length was written to
	// the TUN device (egress direction).
	Forwarded(bytes int)
	// Dropped records that a packet was silently discarded for reason.
	Dropped(reason string)
	// Returned records that a return packet of the given byte length was sent
	// back toward a client endpoint.
	Returned(bytes int)
}

type noopMetrics struct{}

func (noopMetrics) Forwarded(int)  {}
func (noopMetrics) Dropped(string) {}
func (noopMetrics) Returned(int)   {}

// Router ties together the Allocator, PathSet, TUN Device, Framer, and Sink
// into the complete exit-path engine. It is safe for concurrent use.
type Router struct {
	alloc   *Allocator
	paths   *PathSet
	dev     Device
	framer  Framer
	sink    Sink
	metrics Metrics
}

// NewRouter builds a Router. If m is nil, a no-op Metrics implementation is
// used so the router always has a valid metrics sink.
func NewRouter(alloc *Allocator, paths *PathSet, dev Device, framer Framer, sink Sink, m Metrics) *Router {
	if m == nil {
		m = noopMetrics{}
	}
	return &Router{
		alloc:   alloc,
		paths:   paths,
		dev:     dev,
		framer:  framer,
		sink:    sink,
		metrics: m,
	}
}

// Lease reserves (idempotently) the tunnel IP for id and returns it. The gateway
// calls this at admission so the lease and its reverse mapping exist before any
// data flows, and so the assigned address can be reported to the client.
func (r *Router) Lease(id protocol.SessionID) (netip.Addr, error) {
	return r.alloc.Allocate(id)
}

// RecordPath notes addr as a source endpoint for id. Called by the receive
// loop each time a datagram arrives so the return path stays current.
func (r *Router) RecordPath(id protocol.SessionID, addr netip.AddrPort) {
	r.paths.Record(id, addr)
}

// Egress applies the reverse-path filter and, if it passes, writes innerPacket
// to the TUN device. The inner packet's source address must equal the tunnel IP
// leased to id; any mismatch is dropped and ErrReversePath is returned. This
// prevents a compromised or misconfigured client from spoofing another client's
// address inside the tunnel.
func (r *Router) Egress(id protocol.SessionID, innerPacket []byte) error {
	leased, ok := r.alloc.LeaseOf(id)
	if !ok {
		r.metrics.Dropped("no-lease")
		return ErrNoLease
	}

	src, _, ok := parseIPv4(innerPacket)
	if !ok {
		r.metrics.Dropped("bad-inner-packet")
		return errors.New("inner packet is not a valid ipv4 datagram")
	}

	if src != leased {
		r.metrics.Dropped("reverse-path")
		return ErrReversePath
	}

	if _, err := r.dev.Write(innerPacket); err != nil {
		r.metrics.Dropped("tun-write-error")
		return err
	}
	r.metrics.Forwarded(len(innerPacket))
	return nil
}

// Close releases the underlying device if it supports it. Closing the device
// unblocks a ServeReturn goroutine parked in dev.Read, so the caller closes it
// on shutdown to let the return loop exit instead of leaking.
func (r *Router) Close() error {
	if c, ok := r.dev.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// ServeReturn reads inner IP packets from the TUN device and routes each one
// back to every fresh source endpoint known for the destination session. It
// runs until ctx is cancelled or the device returns a read error. A cancelled
// context is not treated as an error — ctx.Err() is checked before surfacing
// device read errors.
func (r *Router) ServeReturn(ctx context.Context) error {
	// 64 KiB is the maximum IPv4 packet size; allocate once and reuse.
	buf := make([]byte, 65535)
	for {
		n, err := r.dev.Read(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		packet := buf[:n]
		_, dst, ok := parseIPv4(packet)
		if !ok {
			r.metrics.Dropped("bad-return-packet")
			continue
		}

		id, ok := r.alloc.Lookup(dst)
		if !ok {
			r.metrics.Dropped("unknown-dst")
			continue
		}

		framed, err := r.framer.FrameReturn(id, packet)
		if err != nil {
			r.metrics.Dropped("frame-error")
			continue
		}

		dsts := r.paths.Fresh(id)
		for _, ep := range dsts {
			// Best-effort delivery to each fresh endpoint; one path failing
			// must not block the others.
			if err := r.sink.SendTo(framed, ep); err == nil {
				r.metrics.Returned(len(framed))
			}
		}
	}
}
