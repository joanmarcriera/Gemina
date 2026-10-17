package exit

import (
	"os/exec"
	"testing"
)

// TestExitHostEgressDenyRules runs scripts/test-exit-egress.sh, which asserts the
// exit host's DOCKER-USER DROP rules (cloud metadata + private ranges) sit above
// the forward ACCEPT and are idempotent, for both iptables and ip6tables.
func TestExitHostEgressDenyRules(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	out, err := exec.Command(bash, "../../scripts/test-exit-egress.sh").CombinedOutput()
	if err != nil {
		t.Fatalf("egress rule test failed: %v\n%s", err, out)
	}
}
