//go:build linux

package main

import (
	"fmt"
	"os"
)

// assertNoCapabilities checks /proc/self/status after a privilege drop and fails
// if any capability is still permitted, effective or ambient (see
// checkNoCapabilities for the parsing rules).
func assertNoCapabilities() error {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return fmt.Errorf("read capabilities: %w", err)
	}
	defer f.Close() //nolint:errcheck
	return checkNoCapabilities(f)
}
