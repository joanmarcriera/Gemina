package darwin

import (
	"errors"
	"net"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/joanmarcriera/gemina/internal/paths"
)

func TestCombinedInterfaceSourceAddsLiveEvidenceWithoutRawIdentifiers(t *testing.T) {
	runner := fakeCommandRunner{
		outputs: map[string][]byte{
			"networksetup\x00-listallhardwareports":          readTestFixture(t, "testdata/networksetup-listallhardwareports.txt"),
			"ioreg\x00-r\x00-c\x00IOEthernetInterface\x00-l": readTestFixture(t, "testdata/ioreg-ioethernet-redacted.txt"),
		},
	}

	got, err := CollectInterfaceSnapshots(CombinedInterfaceSource{
		State: fakeInterfaceSource{records: []InterfaceRecord{
			{BSDName: "redacted-wifi", Flags: net.FlagUp | net.FlagRunning, AddressCIDRs: []string{"192.0.2.10/24"}},
			{BSDName: "redacted-android-usb", Flags: net.FlagUp | net.FlagRunning, AddressCIDRs: []string{"203.0.113.20/24"}},
			{BSDName: "redacted-usb-network", Flags: net.FlagUp | net.FlagRunning, AddressCIDRs: []string{"203.0.113.30/24"}},
		}},
		EvidenceSources: []InterfaceEvidenceSource{
			SystemConfigurationCommandSource{Runner: runner},
			IORegistryCommandSource{Runner: runner},
		},
	})
	if err != nil {
		t.Fatalf("CollectInterfaceSnapshots returned error: %v", err)
	}

	kinds := map[string]paths.LinkKind{}
	for _, snapshot := range got {
		kinds[snapshot.BSDName] = snapshot.Kind
		assertNoRawHardwareIdentifiers(t, snapshot.Evidence)
	}
	if kinds["redacted-wifi"] != paths.LinkKindWiFi {
		t.Fatalf("Wi-Fi kind = %s", kinds["redacted-wifi"])
	}
	if kinds["redacted-android-usb"] != paths.LinkKindAndroidUSBTether {
		t.Fatalf("Android USB kind = %s", kinds["redacted-android-usb"])
	}
	if kinds["redacted-usb-network"] != paths.LinkKindUnknown {
		t.Fatalf("generic USB kind = %s", kinds["redacted-usb-network"])
	}

	classification := paths.Classify(ObservationsFromSnapshots(got))
	if !classification.Complete() {
		t.Fatalf("classification incomplete: %+v", classification)
	}
}

func TestSystemConfigurationCommandSourceIgnoresNonWiFiHardwarePorts(t *testing.T) {
	got := parseNetworkSetupHardwarePorts([]byte(`Hardware Port: USB 10/100/1000 LAN
Device: redacted-usb-network
Ethernet Address: 00:00:00:00:00:00
`))
	if len(got) != 0 {
		t.Fatalf("generic networksetup hardware port produced evidence: %+v", got)
	}
}

func TestIORegistryCommandSourceLeavesGenericUSBUnknown(t *testing.T) {
	got := parseIORegistryEthernetInterfaces([]byte(`+-o IOEthernetInterface@1
  "BSD Name" = "redacted-usb-network"
  "USB Product Name" = "USB 10/100/1000 LAN"
  "USB Vendor Name" = "Generic"
`))
	if len(got) != 0 {
		t.Fatalf("generic IORegistry USB network produced evidence: %+v", got)
	}
}

func TestCombinedInterfaceSourcePropagatesEvidenceErrors(t *testing.T) {
	sourceErr := errors.New("command failed")

	_, err := CollectInterfaceSnapshots(CombinedInterfaceSource{
		State: fakeInterfaceSource{records: []InterfaceRecord{
			{BSDName: "redacted-wifi", Flags: net.FlagUp | net.FlagRunning, AddressCIDRs: []string{"192.0.2.10/24"}},
		}},
		EvidenceSources: []InterfaceEvidenceSource{
			failingEvidenceSource{err: sourceErr},
		},
	})
	if !errors.Is(err, sourceErr) {
		t.Fatalf("CollectInterfaceSnapshots error = %v, want wrapped evidence error", err)
	}
}

func TestCombinedInterfaceSourceRejectsNilStateSource(t *testing.T) {
	_, err := CollectInterfaceSnapshots(CombinedInterfaceSource{})
	if !errors.Is(err, ErrNilInterfaceSource) {
		t.Fatalf("CollectInterfaceSnapshots error = %v, want nil source error", err)
	}
}

type fakeCommandRunner struct {
	outputs map[string][]byte
	err     error
}

func (runner fakeCommandRunner) RunCommand(name string, args ...string) ([]byte, error) {
	if runner.err != nil {
		return nil, runner.err
	}

	keyParts := append([]string{name}, args...)
	key := strings.Join(keyParts, "\x00")
	output, ok := runner.outputs[key]
	if !ok {
		return nil, errors.New("unexpected command")
	}
	return output, nil
}

type failingEvidenceSource struct {
	err error
}

func (source failingEvidenceSource) InterfaceEvidence() ([]InterfaceEvidenceRecord, error) {
	return nil, source.err
}

func readTestFixture(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return data
}

func assertNoRawHardwareIdentifiers(t *testing.T, evidence []Evidence) {
	t.Helper()

	for _, item := range evidence {
		if strings.Contains(item.Key, "address") {
			t.Fatalf("evidence key stores address-like data: %+v", item)
		}
		if strings.Contains(item.Key, "serial") {
			t.Fatalf("evidence key stores serial-like data: %+v", item)
		}
		if slices.Contains([]string{"00:00:00:00:00:00", "Android RNDIS", "USB 10/100/1000 LAN"}, item.Value) {
			t.Fatalf("evidence value stores raw fixture data: %+v", item)
		}
	}
}
