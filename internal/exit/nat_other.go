//go:build !linux

package exit

import "errors"

// ErrIPForwardDisabled is the sentinel returned by AssertIPForward when the
// kernel is not forwarding packets. On non-Linux platforms this is always
// returned because /proc is not available.
var ErrIPForwardDisabled = errors.New("ip forwarding is disabled: set net.ipv4.ip_forward=1")

// AssertIPForward is a stub on non-Linux platforms. It always returns an error
// because the gateway exit path (and the /proc sysctl check) is only supported
// on Linux.
func AssertIPForward() error {
	return errors.New("ip forward check is only supported on linux")
}
