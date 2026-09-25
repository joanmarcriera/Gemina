package main

// Privilege drop for the gateway (issue #5).
//
// The Stage-2 data+exit gateway needs CAP_NET_ADMIN for exactly one moment: to
// attach the TUN device (TUNSETIFF) and, if the host has not already done so,
// set its MTU (SIOCSIFMTU). Docker does not put added capabilities into the
// effective set of a non-root container user, so `--cap-add NET_ADMIN` alone is
// useless for the distroless :nonroot uid (65532) — that is why the WS-F run
// crash-looped with "set mtu on gemina0: operation not permitted".
//
// The pattern: the container starts as uid 0 with only NET_ADMIN, SETUID and
// SETGID; the gateway opens the UDP socket and the TUN fd, then calls
// dropPrivileges, which switches every OS thread to the unprivileged uid/gid.
// Leaving uid 0 clears the permitted/effective/ambient capability sets (no
// SECBIT_KEEP_CAPS is set), so the process keeps only the already-open fds and
// runs the packet path — the part that touches untrusted input — unprivileged.

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// defaultRunAs is the distroless :nonroot uid:gid the image is built for.
const defaultRunAs = "65532:65532"

// runAsKeepRoot is the explicit opt-out value for GEMINA_GATEWAY_RUN_AS.
const runAsKeepRoot = "root"

// parseRunAs parses a "uid[:gid]" spec (numeric only — the distroless image has
// no user database). A missing gid defaults to the uid. It rejects uid/gid 0,
// because "dropping" to root is not a drop.
func parseRunAs(spec string) (uid, gid int, err error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return 0, 0, errors.New("empty run-as spec")
	}
	uidStr, gidStr, hasGid := strings.Cut(spec, ":")
	if uid, err = parseID(uidStr); err != nil {
		return 0, 0, fmt.Errorf("uid %q: %w", uidStr, err)
	}
	gid = uid
	if hasGid {
		if gid, err = parseID(gidStr); err != nil {
			return 0, 0, fmt.Errorf("gid %q: %w", gidStr, err)
		}
	}
	return uid, gid, nil
}

func parseID(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, errors.New("not a number")
	}
	if n <= 0 {
		return 0, errors.New("must be a positive, non-root id")
	}
	return n, nil
}

// dropPrivileges switches the process to uid/gid if it is running as root, and
// verifies the switch is irreversible. It reports whether a drop happened; a
// process that is already unprivileged is left alone (dropped=false, err=nil).
//
// Order matters: supplementary groups and gid must change while still root
// (they need CAP_SETGID), the uid last. Since Go 1.16 these syscalls apply to
// every OS thread on Linux, so no goroutine keeps running with root credentials.
func dropPrivileges(uid, gid int) (dropped bool, err error) {
	if os.Geteuid() != 0 {
		return false, nil
	}
	if err := syscall.Setgroups([]int{}); err != nil {
		return false, fmt.Errorf("setgroups: %w", err)
	}
	if err := syscall.Setgid(gid); err != nil {
		return false, fmt.Errorf("setgid %d: %w", gid, err)
	}
	if err := syscall.Setuid(uid); err != nil {
		return false, fmt.Errorf("setuid %d: %w", uid, err)
	}
	if os.Getuid() != uid || os.Geteuid() != uid || os.Getgid() != gid || os.Getegid() != gid {
		return false, fmt.Errorf("credentials after drop are uid=%d euid=%d gid=%d egid=%d, want %d:%d",
			os.Getuid(), os.Geteuid(), os.Getgid(), os.Getegid(), uid, gid)
	}
	// Regaining root must now be impossible; if it is not, the drop is unsafe.
	if err := syscall.Setuid(0); err == nil {
		return false, errors.New("setuid(0) succeeded after drop: privileges are recoverable")
	}
	if err := assertNoCapabilities(); err != nil {
		return false, err
	}
	return true, nil
}
