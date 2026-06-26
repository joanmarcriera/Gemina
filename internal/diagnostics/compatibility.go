package diagnostics

import (
	"strconv"
	"strings"

	"github.com/joanmarcriera/gemina/internal/paths"
	"github.com/joanmarcriera/gemina/internal/platform/darwin"
)

const CompatibilityReportType = "stage-1-compatibility"

// minMacOSMajor is the oldest macOS major version the product targets. macOS 11
// (Big Sur) is the floor: it is the first with native CDC-NCM host support, and
// the userspace USB + NetworkExtension paths the product relies on are stable
// from there. Pre-11 (e.g. 10.15 Catalina) is reported unsupported.
const minMacOSMajor = 11

// CompatibilityVerdict is the single, plain-language answer a prospective user
// needs before buying or installing: will this Mac + phone combination work, and
// if not, what is the one thing to change.
type CompatibilityVerdict string

const (
	VerdictSupported        CompatibilityVerdict = "supported"
	VerdictNeedsAndroid     CompatibilityVerdict = "needs-android"
	VerdictNeedsWiFi        CompatibilityVerdict = "needs-wifi"
	VerdictNeedsBoth        CompatibilityVerdict = "needs-both"
	VerdictUnsupportedMacOS CompatibilityVerdict = "unsupported-macos"
)

// Android tether modes. RNDIS is universal across Android and driven by the
// app's bundled userspace driver, so its mere presence means "supported" — this
// is what delivers all-Android support with minimal friction. Native NCM is a
// bonus on phones (e.g. Pixel) that macOS claims without any app driver.
const (
	TetherModeNone           = "none"
	TetherModeNativeNCM      = "native-ncm"
	TetherModeAppDriverRNDIS = "app-driver-rndis"
)

// AndroidTetherSupport summarises how (if at all) the connected Android phone can
// serve as the second uplink.
type AndroidTetherSupport struct {
	Present bool   `json:"present"`
	Usable  bool   `json:"usable"` // usable now without the app driver (native NCM)
	Mode    string `json:"mode"`
}

// CompatibilityReport is the onboarding/pre-purchase verdict. It carries only
// coarse tokens (no bsd names, IPs or serials) so it is safe to show in a UI or
// on a website.
type CompatibilityReport struct {
	Type           string               `json:"type"`
	Verdict        CompatibilityVerdict `json:"verdict"`
	Ready          bool                 `json:"ready"`
	MacOSVersion   string               `json:"macos_version"`
	MacOSSupported bool                 `json:"macos_supported"`
	WiFi           bool                 `json:"wifi"`
	AndroidTether  AndroidTetherSupport `json:"android_tether"`
	NextStep       string               `json:"next_step"`
}

// Summary is a single human-readable line for a CLI or log: the verdict and the
// one action to take.
func (r CompatibilityReport) Summary() string {
	return string(r.Verdict) + ": " + r.NextStep
}

// ShareReport renders a compact, copy-pasteable block for the community
// compatibility catalogue. It carries only the coarse technical facts the
// catalogue needs (verdict, macOS version, tether mode) — never a bsd name, IP,
// MAC or serial — and explicitly invites the user to add their own phone model
// when submitting, so we never auto-collect a device identifier.
func (r CompatibilityReport) ShareReport() string {
	return strings.Join([]string{
		"Gemina VPN compatibility report (redacted — safe to share)",
		"  verdict:        " + string(r.Verdict),
		"  macOS:          " + r.MacOSVersion,
		"  android tether: " + r.AndroidTether.Mode,
		"  --- you fill these in when submitting to the catalogue ---",
		"  phone model:    (e.g. OnePlus 12R, Pixel 8 — your call what to share)",
		"  notes:          ",
	}, "\n")
}

// BuildCompatibilityReport derives the verdict from live device evidence plus the
// macOS version string (as from `sw_vers -productVersion`). It is pure so the
// whole matrix is unit-testable without hardware.
func BuildCompatibilityReport(snapshots []darwin.InterfaceSnapshot, deviceFunctions []darwin.USBTetherFunction, macOSVersion string) CompatibilityReport {
	classification := paths.Classify(darwin.ObservationsFromSnapshots(snapshots))
	wifi := classification.Candidate(paths.RoleWiFi) != nil
	tether := androidTetherSupport(classification, deviceFunctions)
	macOSMajor, macOSOK := parseMacOSMajor(macOSVersion)
	macOSSupported := macOSOK && macOSMajor >= minMacOSMajor

	verdict := decideVerdict(macOSSupported, wifi, tether.Present)

	report := CompatibilityReport{
		Type:           CompatibilityReportType,
		Verdict:        verdict,
		Ready:          verdict == VerdictSupported,
		MacOSVersion:   macOSVersion,
		MacOSSupported: macOSSupported,
		WiFi:           wifi,
		AndroidTether:  tether,
		NextStep:       nextStep(verdict, macOSMajor, macOSOK),
	}
	return report
}

func androidTetherSupport(classification paths.Classification, deviceFunctions []darwin.USBTetherFunction) AndroidTetherSupport {
	// A native NCM tether that macOS already claimed shows up as a usable
	// android-usb-tether candidate: no app driver needed.
	if classification.Candidate(paths.RoleAndroidUSBTether) != nil {
		return AndroidTetherSupport{Present: true, Usable: true, Mode: TetherModeNativeNCM}
	}
	// Otherwise an RNDIS function on the USB bus is enough: the app's userspace
	// driver makes it a usable uplink. This is the any-Android path.
	for _, fn := range deviceFunctions {
		if fn.Transport == darwin.EvidenceValueAndroidRNDIS {
			return AndroidTetherSupport{Present: true, Usable: false, Mode: TetherModeAppDriverRNDIS}
		}
	}
	return AndroidTetherSupport{Present: false, Usable: false, Mode: TetherModeNone}
}

func decideVerdict(macOSSupported, wifi, androidPresent bool) CompatibilityVerdict {
	switch {
	case !macOSSupported:
		return VerdictUnsupportedMacOS
	case wifi && androidPresent:
		return VerdictSupported
	case wifi && !androidPresent:
		return VerdictNeedsAndroid
	case !wifi && androidPresent:
		return VerdictNeedsWiFi
	default:
		return VerdictNeedsBoth
	}
}

func nextStep(verdict CompatibilityVerdict, macOSMajor int, macOSOK bool) string {
	switch verdict {
	case VerdictSupported:
		return "You're good to go: Wi-Fi and an Android tether are both available."
	case VerdictNeedsAndroid:
		return "Connect your Android phone by USB and turn on USB tethering."
	case VerdictNeedsWiFi:
		return "Join a Wi-Fi network so there is a second path to bond with."
	case VerdictNeedsBoth:
		return "Connect to Wi-Fi and plug in your Android phone with USB tethering on."
	case VerdictUnsupportedMacOS:
		if !macOSOK {
			return "Could not determine your macOS version; this product needs macOS 11 (Big Sur) or later."
		}
		return "Update macOS to 11 (Big Sur) or later to use this product."
	default:
		return "Run the compatibility check again."
	}
}

// parseMacOSMajor extracts the major version from a sw_vers product version like
// "14.5" or "10.15.7". Returns ok=false for an empty or unparseable string.
func parseMacOSMajor(version string) (int, bool) {
	version = strings.TrimSpace(version)
	if version == "" {
		return 0, false
	}
	major, _, _ := strings.Cut(version, ".")
	n, err := strconv.Atoi(major)
	if err != nil {
		return 0, false
	}
	return n, true
}
