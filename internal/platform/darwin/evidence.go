package darwin

import (
	"strings"
	"unicode"

	"continuity-vpn/internal/paths"
)

// Canonical Darwin evidence vocabulary. The live command sources emit these
// exact key/value strings and LinkKindFromEvidence matches them after
// normalisation. Keeping the vocabulary in one place stops the producing and
// consuming sides from silently drifting, which would otherwise downgrade a
// path to LinkKindUnknown with no error.
const (
	EvidenceKeyInterfaceType = "interface-type"
	EvidenceKeyUSBTransport  = "usb-transport"

	EvidenceValueWiFi         = "wifi"
	EvidenceValueAndroidRNDIS = "android-rndis"
)

// matchesEvidenceToken reports whether value normalises to the same token as any
// of the canonical strings, so callers can compare against the shared
// vocabulary regardless of punctuation or case in the source data.
func matchesEvidenceToken(value string, canonical ...string) bool {
	got := normaliseEvidenceToken(value)
	for _, candidate := range canonical {
		if got == normaliseEvidenceToken(candidate) {
			return true
		}
	}
	return false
}

// LinkKindFromEvidence derives a path link kind only from explicit macOS
// evidence. It returns unknown for missing or conflicting evidence.
func LinkKindFromEvidence(evidence []Evidence) paths.LinkKind {
	wifi := hasWiFiEvidence(evidence)
	androidUSB := hasAndroidUSBTetherEvidence(evidence)

	switch {
	case wifi && !androidUSB:
		return paths.LinkKindWiFi
	case androidUSB && !wifi:
		return paths.LinkKindAndroidUSBTether
	default:
		return paths.LinkKindUnknown
	}
}

func hasWiFiEvidence(evidence []Evidence) bool {
	for _, item := range evidence {
		switch item.Source {
		case EvidenceSourceNetworkFramework, EvidenceSourceSystemConfiguration:
			if matchesEvidenceToken(item.Key, EvidenceKeyInterfaceType) && isWiFiEvidenceValue(item.Value) {
				return true
			}
		}
	}
	return false
}

func hasAndroidUSBTetherEvidence(evidence []Evidence) bool {
	for _, item := range evidence {
		if item.Source != EvidenceSourceIORegistry {
			continue
		}
		// "device-kind" and "interface-association" are reserved for future
		// IORegistry collectors; no live source emits them yet.
		if matchesEvidenceToken(item.Key, EvidenceKeyUSBTransport, "device-kind", "interface-association") &&
			isAndroidUSBTetherEvidenceValue(item.Value) {
			return true
		}
	}
	return false
}

// isWiFiEvidenceValue reports whether value names Wi-Fi. It accepts the canonical
// token plus the equivalent macOS spellings seen in hardware-port and
// SystemConfiguration output.
func isWiFiEvidenceValue(value string) bool {
	return matchesEvidenceToken(value, EvidenceValueWiFi, "ieee80211", "airport")
}

func isAndroidUSBTetherEvidenceValue(value string) bool {
	return matchesEvidenceToken(value, EvidenceValueAndroidRNDIS, "android-usb-tether", "android-usb-tethering")
}

func normaliseEvidenceToken(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
