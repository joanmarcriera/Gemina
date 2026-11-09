package exit

import (
	"errors"
	"fmt"
	"strings"
)

// maxTUNNameLen is IFNAMSIZ-1: Linux interface names are at most 15 bytes plus
// the terminating NUL in the 16-byte ifreq name field.
const maxTUNNameLen = 15

// ErrInvalidTUNName reports a TUN interface name the kernel would reject,
// truncate or reinterpret.
var ErrInvalidTUNName = errors.New("invalid tun interface name")

// ValidateTUNName checks name against the Linux dev_valid_name rules before it
// is copied into a fixed 16-byte ifreq. Without this, an over-long name is
// silently truncated (TUNSETIFF then names one interface while SIOCSIFMTU and
// the logs refer to another), and a '%' turns the name into a kernel template
// that allocates a different interface than the one later configured.
func ValidateTUNName(name string) error {
	switch {
	case name == "":
		return fmt.Errorf("%w: empty", ErrInvalidTUNName)
	case len(name) > maxTUNNameLen:
		return fmt.Errorf("%w: %d bytes, max %d", ErrInvalidTUNName, len(name), maxTUNNameLen)
	case name == "." || name == "..":
		return fmt.Errorf("%w: %q is reserved", ErrInvalidTUNName, name)
	}
	if i := strings.IndexFunc(name, func(r rune) bool {
		return r <= ' ' || r == '/' || r == ':' || r == '%' || r >= 0x7f
	}); i >= 0 {
		return fmt.Errorf("%w: disallowed byte at offset %d", ErrInvalidTUNName, i)
	}
	return nil
}
