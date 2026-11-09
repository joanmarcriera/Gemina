package paths

import "testing"

// TestEnumStrings pins the stable wire/log names of the path enums, including
// the "unknown" fallback for zero and out-of-range values, since they surface in
// diagnostics JSON.
func TestEnumStrings(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"LinkKindUnknown", LinkKindUnknown.String(), "unknown"},
		{"LinkKindWiFi", LinkKindWiFi.String(), "wi-fi"},
		{"LinkKindAndroidUSBTether", LinkKindAndroidUSBTether.String(), "android-usb-tether"},
		{"LinkKind(200)", LinkKind(200).String(), "unknown"},
		{"Role(0)", Role(0).String(), "unknown"},
		{"RoleWiFi", RoleWiFi.String(), "wi-fi"},
		{"RoleAndroidUSBTether", RoleAndroidUSBTether.String(), "android-usb-tether"},
		{"Role(200)", Role(200).String(), "unknown"},
		{"IssueCode(0)", IssueCode(0).String(), "unknown"},
		{"IssueMissingCandidate", IssueMissingCandidate.String(), "missing-candidate"},
		{"IssueAmbiguousCandidate", IssueAmbiguousCandidate.String(), "ambiguous-candidate"},
		{"IssueCode(200)", IssueCode(200).String(), "unknown"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s.String() = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
