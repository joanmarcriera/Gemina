package diagnostics

import (
	"encoding/json"
	"strings"
	"testing"

	"continuity-vpn/internal/paths"
	"continuity-vpn/internal/platform/darwin"
)

func usableWiFi() darwin.InterfaceSnapshot {
	return darwin.InterfaceSnapshot{
		BSDName: "redacted-wifi",
		Kind:    paths.LinkKindWiFi,
		Up:      true,
		Running: true,
		HasIPv4: true,
	}
}

// A native NCM tether that macOS claimed as a usable enX (rare: Pixel/AOSP).
func usableAndroidNIC() darwin.InterfaceSnapshot {
	return darwin.InterfaceSnapshot{
		BSDName: "redacted-android-usb",
		Kind:    paths.LinkKindAndroidUSBTether,
		Up:      true,
		Running: true,
		HasIPv4: true,
	}
}

// The common case: an RNDIS tether function present on USB but not a NIC. The
// app's userspace driver makes it usable, so this must count as supported.
func rndisFunction() darwin.USBTetherFunction {
	return darwin.USBTetherFunction{Transport: darwin.EvidenceValueAndroidRNDIS, HostDriverClaimed: false}
}

func TestBuildCompatibilityReport(t *testing.T) {
	tests := []struct {
		name        string
		snapshots   []darwin.InterfaceSnapshot
		functions   []darwin.USBTetherFunction
		macOS       string
		wantVerdict CompatibilityVerdict
		wantReady   bool
		wantMode    string
	}{
		{
			name:        "wifi plus rndis function is supported via app driver",
			snapshots:   []darwin.InterfaceSnapshot{usableWiFi()},
			functions:   []darwin.USBTetherFunction{rndisFunction()},
			macOS:       "14.5",
			wantVerdict: VerdictSupported,
			wantReady:   true,
			wantMode:    TetherModeAppDriverRNDIS,
		},
		{
			name:        "wifi plus native ncm nic is supported natively",
			snapshots:   []darwin.InterfaceSnapshot{usableWiFi(), usableAndroidNIC()},
			functions:   nil,
			macOS:       "26.0",
			wantVerdict: VerdictSupported,
			wantReady:   true,
			wantMode:    TetherModeNativeNCM,
		},
		{
			name:        "wifi but no android is needs-android",
			snapshots:   []darwin.InterfaceSnapshot{usableWiFi()},
			functions:   nil,
			macOS:       "14.5",
			wantVerdict: VerdictNeedsAndroid,
			wantReady:   false,
			wantMode:    TetherModeNone,
		},
		{
			name:        "android present but no wifi is needs-wifi",
			snapshots:   nil,
			functions:   []darwin.USBTetherFunction{rndisFunction()},
			macOS:       "14.5",
			wantVerdict: VerdictNeedsWiFi,
			wantReady:   false,
			wantMode:    TetherModeAppDriverRNDIS,
		},
		{
			name:        "nothing present is needs-both",
			snapshots:   nil,
			functions:   nil,
			macOS:       "14.5",
			wantVerdict: VerdictNeedsBoth,
			wantReady:   false,
			wantMode:    TetherModeNone,
		},
		{
			name:        "old macOS is unsupported regardless of devices",
			snapshots:   []darwin.InterfaceSnapshot{usableWiFi()},
			functions:   []darwin.USBTetherFunction{rndisFunction()},
			macOS:       "10.15.7",
			wantVerdict: VerdictUnsupportedMacOS,
			wantReady:   false,
			wantMode:    TetherModeAppDriverRNDIS,
		},
		{
			name:        "unparseable macOS version is unsupported",
			snapshots:   []darwin.InterfaceSnapshot{usableWiFi()},
			functions:   []darwin.USBTetherFunction{rndisFunction()},
			macOS:       "",
			wantVerdict: VerdictUnsupportedMacOS,
			wantReady:   false,
			wantMode:    TetherModeAppDriverRNDIS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := BuildCompatibilityReport(tt.snapshots, tt.functions, tt.macOS)

			if report.Verdict != tt.wantVerdict {
				t.Errorf("verdict = %q, want %q", report.Verdict, tt.wantVerdict)
			}
			if report.Ready != tt.wantReady {
				t.Errorf("ready = %v, want %v", report.Ready, tt.wantReady)
			}
			if report.AndroidTether.Mode != tt.wantMode {
				t.Errorf("tether mode = %q, want %q", report.AndroidTether.Mode, tt.wantMode)
			}
			if report.NextStep == "" {
				t.Error("next step must never be empty — it is the user's action")
			}
			if report.Type != CompatibilityReportType {
				t.Errorf("type = %q", report.Type)
			}
		})
	}
}

func TestCompatibilityReportSummaryIsHumanAndCarriesNextStep(t *testing.T) {
	report := BuildCompatibilityReport(
		[]darwin.InterfaceSnapshot{usableWiFi()},
		nil,
		"14.5",
	)
	summary := report.Summary()
	if !strings.Contains(summary, string(VerdictNeedsAndroid)) {
		t.Errorf("summary %q missing verdict", summary)
	}
	if !strings.Contains(summary, report.NextStep) {
		t.Errorf("summary %q missing the next step", summary)
	}
}

func TestCompatibilityReportReadyImpliesActionableAndRedacted(t *testing.T) {
	report := BuildCompatibilityReport(
		[]darwin.InterfaceSnapshot{usableWiFi()},
		[]darwin.USBTetherFunction{rndisFunction()},
		"14.5",
	)
	if !report.Ready {
		t.Fatal("expected ready")
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out := string(encoded)
	// The report must carry only coarse tokens — never a raw bsd name leaks via
	// the verdict surface (those live in the evidence report, not the verdict).
	for _, forbidden := range []string{"redacted-wifi", "redacted-android-usb"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("compatibility report leaked %q: %s", forbidden, out)
		}
	}
}
