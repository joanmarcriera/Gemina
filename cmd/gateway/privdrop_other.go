//go:build !linux

package main

// assertNoCapabilities is a no-op off Linux: there are no Linux capability sets
// to inspect, and dropPrivileges' setuid(0) probe already proves the uid switch
// is irreversible. The gateway only deploys on Linux; this keeps dev builds
// compiling on macOS.
func assertNoCapabilities() error { return nil }
