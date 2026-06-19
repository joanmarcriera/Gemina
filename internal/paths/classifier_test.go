package paths

import "testing"

func TestClassifySelectsUsableWiFiAndAndroidUSB(t *testing.T) {
	got := Classify([]Observation{
		usable("coffee-shop-uplink", "Train Wi-Fi", LinkKindWiFi),
		usable("phone-rndis-path", "Android USB tethering", LinkKindAndroidUSBTether),
	})

	if !got.Complete() {
		t.Fatalf("classification incomplete: %+v", got)
	}
	if candidate := got.Candidate(RoleWiFi); candidate == nil || candidate.Observation.ID != "coffee-shop-uplink" {
		t.Fatalf("Wi-Fi candidate = %+v", candidate)
	}
	if candidate := got.Candidate(RoleAndroidUSBTether); candidate == nil || candidate.Observation.ID != "phone-rndis-path" {
		t.Fatalf("Android USB candidate = %+v", candidate)
	}
}

func TestClassifyDoesNotDependOnInterfaceNames(t *testing.T) {
	got := Classify([]Observation{
		usable("alpha-wireless-path", "arbitrary wireless label", LinkKindWiFi),
		usable("beta-phone-path", "arbitrary phone label", LinkKindAndroidUSBTether),
	})

	if !got.Complete() {
		t.Fatalf("classification incomplete for non-standard names: %+v", got)
	}
}

func TestClassifyIgnoresUnusableObservations(t *testing.T) {
	tests := []struct {
		name        string
		observation Observation
	}{
		{name: "empty ID", observation: Observation{Kind: LinkKindWiFi, Up: true, Running: true, HasIPv4: true}},
		{name: "down", observation: Observation{ID: "wifi", Kind: LinkKindWiFi, Running: true, HasIPv4: true}},
		{name: "not running", observation: Observation{ID: "wifi", Kind: LinkKindWiFi, Up: true, HasIPv4: true}},
		{name: "loopback", observation: Observation{ID: "wifi", Kind: LinkKindWiFi, Up: true, Running: true, Loopback: true, HasIPv4: true}},
		{name: "no IPv4", observation: Observation{ID: "wifi", Kind: LinkKindWiFi, Up: true, Running: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify([]Observation{
				tt.observation,
				usable("phone", "Android", LinkKindAndroidUSBTether),
			})

			assertIssue(t, got, RoleWiFi, IssueMissingCandidate, 0)
		})
	}
}

func TestClassifyReportsMissingCandidate(t *testing.T) {
	got := Classify([]Observation{
		usable("wifi-a", "Wi-Fi", LinkKindWiFi),
	})

	assertIssue(t, got, RoleAndroidUSBTether, IssueMissingCandidate, 0)
	if got.Complete() {
		t.Fatal("classification unexpectedly complete")
	}
}

func TestClassifyReportsAmbiguousWiFiCandidates(t *testing.T) {
	got := Classify([]Observation{
		usable("wifi-a", "Wi-Fi A", LinkKindWiFi),
		usable("wifi-b", "Wi-Fi B", LinkKindWiFi),
		usable("phone", "Android", LinkKindAndroidUSBTether),
	})

	assertIssue(t, got, RoleWiFi, IssueAmbiguousCandidate, 2)
	if candidate := got.Candidate(RoleWiFi); candidate != nil {
		t.Fatalf("ambiguous role produced candidate: %+v", candidate)
	}
}

func TestClassifyReportsAmbiguousAndroidUSBTetherCandidates(t *testing.T) {
	got := Classify([]Observation{
		usable("wifi", "Wi-Fi", LinkKindWiFi),
		usable("phone-a", "Android A", LinkKindAndroidUSBTether),
		usable("phone-b", "Android B", LinkKindAndroidUSBTether),
	})

	assertIssue(t, got, RoleAndroidUSBTether, IssueAmbiguousCandidate, 2)
	if candidate := got.Candidate(RoleAndroidUSBTether); candidate != nil {
		t.Fatalf("ambiguous role produced candidate: %+v", candidate)
	}
}

func TestClassifyIgnoresUnknownKinds(t *testing.T) {
	got := Classify([]Observation{
		usable("ethernet", "Ethernet", LinkKindUnknown),
		usable("wifi", "Wi-Fi", LinkKindWiFi),
		usable("phone", "Android", LinkKindAndroidUSBTether),
	})

	if !got.Complete() {
		t.Fatalf("classification incomplete: %+v", got)
	}
}

func usable(id, displayName string, kind LinkKind) Observation {
	return Observation{
		ID:          id,
		DisplayName: displayName,
		Kind:        kind,
		Up:          true,
		Running:     true,
		HasIPv4:     true,
	}
}

func assertIssue(t *testing.T, classification Classification, role Role, code IssueCode, count int) {
	t.Helper()
	for _, issue := range classification.Issues {
		if issue.Role == role && issue.Code == code && issue.Count == count {
			return
		}
	}
	t.Fatalf("missing issue role=%s code=%s count=%d in %+v", role, code, count, classification.Issues)
}
