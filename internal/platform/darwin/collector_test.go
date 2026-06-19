package darwin

import (
	"errors"
	"net"
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
