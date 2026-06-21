//go:build !darwin

package transport

import (
	"errors"
	"runtime"
)

// ErrUnsupportedPlatform is returned when per-path egress binding is attempted
// on a platform without an implementation. The continuity client is macOS-only
// (see docs/adr/0002-swift-client-go-core.md); this stub exists so the package
// builds on the Linux gateway/CI hosts.
var ErrUnsupportedPlatform = errors.New("per-path egress binding is only implemented on darwin")

func bindToInterface(_ uintptr, _ string, _ int) error {
	return errors.New("path egress binding unsupported on " + runtime.GOOS + ": " + ErrUnsupportedPlatform.Error())
}
