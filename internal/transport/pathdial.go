package transport

import (
	"errors"
	"fmt"
	"net"
	"syscall"
)

// ErrNoInterface is returned when a PathDialer has no interface name set.
var ErrNoInterface = errors.New("path dialer requires an interface name")

// PathDialer dials a connected UDP socket whose egress is bound to a specific
// network interface, so a datagram leaves a chosen path regardless of the
// default route. This is the client-side primitive for proving per-path egress:
// one PathDialer per uplink (e.g. Wi-Fi, Android USB tether).
//
// Binding is enforced at the socket layer (Darwin IP_BOUND_IF), not by source
// address selection, so it holds even when another interface owns the default
// route.
type PathDialer struct {
	// Interface is the BSD interface name to bind egress to, e.g. "en0".
	Interface string
}

// DialUDP resolves the interface, then dials a connected UDP socket bound to it.
// remote is a host:port string. The returned conn writes only over the bound
// interface.
func (d PathDialer) DialUDP(remote string) (*net.UDPConn, error) {
	if d.Interface == "" {
		return nil, ErrNoInterface
	}

	iface, err := net.InterfaceByName(d.Interface)
	if err != nil {
		return nil, fmt.Errorf("resolve interface %q: %w", d.Interface, err)
	}

	dialer := net.Dialer{
		Control: func(network, _ string, c syscall.RawConn) error {
			var bindErr error
			if err := c.Control(func(fd uintptr) {
				bindErr = bindToInterface(fd, network, iface.Index)
			}); err != nil {
				return err
			}
			return bindErr
		},
	}

	conn, err := dialer.Dial("udp", remote)
	if err != nil {
		return nil, fmt.Errorf("dial %s via %s: %w", remote, d.Interface, err)
	}

	udp, ok := conn.(*net.UDPConn)
	if !ok {
		_ = conn.Close()
		return nil, fmt.Errorf("dial %s via %s: not a UDP connection", remote, d.Interface)
	}
	return udp, nil
}
