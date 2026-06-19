package darwin

import (
	"testing"

	"continuity-vpn/internal/paths"
)

func TestObservationsFromSnapshotsPreservesObservationFields(t *testing.T) {
	got := ObservationsFromSnapshots([]InterfaceSnapshot{
		{
			BSDName:     "wireless-alpha",
			DisplayName: "Wi-Fi",
			Kind:        paths.LinkKindWiFi,
			Up:          true,
			Running:     true,
			HasIPv4:     true,
			Evidence: []Evidence{
				{Source: EvidenceSourceSystemConfiguration, Key: "interface-type", Value: "IEEE80211"},
			},
		},
	})

	if len(got) != 1 {
		t.Fatalf("observation count = %d", len(got))
	}
	if got[0].ID != "wireless-alpha" {
		t.Fatalf("ID = %q", got[0].ID)
	}
	if got[0].Kind != paths.LinkKindWiFi {
		t.Fatalf("Kind = %s", got[0].Kind)
	}
	if !got[0].Usable() {
		t.Fatalf("observation should be usable: %+v", got[0])
	}
}

func TestObservationsFromSnapshotsCanFeedPathClassification(t *testing.T) {
	observations := ObservationsFromSnapshots([]InterfaceSnapshot{
		{
			BSDName:     "wireless-alpha",
			DisplayName: "Wi-Fi",
			Kind:        paths.LinkKindWiFi,
			Up:          true,
			Running:     true,
			HasIPv4:     true,
			Evidence: []Evidence{
				{Source: EvidenceSourceNetworkFramework, Key: "interface-type", Value: "wifi"},
			},
		},
		{
			BSDName:     "phone-beta",
			DisplayName: "Android USB tethering",
			Kind:        paths.LinkKindAndroidUSBTether,
			Up:          true,
			Running:     true,
			HasIPv4:     true,
			Evidence: []Evidence{
				{Source: EvidenceSourceIORegistry, Key: "usb-transport", Value: "android-rndis"},
			},
		},
	})

	classification := paths.Classify(observations)
	if !classification.Complete() {
		t.Fatalf("classification incomplete: %+v", classification)
	}
}

func TestKnownBSDNamesStillRequireExplicitLinkKind(t *testing.T) {
	observations := ObservationsFromSnapshots([]InterfaceSnapshot{
		{
			BSDName: "en0",
			Kind:    paths.LinkKindWiFi,
			Up:      true,
			Running: true,
			HasIPv4: true,
		},
		{
			BSDName: "en7",
			Kind:    paths.LinkKindAndroidUSBTether,
			Up:      true,
			Running: true,
			HasIPv4: true,
		},
	})

	classification := paths.Classify(observations)
	if !classification.Complete() {
		t.Fatalf("classification incomplete: %+v", classification)
	}
}

func TestSnapshotNamesDoNotAssignLinkKind(t *testing.T) {
	observations := ObservationsFromSnapshots([]InterfaceSnapshot{
		{
			BSDName:     "en0",
			DisplayName: "Looks like Wi-Fi",
			Kind:        paths.LinkKindUnknown,
			Up:          true,
			Running:     true,
			HasIPv4:     true,
		},
		{
			BSDName:     "en7",
			DisplayName: "Looks like Android",
			Kind:        paths.LinkKindUnknown,
			Up:          true,
			Running:     true,
			HasIPv4:     true,
		},
	})

	classification := paths.Classify(observations)
	if classification.Complete() {
		t.Fatalf("classification used names as roles: %+v", classification)
	}
	if classification.Candidate(paths.RoleWiFi) != nil {
		t.Fatalf("unexpected Wi-Fi candidate from name-only evidence")
	}
	if classification.Candidate(paths.RoleAndroidUSBTether) != nil {
		t.Fatalf("unexpected Android USB candidate from name-only evidence")
	}
}

func TestEvidenceBySourceFiltersSnapshotMetadata(t *testing.T) {
	snapshot := InterfaceSnapshot{
		BSDName: "phone-beta",
		Kind:    paths.LinkKindAndroidUSBTether,
		Evidence: []Evidence{
			{Source: EvidenceSourceSystemConfiguration, Key: "interface-type", Value: "ethernet"},
			{Source: EvidenceSourceIORegistry, Key: "usb-transport", Value: "android-rndis"},
			{Source: EvidenceSourceIORegistry, Key: "vendor", Value: "android"},
		},
	}

	got := snapshot.EvidenceBySource(EvidenceSourceIORegistry)
	if len(got) != 2 {
		t.Fatalf("IORegistry evidence count = %d", len(got))
	}
	if got[0].Key != "usb-transport" || got[0].Value != "android-rndis" {
		t.Fatalf("first evidence = %+v", got[0])
	}
}

func TestMissingBSDNameRemainsUnusable(t *testing.T) {
	got := InterfaceSnapshot{
		DisplayName: "Wi-Fi",
		Kind:        paths.LinkKindWiFi,
		Up:          true,
		Running:     true,
		HasIPv4:     true,
	}.Observation()

	if got.Usable() {
		t.Fatalf("observation without BSD name should be unusable: %+v", got)
	}
}

func TestEvidenceSourceStrings(t *testing.T) {
	tests := map[EvidenceSource]string{
		EvidenceSourceUnknown:             "unknown",
		EvidenceSourceBSDNetworkState:     "bsd-network-state",
		EvidenceSourceSystemConfiguration: "system-configuration",
		EvidenceSourceNetworkFramework:    "network-framework",
		EvidenceSourceIORegistry:          "io-registry",
	}

	for source, want := range tests {
		if got := source.String(); got != want {
			t.Fatalf("%v.String() = %q, want %q", uint8(source), got, want)
		}
	}
}
