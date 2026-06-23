package darwin

import (
	"fmt"
	"strconv"
	"strings"
)

// USB interface descriptor values that identify an Android RNDIS tethering
// function. Android exposes RNDIS tethering as a "Wireless Controller" control
// interface (class 0xE0) with the Microsoft RNDIS subclass/protocol, paired with
// a CDC-data interface. Ordinary USB network adapters (CDC-ECM/NCM) use the
// Communications class (0x02) instead, so matching on this class — not a vendor
// string — distinguishes a phone tether from a dock's built-in NIC.
const (
	usbClassWirelessController = 224 // 0xE0
	usbSubclassRNDIS           = 1   // 0x01
	usbProtocolRNDIS           = 3   // 0x03
)

// USBTetherFunction is a device-level signal that a USB function capable of
// carrying a tethered uplink is present on the bus. It is deliberately NOT a
// usable network path: with no host driver claiming it there is no enX NIC, no
// address and no link. It exists so the diagnostic can honestly explain why an
// expected uplink candidate is absent, rather than silently omitting it. It must
// never be promoted into a usable path Candidate.
type USBTetherFunction struct {
	// Transport is a coarse, redacted token from the shared evidence vocabulary
	// (e.g. EvidenceValueAndroidRNDIS). It never carries a serial, product or
	// vendor string.
	Transport string
	// HostDriverClaimed reports whether macOS has bound a driver and produced a
	// network interface for this function. It is false whenever the function is
	// only visible at the USB layer — which, for Android RNDIS on macOS, is
	// always, because no RNDIS host driver ships.
	HostDriverClaimed bool
}

// USBFunctionDeviceSource detects tether-capable USB functions at the USB layer,
// before (and independently of) any BSD network interface appearing. This is the
// device-level companion to the interface-level evidence sources: an unclaimed
// RNDIS function never publishes an IOEthernetInterface, so it can only be seen
// here.
type USBFunctionDeviceSource struct {
	Runner CommandRunner
}

// TetherFunctions queries the USB layer and returns the tether-capable functions
// present on the bus.
func (source USBFunctionDeviceSource) TetherFunctions() ([]USBTetherFunction, error) {
	runner := source.Runner
	if runner == nil {
		runner = OSCommandRunner{}
	}

	output, err := runner.RunCommand("ioreg", "-r", "-c", "IOUSBHostInterface", "-l")
	if err != nil {
		return nil, fmt.Errorf("ioreg usb host interfaces: %w", err)
	}
	return parseUSBHostInterfaceFunctions(output), nil
}

// parseUSBHostInterfaceFunctions scans `ioreg -r -c IOUSBHostInterface -l` output
// for the Android RNDIS control-interface signature and reports one tether
// function per match. It reads only the numeric interface-descriptor fields, so
// no product/vendor/serial string can reach the result.
func parseUSBHostInterfaceFunctions(output []byte) []USBTetherFunction {
	var functions []USBTetherFunction
	for _, block := range splitIORegistryBlocks(string(output)) {
		if blockIsAndroidRNDISControl(block) {
			functions = append(functions, USBTetherFunction{
				Transport:         EvidenceValueAndroidRNDIS,
				HostDriverClaimed: false,
			})
		}
	}
	return functions
}

func blockIsAndroidRNDISControl(block string) bool {
	class, ok := ioregIntProperty(block, "bInterfaceClass")
	if !ok || class != usbClassWirelessController {
		return false
	}
	subclass, ok := ioregIntProperty(block, "bInterfaceSubClass")
	if !ok || subclass != usbSubclassRNDIS {
		return false
	}
	protocol, ok := ioregIntProperty(block, "bInterfaceProtocol")
	return ok && protocol == usbProtocolRNDIS
}

// ioregIntProperty reads an unquoted integer ioreg property of the form
// `"key" = 224`. It complements ioregQuotedProperty, which reads string values.
func ioregIntProperty(block, key string) (int, bool) {
	prefix := `"` + key + `" = `
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		value, ok := strings.CutPrefix(line, prefix)
		if !ok {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}
