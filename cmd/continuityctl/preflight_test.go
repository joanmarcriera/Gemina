package main

import (
	"bytes"
	"strings"
	"testing"

	"continuity-vpn/internal/diagnostics"
	"continuity-vpn/internal/platform/darwin"
)

func TestWritePreflightSummaryMode(t *testing.T) {
	report := diagnostics.BuildCompatibilityReport([]darwin.InterfaceSnapshot{
		{BSDName: "redacted-wifi", Kind: 0},
	}, nil, "14.5")

	var buf bytes.Buffer
	if err := writePreflight(&buf, report, false, false); err != nil {
		t.Fatalf("writePreflight: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, report.Summary()) {
		t.Fatalf("summary mode output %q missing summary", out)
	}
	if strings.Contains(out, "{") {
		t.Fatalf("summary mode should not print JSON: %q", out)
	}
}

func TestWritePreflightJSONMode(t *testing.T) {
	report := diagnostics.BuildCompatibilityReport(nil, nil, "14.5")

	var buf bytes.Buffer
	if err := writePreflight(&buf, report, true, false); err != nil {
		t.Fatalf("writePreflight: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, diagnostics.CompatibilityReportType) {
		t.Fatalf("JSON mode output missing report type: %q", out)
	}
	if !strings.Contains(out, "next_step") {
		t.Fatalf("JSON mode output missing next_step field: %q", out)
	}
}
