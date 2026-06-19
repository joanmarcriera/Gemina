package darwin

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"testing"

	"continuity-vpn/internal/paths"
)

func TestCollectInterfaceSnapshotsMapsNetworkState(t *testing.T) {
	got, err := CollectInterfaceSnapshots(fakeInterfaceSource{
		records: []InterfaceRecord{
			{
				BSDName:      "uplink-alpha",
				DisplayName:  "Wi-Fi service",
				Flags:        net.FlagUp | net.FlagRunning | net.FlagBroadcast,
				AddressCIDRs: []string{"192.0.2.10/24", "fe80::1/64"},
				Kind:         paths.LinkKindWiFi,
				Evidence: []Evidence{
					{Source: EvidenceSourceNetworkFramework, Key: "interface-type", Value: "wifi"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CollectInterfaceSnapshots returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("snapshot count = %d", len(got))
	}

	snapshot := got[0]
	if snapshot.BSDName != "uplink-alpha" {
		t.Fatalf("BSDName = %q", snapshot.BSDName)
	}
	if snapshot.DisplayName != "Wi-Fi service" {
		t.Fatalf("DisplayName = %q", snapshot.DisplayName)
	}
	if snapshot.Kind != paths.LinkKindWiFi {
		t.Fatalf("Kind = %s", snapshot.Kind)
	}
	if !snapshot.Up || !snapshot.Running || snapshot.Loopback || !snapshot.HasIPv4 {
		t.Fatalf("network state = %+v", snapshot)
	}

	if got := snapshot.EvidenceBySource(EvidenceSourceBSDNetworkState); len(got) != 4 {
		t.Fatalf("BSD network evidence count = %d", len(got))
	}
	assertEvidenceValue(t, snapshot, EvidenceSourceBSDNetworkState, "flag-up", "present")
	assertEvidenceValue(t, snapshot, EvidenceSourceBSDNetworkState, "flag-running", "present")
	assertEvidenceValue(t, snapshot, EvidenceSourceBSDNetworkState, "flag-loopback", "absent")
	assertEvidenceValue(t, snapshot, EvidenceSourceBSDNetworkState, "ipv4", "present")
	if got := snapshot.EvidenceBySource(EvidenceSourceNetworkFramework); len(got) != 1 {
		t.Fatalf("Network framework evidence count = %d", len(got))
	}
}

func TestCollectInterfaceSnapshotsDoesNotInferKindFromNames(t *testing.T) {
	got, err := CollectInterfaceSnapshots(fakeInterfaceSource{
		records: []InterfaceRecord{
			{
				BSDName:      "en0",
				DisplayName:  "Wi-Fi",
				Flags:        net.FlagUp | net.FlagRunning,
				AddressCIDRs: []string{"198.51.100.5/24"},
			},
			{
				BSDName:      "en7",
				DisplayName:  "Android USB tethering",
				Flags:        net.FlagUp | net.FlagRunning,
				AddressCIDRs: []string{"203.0.113.9/24"},
			},
		},
	})
	if err != nil {
		t.Fatalf("CollectInterfaceSnapshots returned error: %v", err)
	}

	observations := ObservationsFromSnapshots(got)
	classification := paths.Classify(observations)
	if classification.Complete() {
		t.Fatalf("classification inferred roles from names: %+v", classification)
	}
	if classification.Candidate(paths.RoleWiFi) != nil {
		t.Fatalf("unexpected Wi-Fi candidate from name-only evidence")
	}
	if classification.Candidate(paths.RoleAndroidUSBTether) != nil {
		t.Fatalf("unexpected Android USB candidate from name-only evidence")
	}
}

func TestCollectInterfaceSnapshotsDerivesKindFromExplicitEvidence(t *testing.T) {
	got, err := CollectInterfaceSnapshots(fakeInterfaceSource{
		records: []InterfaceRecord{
			loadInterfaceRecordFixture(t, "testdata/wifi-network-framework.json"),
			loadInterfaceRecordFixture(t, "testdata/android-usb-ioregistry.json"),
		},
	})
	if err != nil {
		t.Fatalf("CollectInterfaceSnapshots returned error: %v", err)
	}
	if got[0].Kind != paths.LinkKindWiFi {
		t.Fatalf("Wi-Fi snapshot kind = %s", got[0].Kind)
	}
	if got[1].Kind != paths.LinkKindAndroidUSBTether {
		t.Fatalf("Android USB snapshot kind = %s", got[1].Kind)
	}

	classification := paths.Classify(ObservationsFromSnapshots(got))
	if !classification.Complete() {
		t.Fatalf("classification incomplete from evidence-derived kinds: %+v", classification)
	}
}

func TestCollectInterfaceSnapshotsDerivesWiFiFromSystemConfigurationEvidence(t *testing.T) {
	got, err := CollectInterfaceSnapshots(fakeInterfaceSource{
		records: []InterfaceRecord{
			{
				BSDName:      "wireless-system-config",
				Flags:        net.FlagUp | net.FlagRunning,
				AddressCIDRs: []string{"192.0.2.55/24"},
				Evidence: []Evidence{
					{Source: EvidenceSourceSystemConfiguration, Key: "interface-type", Value: "IEEE80211"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CollectInterfaceSnapshots returned error: %v", err)
	}
	if got[0].Kind != paths.LinkKindWiFi {
		t.Fatalf("snapshot kind = %s", got[0].Kind)
	}
}

func TestCollectInterfaceSnapshotsLeavesGenericUSBNetworkUnknown(t *testing.T) {
	got, err := CollectInterfaceSnapshots(fakeInterfaceSource{
		records: []InterfaceRecord{
			loadInterfaceRecordFixture(t, "testdata/generic-usb-network-ioregistry.json"),
		},
	})
	if err != nil {
		t.Fatalf("CollectInterfaceSnapshots returned error: %v", err)
	}
	if got[0].Kind != paths.LinkKindUnknown {
		t.Fatalf("generic USB network kind = %s", got[0].Kind)
	}

	classification := paths.Classify(ObservationsFromSnapshots(got))
	if classification.Candidate(paths.RoleAndroidUSBTether) != nil {
		t.Fatalf("generic USB network produced Android candidate: %+v", classification)
	}
	assertPathIssue(t, classification, paths.RoleAndroidUSBTether, paths.IssueMissingCandidate)
}

func TestCollectInterfaceSnapshotsLeavesConflictingEvidenceUnknown(t *testing.T) {
	got, err := CollectInterfaceSnapshots(fakeInterfaceSource{
		records: []InterfaceRecord{
			{
				BSDName:      "conflicting",
				Flags:        net.FlagUp | net.FlagRunning,
				AddressCIDRs: []string{"198.51.100.44/24"},
				Evidence: []Evidence{
					{Source: EvidenceSourceNetworkFramework, Key: "interface-type", Value: "wifi"},
					{Source: EvidenceSourceIORegistry, Key: "usb-transport", Value: "android-rndis"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CollectInterfaceSnapshots returned error: %v", err)
	}
	if got[0].Kind != paths.LinkKindUnknown {
		t.Fatalf("conflicting evidence kind = %s", got[0].Kind)
	}
}

func TestCollectInterfaceSnapshotsTreatsIPv6OnlyAsNoIPv4(t *testing.T) {
	got, err := CollectInterfaceSnapshots(fakeInterfaceSource{
		records: []InterfaceRecord{
			{
				BSDName:      "ipv6-only",
				Flags:        net.FlagUp | net.FlagRunning,
				AddressCIDRs: []string{"2001:db8::1/64"},
				Kind:         paths.LinkKindWiFi,
			},
		},
	})
	if err != nil {
		t.Fatalf("CollectInterfaceSnapshots returned error: %v", err)
	}

	if got[0].HasIPv4 {
		t.Fatalf("IPv6-only snapshot reported IPv4: %+v", got[0])
	}
	if got[0].Observation().Usable() {
		t.Fatalf("IPv6-only observation should be unusable: %+v", got[0].Observation())
	}
}

func TestCollectInterfaceSnapshotsPropagatesSourceErrors(t *testing.T) {
	sourceErr := errors.New("source failed")

	_, err := CollectInterfaceSnapshots(fakeInterfaceSource{err: sourceErr})
	if !errors.Is(err, sourceErr) {
		t.Fatalf("CollectInterfaceSnapshots error = %v, want wrapped source error", err)
	}
}

func TestCollectInterfaceSnapshotsRejectsNilSource(t *testing.T) {
	_, err := CollectInterfaceSnapshots(nil)
	if !errors.Is(err, ErrNilInterfaceSource) {
		t.Fatalf("CollectInterfaceSnapshots(nil) error = %v", err)
	}
}

type fakeInterfaceSource struct {
	records []InterfaceRecord
	err     error
}

func (source fakeInterfaceSource) Interfaces() ([]InterfaceRecord, error) {
	if source.err != nil {
		return nil, source.err
	}
	return source.records, nil
}

func assertEvidenceValue(t *testing.T, snapshot InterfaceSnapshot, source EvidenceSource, key, want string) {
	t.Helper()
	for _, evidence := range snapshot.EvidenceBySource(source) {
		if evidence.Key == key {
			if evidence.Value != want {
				t.Fatalf("evidence %s value = %q, want %q", key, evidence.Value, want)
			}
			return
		}
	}
	t.Fatalf("missing evidence key %q in %+v", key, snapshot.EvidenceBySource(source))
}

func assertPathIssue(t *testing.T, classification paths.Classification, role paths.Role, code paths.IssueCode) {
	t.Helper()
	for _, issue := range classification.Issues {
		if issue.Role == role && issue.Code == code {
			return
		}
	}
	t.Fatalf("missing path issue role=%s code=%s in %+v", role, code, classification.Issues)
}

type interfaceRecordFixture struct {
	BSDName      string            `json:"bsd_name"`
	DisplayName  string            `json:"display_name"`
	Flags        []string          `json:"flags"`
	AddressCIDRs []string          `json:"address_cidrs"`
	Evidence     []evidenceFixture `json:"evidence"`
}

type evidenceFixture struct {
	Source string `json:"source"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

func loadInterfaceRecordFixture(t *testing.T, path string) InterfaceRecord {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}

	var fixture interfaceRecordFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("parse fixture %s: %v", path, err)
	}

	evidence := make([]Evidence, 0, len(fixture.Evidence))
	for _, item := range fixture.Evidence {
		evidence = append(evidence, Evidence{
			Source: evidenceSourceFromFixture(t, item.Source),
			Key:    item.Key,
			Value:  item.Value,
		})
	}

	return InterfaceRecord{
		BSDName:      fixture.BSDName,
		DisplayName:  fixture.DisplayName,
		Flags:        flagsFromFixture(t, fixture.Flags),
		AddressCIDRs: fixture.AddressCIDRs,
		Evidence:     evidence,
	}
}

func flagsFromFixture(t *testing.T, values []string) net.Flags {
	t.Helper()

	var flags net.Flags
	for _, value := range values {
		switch value {
		case "up":
			flags |= net.FlagUp
		case "running":
			flags |= net.FlagRunning
		case "loopback":
			flags |= net.FlagLoopback
		default:
			t.Fatalf("unknown fixture flag %q", value)
		}
	}
	return flags
}

func evidenceSourceFromFixture(t *testing.T, value string) EvidenceSource {
	t.Helper()

	switch value {
	case EvidenceSourceBSDNetworkState.String():
		return EvidenceSourceBSDNetworkState
	case EvidenceSourceSystemConfiguration.String():
		return EvidenceSourceSystemConfiguration
	case EvidenceSourceNetworkFramework.String():
		return EvidenceSourceNetworkFramework
	case EvidenceSourceIORegistry.String():
		return EvidenceSourceIORegistry
	default:
		t.Fatalf("unknown fixture evidence source %q", value)
		return EvidenceSourceUnknown
	}
}
