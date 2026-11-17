package gateway

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joanmarcriera/gemina/pkg/clientcore"
)

// replyFailConn is a net.PacketConn that yields queued datagrams, then blocks
// until closed, and fails every WriteTo so a lost reply can be observed.
type replyFailConn struct {
	in     chan []byte
	closed chan struct{}
	once   sync.Once
}

func newReplyFailConn(datagrams ...[]byte) *replyFailConn {
	c := &replyFailConn{in: make(chan []byte, len(datagrams)), closed: make(chan struct{})}
	for _, d := range datagrams {
		c.in <- d
	}
	return c
}

func (c *replyFailConn) ReadFrom(p []byte) (int, net.Addr, error) {
	select {
	case d := <-c.in:
		return copy(p, d), &net.UDPAddr{IP: net.IPv4(192, 0, 2, 1), Port: 4000}, nil
	case <-c.closed:
		return 0, nil, net.ErrClosed
	}
}

func (c *replyFailConn) WriteTo([]byte, net.Addr) (int, error) {
	return 0, errors.New("fake send failure")
}

func (c *replyFailConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return nil
}

func (c *replyFailConn) LocalAddr() net.Addr              { return &net.UDPAddr{} }
func (c *replyFailConn) SetDeadline(time.Time) error      { return nil }
func (c *replyFailConn) SetReadDeadline(time.Time) error  { return nil }
func (c *replyFailConn) SetWriteDeadline(time.Time) error { return nil }

// TestServeCountsReplySendFailures pins that a reply (here a ping echo; the same
// path carries the ServerHello) that the socket fails to send is counted rather
// than silently discarded, so an admitted-but-unanswered handshake is visible.
func TestServeCountsReplySendFailures(t *testing.T) {
	idPriv, _, _ := clientcore.GenerateIdentity()
	service, _ := hostedService(t)
	gw := NewDataGateway(idPriv, service, 64, nil)

	conn := newReplyFailConn(clientcore.EncodePing(1), clientcore.EncodePing(2))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- gw.Serve(ctx, conn) }()

	want := "gemina_reply_send_errors_total 2"
	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(gw.Metrics().Render(), want) {
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("metrics missing %q in:\n%s", want, gw.Metrics().Render())
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Serve returned %v, want nil on cancel", err)
	}
}
