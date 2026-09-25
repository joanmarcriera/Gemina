//go:build linux

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// assertNoCapabilities checks /proc/self/status after a privilege drop and fails
// if any capability is still permitted or effective. Leaving uid 0 should have
// cleared both; a non-zero set means something (e.g. SECBIT_KEEP_CAPS or file
// capabilities) kept privilege we explicitly meant to shed.
func assertNoCapabilities() error {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return fmt.Errorf("read capabilities: %w", err)
	}
	defer f.Close() //nolint:errcheck

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		key, val, ok := strings.Cut(sc.Text(), ":")
		if !ok || (key != "CapPrm" && key != "CapEff" && key != "CapAmb") {
			continue
		}
		bits, err := strconv.ParseUint(strings.TrimSpace(val), 16, 64)
		if err != nil {
			return fmt.Errorf("parse %s %q: %w", key, val, err)
		}
		if bits != 0 {
			return fmt.Errorf("%s=%016x after privilege drop, want 0", key, bits)
		}
	}
	return sc.Err()
}
