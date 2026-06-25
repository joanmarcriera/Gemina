//go:build linux

package exit

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"
)

// TUNSETIFF is the ioctl request that configures a /dev/net/tun file
// descriptor as a specific TUN/TAP interface. Value is architecture-independent
// on Linux (0x400454ca).
const tunsetiff = 0x400454ca

// siocsifmtu is the ioctl request to set a network interface's MTU.
const siocsifmtu = 0x8922

// ifReqSize is the fixed size of the ifreq struct used by both TUNSETIFF and
// SIOCSIFMTU ioctls. The kernel ABI defines this as 40 bytes on all
// architectures (16-byte name + 24-byte union).
const ifReqSize = 40

// TUN is a Linux TUN device opened via /dev/net/tun. It implements Device so
// it can be plugged directly into a Router. The file descriptor is owned by
// this struct; call Close when done.
type TUN struct {
	f *os.File
}

// OpenTUN opens (or creates) the TUN interface named name with the given MTU.
// It uses the IFF_TUN|IFF_NO_PI flags: raw IP packets, no extra packet-info
// header prepended by the kernel. The caller must have CAP_NET_ADMIN.
func OpenTUN(name string, mtu int) (*TUN, error) {
	f, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/net/tun: %w", err)
	}

	if err := tunSetIff(f, name); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("TUNSETIFF %s: %w", name, err)
	}

	if err := setMTU(name, mtu); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("set mtu on %s: %w", name, err)
	}

	return &TUN{f: f}, nil
}

// Read implements Device. It blocks until a packet arrives from the kernel and
// returns the raw IPv4 datagram in p.
func (t *TUN) Read(p []byte) (int, error) {
	return t.f.Read(p)
}

// Write implements Device. It injects the raw IPv4 datagram p into the kernel's
// IP stack via the TUN interface.
func (t *TUN) Write(p []byte) (int, error) {
	return t.f.Write(p)
}

// Close releases the TUN file descriptor. The interface is removed from the
// kernel when the last fd referencing it is closed.
func (t *TUN) Close() error {
	return t.f.Close()
}

// tunSetIff sends TUNSETIFF to configure the fd as interface name with
// IFF_TUN|IFF_NO_PI. The ifreq layout is: 16-byte null-padded interface name
// followed by a 2-byte flags field (the rest of the union is zeroed).
func tunSetIff(f *os.File, name string) error {
	// ifreq is a fixed 40-byte region; zero it then fill name and flags.
	var ifr [ifReqSize]byte
	copy(ifr[:16], name)
	// IFF_TUN = 0x0001, IFF_NO_PI = 0x1000; little-endian on all Linux arches
	// we target (amd64, arm64).
	const iffTun uint16 = 0x0001
	const iffNoPI uint16 = 0x1000
	flags := iffTun | iffNoPI
	ifr[16] = byte(flags)
	ifr[17] = byte(flags >> 8)

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		tunsetiff,
		uintptr(unsafe.Pointer(&ifr[0])),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// setMTU opens a temporary AF_INET socket and uses SIOCSIFMTU to set the MTU
// on the named interface. A socket is needed because SIOCSIFMTU operates on a
// socket fd, not the TUN fd.
func setMTU(name string, mtu int) error {
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return fmt.Errorf("socket: %w", err)
	}
	defer syscall.Close(sock) //nolint:errcheck

	// ifreq layout for SIOCSIFMTU: 16-byte name, then 4-byte MTU (int).
	var ifr [ifReqSize]byte
	copy(ifr[:16], name)
	// Store MTU as a native-endian int32 at offset 16.
	m := uint32(mtu)
	ifr[16] = byte(m)
	ifr[17] = byte(m >> 8)
	ifr[18] = byte(m >> 16)
	ifr[19] = byte(m >> 24)

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(sock),
		siocsifmtu,
		uintptr(unsafe.Pointer(&ifr[0])),
	)
	if errno != 0 {
		return errno
	}

	// Verify the interface is now visible to the net package (sanity check only;
	// the TUN fd is already valid if TUNSETIFF succeeded).
	if _, err := net.InterfaceByName(name); err != nil {
		return fmt.Errorf("interface %s not found after creation: %w", name, err)
	}
	return nil
}
