package darwin

import (
	"os"
	"strings"
	"testing"
)

// TestUSBFunctionDeviceSourceLiveDetectsConnectedTether runs the real ioreg USB
// query against whatever is plugged in. It is skipped unless CONTINUITY_LIVE_USB
// is set, so CI and offline runs are unaffected; set it with an Android phone
// USB-tethered to confirm the detector sees a real RNDIS function.
func TestUSBFunctionDeviceSourceLiveDetectsConnectedTether(t *testing.T) {
	if os.Getenv("CONTINUITY_LIVE_USB") == "" {
		t.Skip("set CONTINUITY_LIVE_USB=1 with an Android phone USB-tethered to run")
	}

	got, err := USBFunctionDeviceSource{}.TetherFunctions()
	if err != nil {
		t.Fatalf("live TetherFunctions returned error: %v", err)
	}
	t.Logf("live USB tether functions: %+v", got)
	if len(got) == 0 {
		t.Fatalf("no tether function detected; expected one android-rndis function")
	}
}

func TestParseUSBHostInterfaceFunctionsDetectsAndroidRNDISFromClassNotVendor(t *testing.T) {
	got := parseUSBHostInterfaceFunctions(readTestFixture(t, "testdata/ioreg-usbhostinterface-rndis.txt"))

	if len(got) != 1 {
		t.Fatalf("got %d tether function(s), want exactly 1 (the RNDIS control interface): %+v", len(got), got)
	}
	fn := got[0]
	if fn.Transport != EvidenceValueAndroidRNDIS {
		t.Fatalf("transport = %q, want %q", fn.Transport, EvidenceValueAndroidRNDIS)
	}
	if fn.HostDriverClaimed {
		t.Fatalf("HostDriverClaimed = true; an unclaimed RNDIS function has no host driver and must not look usable")
	}
}

func TestParseUSBHostInterfaceFunctionsIgnoresCDCAndHubs(t *testing.T) {
	// A CDC-NCM USB ethernet adapter (control class 2) and a plain hub (class 9)
	// must not be mistaken for an Android RNDIS tether: only the RNDIS control
	// signature (class 224 / subclass 1 / protocol 3) counts.
	got := parseUSBHostInterfaceFunctions([]byte(`+-o AppleUSBNCMControl@0  <class IOUSBHostInterface, id 0xffffff0001>
      "bInterfaceClass" = 2
      "bInterfaceSubClass" = 13
      "bInterfaceProtocol" = 0
+-o IOUSBHostInterface@0  <class IOUSBHostInterface, id 0xffffff0002>
      "bInterfaceClass" = 9
      "bInterfaceSubClass" = 0
`))
	if len(got) != 0 {
		t.Fatalf("CDC/hub interfaces produced tether evidence: %+v", got)
	}
}

func TestParseUSBHostInterfaceFunctionsLeaksNoRawIdentifiers(t *testing.T) {
	// The fixture embeds product/vendor strings; detection keys on interface
	// class only, so no raw identifier may survive into the coarse token.
	for _, fn := range parseUSBHostInterfaceFunctions(readTestFixture(t, "testdata/ioreg-usbhostinterface-rndis.txt")) {
		if strings.Contains(fn.Transport, "redacted-") || strings.Contains(fn.Transport, "phone") {
			t.Fatalf("tether function transport leaked a raw identifier: %q", fn.Transport)
		}
	}
}

func TestUSBFunctionDeviceSourceRunsUSBLayerQuery(t *testing.T) {
	runner := fakeCommandRunner{
		outputs: map[string][]byte{
			"ioreg\x00-r\x00-c\x00IOUSBHostInterface\x00-l": readTestFixture(t, "testdata/ioreg-usbhostinterface-rndis.txt"),
		},
	}

	got, err := USBFunctionDeviceSource{Runner: runner}.TetherFunctions()
	if err != nil {
		t.Fatalf("TetherFunctions returned error: %v", err)
	}
	if len(got) != 1 || got[0].Transport != EvidenceValueAndroidRNDIS {
		t.Fatalf("TetherFunctions = %+v, want one android-rndis function", got)
	}
}
