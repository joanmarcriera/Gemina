package darwin

import (
	"fmt"
	"strings"
)

// MacOSProductVersion returns the macOS product version (e.g. "14.5") via
// `sw_vers -productVersion`. The runner is injectable so the boundary is
// testable from a fixture. The OS version is not a host identifier, so it is
// safe to surface; it feeds the compatibility verdict.
func MacOSProductVersion(runner CommandRunner) (string, error) {
	if runner == nil {
		runner = OSCommandRunner{}
	}
	out, err := runner.RunCommand("sw_vers", "-productVersion")
	if err != nil {
		return "", fmt.Errorf("sw_vers product version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// LiveMacOSProductVersion reads the macOS version from the live system.
func LiveMacOSProductVersion() (string, error) {
	return MacOSProductVersion(OSCommandRunner{})
}
