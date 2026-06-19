package darwin

import (
	"strings"
	"unicode"

	"continuity-vpn/internal/paths"
)

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
			if normaliseEvidenceToken(item.Key) == "interfacetype" && isWiFiEvidenceValue(item.Value) {
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

		switch normaliseEvidenceToken(item.Key) {
		case "usbtransport", "devicekind", "interfaceassociation":
			if isAndroidUSBTetherEvidenceValue(item.Value) {
				return true
			}
		}
	}
	return false
}

func isWiFiEvidenceValue(value string) bool {
	switch normaliseEvidenceToken(value) {
	case "wifi", "ieee80211", "airport":
		return true
	default:
		return false
	}
}

func isAndroidUSBTetherEvidenceValue(value string) bool {
	switch normaliseEvidenceToken(value) {
	case "androidrndis", "androidusbtether", "androidusbtethering":
		return true
	default:
		return false
	}
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
