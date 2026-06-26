package diagnostics

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/joanmarcriera/gemina/internal/paths"
	"github.com/joanmarcriera/gemina/internal/platform/darwin"
)

func TestBuildDarwinEvidenceReportSurfacesPresentButUnusableTether(t *testing.T) {
	// Wi-Fi is usable but the Android uplink only exists as an unclaimed USB
	// RNDIS function: present on the bus, no host driver, no NIC. The report must
	// show it honestly (usable=false) and explain the missing candidate WITHOUT
	// inventing a usable android-usb-tether candidate.
	report := BuildDarwinEvidenceReport(
		[]darwin.InterfaceSnapshot{
			{BSDName: "redacted-wifi", Kind: paths.LinkKindWiFi, Up: true, Running: true, HasIPv4: true},
		},
		[]darwin.USBTetherFunction{
			{Transport: darwin.EvidenceValueAndroidRNDIS, HostDriverClaimed: false},
		},
	)

	if report.Complete {
		t.Fatal("a present-but-unusable tether must not make the classification complete")
	}
	if len(report.DeviceFunctions) != 1 {
		t.Fatalf("device function count = %d, want 1: %+v", len(report.DeviceFunctions), report.DeviceFunctions)
	}
	fn := report.DeviceFunctions[0]
	if fn.Transport != darwin.EvidenceValueAndroidRNDIS || fn.Usable {
		t.Fatalf("device function = %+v, want android-rndis usable=false", fn)
	}
	for _, candidate := range report.Candidates {
		if candidate.Role == paths.RoleAndroidUSBTether.String() {
			t.Fatalf("present-but-unusable tether was promoted to a candidate: %+v", candidate)
		}
	}
	if !hasIssue(report.Issues, IssueCodeTetherPresentNotUsable, paths.RoleAndroidUSBTether.String()) {
		t.Fatalf("missing explanatory issue %q for the present-but-unusable tether: %+v", IssueCodeTetherPresentNotUsable, report.Issues)
	}
}

func hasIssue(issues []DiagnosticIssue, code, role string) bool {
	for _, issue := range issues {
		if issue.Code == code && issue.Role == role {
			return true
		}
	}
	return false
}

func TestBuildDarwinEvidenceReportRedactsAndReportsClassification(t *testing.T) {
	report := BuildDarwinEvidenceReport([]darwin.InterfaceSnapshot{
		{
			BSDName:     "redacted-wifi",
			DisplayName: "raw display name should not appear",
			Kind:        paths.LinkKindWiFi,
			Up:          true,
			Running:     true,
			HasIPv4:     true,
			Evidence: []darwin.Evidence{
				{Source: darwin.EvidenceSourceSystemConfiguration, Key: "interface-type", Value: "wifi"},
			},
		},
		{
			BSDName: "redacted-android-usb",
			Kind:    paths.LinkKindAndroidUSBTether,
			Up:      true,
			Running: true,
			HasIPv4: true,
			Evidence: []darwin.Evidence{
				{Source: darwin.EvidenceSourceIORegistry, Key: "usb-transport", Value: "android-rndis"},
			},
		},
	}, nil)

	if report.Type != DarwinEvidenceReportType {
		t.Fatalf("Type = %q", report.Type)
	}
	if report.Claim != "diagnostic-only-not-path-success" {
		t.Fatalf("Claim = %q", report.Claim)
	}
	if !report.Complete || report.ClassificationStatus != "complete" {
		t.Fatalf("classification status = %s complete=%v", report.ClassificationStatus, report.Complete)
	}
	if len(report.Candidates) != 2 {
		t.Fatalf("candidate count = %d", len(report.Candidates))
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	output := string(encoded)
	for _, forbidden := range []string{
		"raw display name",
		"00:00:00:00:00:00",
		"Android RNDIS",
		"USB 10/100/1000 LAN",
		"cv1_",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("report leaked %q: %s", forbidden, output)
		}
	}
}

func TestBuildDarwinEvidenceReportMarksIncompleteClassification(t *testing.T) {
	report := BuildDarwinEvidenceReport([]darwin.InterfaceSnapshot{
		{
			BSDName: "redacted-wifi",
			Kind:    paths.LinkKindWiFi,
			Up:      true,
			Running: true,
			HasIPv4: true,
		},
	}, nil)

	if report.Complete {
		t.Fatal("report unexpectedly complete")
	}
	if report.ClassificationStatus != "incomplete" {
		t.Fatalf("classification status = %q", report.ClassificationStatus)
	}
	if len(report.Issues) != 1 {
		t.Fatalf("issue count = %d", len(report.Issues))
	}
	if report.Issues[0].Role != paths.RoleAndroidUSBTether.String() {
		t.Fatalf("issue role = %q", report.Issues[0].Role)
	}
}
