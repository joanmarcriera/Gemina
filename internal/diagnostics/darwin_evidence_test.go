package diagnostics

import (
	"encoding/json"
	"strings"
	"testing"

	"continuity-vpn/internal/paths"
	"continuity-vpn/internal/platform/darwin"
)

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
	})

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
	})

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
