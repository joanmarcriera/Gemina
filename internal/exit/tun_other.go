//go:build !linux

package exit

import "errors"

// TUN is a placeholder type on non-Linux platforms. The gateway's exit path
// requires a Linux TUN device; this stub exists only so that darwin and other
// builds compile without the linux-specific file.
type TUN struct{}

// OpenTUN always returns an error on non-Linux platforms. The gateway is
// deployed on linux/arm64; this stub allows the macOS development build to
// compile and the darwin client to import internal/exit for its interface
// types without pulling in linux syscalls.
func OpenTUN(name string, mtu int) (*TUN, error) {
	return nil, errors.New("tun device is only supported on linux")
}

// Read is a placeholder to satisfy the Device interface. It always errors.
func (t *TUN) Read(p []byte) (int, error) {
	return 0, errors.New("tun device is only supported on linux")
}

// Write is a placeholder to satisfy the Device interface. It always errors.
func (t *TUN) Write(p []byte) (int, error) {
	return 0, errors.New("tun device is only supported on linux")
}

// Close is a no-op on non-Linux platforms.
func (t *TUN) Close() error {
	return nil
}
