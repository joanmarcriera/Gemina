//go:build linux

package exit

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
)

// newSocketpairTUN builds a TUN around one end of a SOCK_SEQPACKET socketpair,
// which, like a real TUN fd, preserves packet boundaries. The other end plays
// the kernel. No CAP_NET_ADMIN needed.
func newSocketpairTUN(t *testing.T) (*TUN, *os.File) {
	t.Helper()
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_SEQPACKET, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}
	tun := &TUN{f: os.NewFile(uintptr(fds[0]), "tun-test")}
	peer := os.NewFile(uintptr(fds[1]), "kernel-test")
	t.Cleanup(func() { _ = peer.Close() })
	return tun, peer
}

// TestTUNReadWriteClose pins that TUN's Device methods are thin, boundary-
// preserving passthroughs to the fd and that Close releases it (later I/O
// fails with os.ErrClosed rather than touching a reused descriptor).
func TestTUNReadWriteClose(t *testing.T) {
	tun, peer := newSocketpairTUN(t)
	var _ Device = tun

	pkts := [][]byte{{0x45, 1, 2, 3}, bytes.Repeat([]byte{0xAB}, 1500)}
	for _, p := range pkts {
		if n, err := tun.Write(p); err != nil || n != len(p) {
			t.Fatalf("Write = %d, %v; want %d, nil", n, err, len(p))
		}
		buf := make([]byte, 65535)
		n, err := peer.Read(buf)
		if err != nil || !bytes.Equal(buf[:n], p) {
			t.Fatalf("peer read %d bytes, %v; want the %d-byte packet", n, err, len(p))
		}
	}

	for _, p := range pkts {
		if _, err := peer.Write(p); err != nil {
			t.Fatalf("peer write: %v", err)
		}
	}
	for _, p := range pkts {
		buf := make([]byte, 65535)
		n, err := tun.Read(buf)
		if err != nil || !bytes.Equal(buf[:n], p) {
			t.Fatalf("Read = %d, %v; want the %d-byte packet intact", n, err, len(p))
		}
	}

	if err := tun.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := tun.Read(make([]byte, 8)); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("Read after Close = %v, want os.ErrClosed", err)
	}
	if _, err := tun.Write([]byte{1}); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("Write after Close = %v, want os.ErrClosed", err)
	}
	if err := tun.Close(); err == nil {
		t.Fatal("second Close succeeded, want error")
	}
}

// TestAssertIPForwardMatchesSysctl pins that AssertIPForward's verdict agrees
// with the host sysctl it reads: nil iff the value is "1", ErrIPForwardDisabled
// for any other value, and the read error when the file is unreadable.
func TestAssertIPForwardMatchesSysctl(t *testing.T) {
	err := AssertIPForward()
	data, readErr := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	switch {
	case readErr != nil:
		if err == nil || errors.Is(err, ErrIPForwardDisabled) {
			t.Fatalf("unreadable sysctl (%v) but AssertIPForward = %v", readErr, err)
		}
	case strings.TrimSpace(string(data)) == "1":
		if err != nil {
			t.Fatalf("ip_forward=1 but AssertIPForward = %v", err)
		}
	default:
		if !errors.Is(err, ErrIPForwardDisabled) {
			t.Fatalf("ip_forward=%q but AssertIPForward = %v, want ErrIPForwardDisabled", data, err)
		}
	}
}
