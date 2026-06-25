//go:build linux

package exit

import (
	"errors"
	"os"
	"strings"
)

// ErrIPForwardDisabled is returned by AssertIPForward when the kernel is not
// configured to forward IPv4 packets. The operator must enable this before
// traffic can be routed through the gateway's exit interface.
//
// Enable with: echo 1 > /proc/sys/net/ipv4/ip_forward
// or persistently via sysctl net.ipv4.ip_forward=1 in /etc/sysctl.conf.
//
// Go does not configure the firewall or NAT rules (MASQUERADE) — that is the
// operator's responsibility and is intentionally left to kernel config so the
// gateway binary does not require elevated privileges beyond the TUN fd.
var ErrIPForwardDisabled = errors.New("ip forwarding is disabled: set net.ipv4.ip_forward=1")

// AssertIPForward reads /proc/sys/net/ipv4/ip_forward and returns
// ErrIPForwardDisabled if it is not set to "1". This is a health check only;
// the gateway does not write to the sysctl. Call this at startup so the
// operator gets a clear error rather than silent packet loss.
func AssertIPForward() error {
	data, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(data)) != "1" {
		return ErrIPForwardDisabled
	}
	return nil
}
