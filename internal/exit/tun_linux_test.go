//go:build linux

package exit

import (
	"net"
	"os"
	"testing"
)

// TestSetMTUSkipsWhenAlreadySet pins the least-privilege shortcut: when the
// interface already has the requested MTU (the host provisioned it), setMTU
// must not issue SIOCSIFMTU, which needs CAP_NET_ADMIN even for a no-op. Run
// unprivileged, a regression surfaces as EPERM.
func TestSetMTUSkipsWhenAlreadySet(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: the ioctl would succeed either way")
	}
	ifs, err := net.Interfaces()
	if err != nil {
		t.Fatalf("interfaces: %v", err)
	}
	for _, ifi := range ifs {
		if ifi.Flags&net.FlagLoopback != 0 {
			if err := setMTU(ifi.Name, ifi.MTU); err != nil {
				t.Fatalf("setMTU(%s, current %d) = %v, want nil (no ioctl)", ifi.Name, ifi.MTU, err)
			}
			return
		}
	}
	t.Skip("no loopback interface")
}
