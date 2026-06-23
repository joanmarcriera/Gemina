package darwin

import "testing"

func TestMacOSProductVersionTrimsSwVersOutput(t *testing.T) {
	runner := fakeCommandRunner{
		outputs: map[string][]byte{
			"sw_vers\x00-productVersion": []byte("14.5\n"),
		},
	}

	got, err := MacOSProductVersion(runner)
	if err != nil {
		t.Fatalf("MacOSProductVersion returned error: %v", err)
	}
	if got != "14.5" {
		t.Fatalf("version = %q, want %q", got, "14.5")
	}
}
