package darwin

import (
	"testing"

	"github.com/joanmarcriera/gemina/internal/paths"
)

// TestCanonicalEvidenceConstantsClassify guards that the exact key/value
// constants the live command sources emit are the ones LinkKindFromEvidence
// accepts. If a constant is changed on only one side this test fails instead of
// silently downgrading the path to LinkKindUnknown.
func TestCanonicalEvidenceConstantsClassify(t *testing.T) {
	tests := []struct {
		name     string
		evidence []Evidence
		want     paths.LinkKind
	}{
		{
			name: "wifi from system configuration",
			evidence: []Evidence{
				{Source: EvidenceSourceSystemConfiguration, Key: EvidenceKeyInterfaceType, Value: EvidenceValueWiFi},
			},
			want: paths.LinkKindWiFi,
		},
		{
			name: "wifi from network framework",
			evidence: []Evidence{
				{Source: EvidenceSourceNetworkFramework, Key: EvidenceKeyInterfaceType, Value: EvidenceValueWiFi},
			},
			want: paths.LinkKindWiFi,
		},
		{
			name: "android usb tether from io registry",
			evidence: []Evidence{
				{Source: EvidenceSourceIORegistry, Key: EvidenceKeyUSBTransport, Value: EvidenceValueAndroidRNDIS},
			},
			want: paths.LinkKindAndroidUSBTether,
		},
		{
			name: "matching ignores case and punctuation",
			evidence: []Evidence{
				{Source: EvidenceSourceIORegistry, Key: "USB Transport", Value: "Android RNDIS"},
			},
			want: paths.LinkKindAndroidUSBTether,
		},
		{
			name: "wrong source is not classified",
			evidence: []Evidence{
				{Source: EvidenceSourceIORegistry, Key: EvidenceKeyInterfaceType, Value: EvidenceValueWiFi},
			},
			want: paths.LinkKindUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LinkKindFromEvidence(tt.evidence); got != tt.want {
				t.Fatalf("LinkKindFromEvidence(%+v) = %s, want %s", tt.evidence, got, tt.want)
			}
		})
	}
}
