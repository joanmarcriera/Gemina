//go:build darwin

package transport

import "syscall"

// Darwin socket options that pin a socket's egress to a specific interface,
// regardless of the routing table. Values from <netinet/in.h> /
// <netinet6/in6.h>; not exposed by the syscall package.
const (
	ipBoundIf   = 0x19 // IP_BOUND_IF
	ipv6BoundIf = 0x7d // IPV6_BOUND_IF
)

// bindToInterface forces the socket identified by fd to egress through the
// interface with the given index. It sets the family-appropriate BOUND_IF
// option based on the dialed network; for an unspecified family it sets both as
// a best effort.
func bindToInterface(fd uintptr, network string, ifIndex int) error {
	switch network {
	case "udp4", "tcp4", "ip4":
		return syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, ipBoundIf, ifIndex)
	case "udp6", "tcp6", "ip6":
		return syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, ipv6BoundIf, ifIndex)
	default:
		if err := syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, ipBoundIf, ifIndex); err != nil {
			return err
		}
		return syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, ipv6BoundIf, ifIndex)
	}
}
