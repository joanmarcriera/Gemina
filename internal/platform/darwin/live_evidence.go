package darwin

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// CommandRunner abstracts macOS command execution so live evidence collection
// can be tested from redacted fixtures.
type CommandRunner interface {
	RunCommand(name string, args ...string) ([]byte, error)
}

type OSCommandRunner struct{}

func (OSCommandRunner) RunCommand(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

type InterfaceEvidenceRecord struct {
	BSDName  string
	Evidence []Evidence
}

type InterfaceEvidenceSource interface {
	InterfaceEvidence() ([]InterfaceEvidenceRecord, error)
}

type CombinedInterfaceSource struct {
	State           InterfaceSource
	EvidenceSources []InterfaceEvidenceSource
}

func LiveEvidenceInterfaceSnapshots() ([]InterfaceSnapshot, error) {
	return CollectInterfaceSnapshots(CombinedInterfaceSource{
		State: NetInterfaceSource{},
		EvidenceSources: []InterfaceEvidenceSource{
			SystemConfigurationCommandSource{Runner: OSCommandRunner{}},
			IORegistryCommandSource{Runner: OSCommandRunner{}},
		},
	})
}

func (source CombinedInterfaceSource) Interfaces() ([]InterfaceRecord, error) {
	if source.State == nil {
		return nil, ErrNilInterfaceSource
	}

	records, err := source.State.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("collect base interface state: %w", err)
	}

	evidenceByBSDName := make(map[string][]Evidence)
	for _, evidenceSource := range source.EvidenceSources {
		if evidenceSource == nil {
			continue
		}
		evidenceRecords, err := evidenceSource.InterfaceEvidence()
		if err != nil {
			return nil, fmt.Errorf("collect interface evidence: %w", err)
		}
		for _, evidenceRecord := range evidenceRecords {
			if evidenceRecord.BSDName == "" {
				continue
			}
			evidenceByBSDName[evidenceRecord.BSDName] = append(evidenceByBSDName[evidenceRecord.BSDName], evidenceRecord.Evidence...)
		}
	}

	for i := range records {
		records[i].Evidence = append(records[i].Evidence, evidenceByBSDName[records[i].BSDName]...)
	}
	return records, nil
}

type SystemConfigurationCommandSource struct {
	Runner CommandRunner
}

func (source SystemConfigurationCommandSource) InterfaceEvidence() ([]InterfaceEvidenceRecord, error) {
	runner := source.Runner
	if runner == nil {
		runner = OSCommandRunner{}
	}

	output, err := runner.RunCommand("networksetup", "-listallhardwareports")
	if err != nil {
		return nil, fmt.Errorf("networksetup list hardware ports: %w", err)
	}
	return parseNetworkSetupHardwarePorts(output), nil
}

type IORegistryCommandSource struct {
	Runner CommandRunner
}

func (source IORegistryCommandSource) InterfaceEvidence() ([]InterfaceEvidenceRecord, error) {
	runner := source.Runner
	if runner == nil {
		runner = OSCommandRunner{}
	}

	output, err := runner.RunCommand("ioreg", "-r", "-c", "IOEthernetInterface", "-l")
	if err != nil {
		return nil, fmt.Errorf("ioreg ethernet interfaces: %w", err)
	}
	return parseIORegistryEthernetInterfaces(output), nil
}

func parseNetworkSetupHardwarePorts(output []byte) []InterfaceEvidenceRecord {
	var records []InterfaceEvidenceRecord
	var hardwarePort, bsdName string

	flush := func() {
		if bsdName != "" && isWiFiHardwarePort(hardwarePort) {
			records = append(records, InterfaceEvidenceRecord{
				BSDName: bsdName,
				Evidence: []Evidence{
					{Source: EvidenceSourceSystemConfiguration, Key: "interface-type", Value: "wifi"},
				},
			})
		}
		hardwarePort = ""
		bsdName = ""
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			flush()
			continue
		}
		if value, ok := strings.CutPrefix(line, "Hardware Port:"); ok {
			flush()
			hardwarePort = strings.TrimSpace(value)
			continue
		}
		if value, ok := strings.CutPrefix(line, "Device:"); ok {
			bsdName = strings.TrimSpace(value)
		}
	}
	flush()

	return records
}

func parseIORegistryEthernetInterfaces(output []byte) []InterfaceEvidenceRecord {
	var records []InterfaceEvidenceRecord
	for _, block := range splitIORegistryBlocks(string(output)) {
		bsdName := ioregQuotedProperty(block, "BSD Name")
		if bsdName == "" || !blockHasAndroidUSBTetherEvidence(block) {
			continue
		}

		records = append(records, InterfaceEvidenceRecord{
			BSDName: bsdName,
			Evidence: []Evidence{
				{Source: EvidenceSourceIORegistry, Key: "usb-transport", Value: "android-rndis"},
			},
		})
	}
	return records
}

func splitIORegistryBlocks(output string) []string {
	var blocks []string
	var current strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "+-o ") && current.Len() > 0 {
			blocks = append(blocks, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	if current.Len() > 0 {
		blocks = append(blocks, current.String())
	}
	return blocks
}

func ioregQuotedProperty(block, key string) string {
	prefix := `"` + key + `" = "`
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		value := strings.TrimPrefix(line, prefix)
		if before, _, ok := strings.Cut(value, `"`); ok {
			return before
		}
	}
	return ""
}

func blockHasAndroidUSBTetherEvidence(block string) bool {
	normalised := normaliseEvidenceToken(block)
	return strings.Contains(normalised, "android") &&
		(strings.Contains(normalised, "rndis") || strings.Contains(normalised, "tether"))
}

func isWiFiHardwarePort(value string) bool {
	switch normaliseEvidenceToken(value) {
	case "wifi", "airport", "ieee80211":
		return true
	default:
		return false
	}
}
