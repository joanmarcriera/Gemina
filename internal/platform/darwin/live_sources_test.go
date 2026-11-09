package darwin

import (
	"errors"
	"net"
	"os/exec"
	"strings"
	"testing"

	"github.com/joanmarcriera/gemina/internal/paths"
)

// TestOSCommandRunner pins the real runner's contract: stdout is returned, a
// non-zero exit surfaces as *exec.ExitError, and a missing binary as
// exec.ErrNotFound. Uses only POSIX utilities present on macOS and Linux.
func TestOSCommandRunner(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH")
	}
	runner := OSCommandRunner{}

	out, err := runner.RunCommand("sh", "-c", "printf 'hello'")
	if err != nil || string(out) != "hello" {
		t.Fatalf("RunCommand(printf) = %q, %v", out, err)
	}

	_, err = runner.RunCommand("sh", "-c", "exit 3")
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 3 {
		t.Fatalf("RunCommand(exit 3) err = %v, want ExitError code 3", err)
	}

	if _, err := runner.RunCommand("gemina-definitely-not-a-binary"); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("missing binary err = %v, want exec.ErrNotFound", err)
	}
}

// TestNetInterfaceSourceMatchesHost pins that the live source reports exactly
// the host's interfaces, carries flags through, and never guesses a link kind
// (classification is evidence-driven, not name-driven). Addresses are compared
// only by count so no host address is logged.
func TestNetInterfaceSourceMatchesHost(t *testing.T) {
	host, err := net.Interfaces()
	if err != nil {
		t.Fatalf("net.Interfaces: %v", err)
	}
	records, err := NetInterfaceSource{}.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces: %v", err)
	}
	if len(records) != len(host) {
		t.Fatalf("got %d records, host has %d interfaces", len(records), len(host))
	}
	for i, rec := range records {
		if rec.BSDName != host[i].Name || rec.Flags != host[i].Flags {
			t.Errorf("record %d = %s/%v, want %s/%v", i, rec.BSDName, rec.Flags, host[i].Name, host[i].Flags)
		}
		if rec.Kind != paths.LinkKindUnknown {
			t.Errorf("record %s Kind = %v, want unknown (no name-based guessing)", rec.BSDName, rec.Kind)
		}
		addrs, _ := host[i].Addrs()
		if len(rec.AddressCIDRs) != len(addrs) {
			t.Errorf("record %s has %d addresses, host has %d", rec.BSDName, len(rec.AddressCIDRs), len(addrs))
		}
		for _, cidr := range rec.AddressCIDRs {
			if !strings.Contains(cidr, "/") {
				t.Errorf("record %s address not in CIDR form", rec.BSDName)
			}
		}
	}

	snaps, err := LiveInterfaceSnapshots()
	if err != nil {
		t.Fatalf("LiveInterfaceSnapshots: %v", err)
	}
	if len(snaps) != len(host) {
		t.Fatalf("LiveInterfaceSnapshots gave %d, host has %d", len(snaps), len(host))
	}
	for i, s := range snaps {
		if s.Loopback != (host[i].Flags&net.FlagLoopback != 0) || s.Up != (host[i].Flags&net.FlagUp != 0) {
			t.Errorf("snapshot %s flags mismatch: %+v vs %v", s.BSDName, s, host[i].Flags)
		}
	}
}
