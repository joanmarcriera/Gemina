package main

import (
	"os"
	"testing"
)

func TestParseRunAs(t *testing.T) {
	tests := []struct {
		spec     string
		uid, gid int
		wantErr  bool
	}{
		{spec: "65532:65532", uid: 65532, gid: 65532},
		{spec: "65532", uid: 65532, gid: 65532},
		{spec: " 1000:2000 ", uid: 1000, gid: 2000},
		{spec: "", wantErr: true},
		{spec: "0:0", wantErr: true},       // root is not a drop
		{spec: "65532:0", wantErr: true},   // root group is not a drop
		{spec: "nonroot", wantErr: true},   // distroless has no user db: numeric only
		{spec: "65532:abc", wantErr: true}, // bad gid
		{spec: "-5:65532", wantErr: true},  // negative uid
		{spec: "65532:65532:1", wantErr: true},
	}
	for _, tt := range tests {
		uid, gid, err := parseRunAs(tt.spec)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseRunAs(%q) = %d:%d, want error", tt.spec, uid, gid)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseRunAs(%q) error: %v", tt.spec, err)
			continue
		}
		if uid != tt.uid || gid != tt.gid {
			t.Errorf("parseRunAs(%q) = %d:%d, want %d:%d", tt.spec, uid, gid, tt.uid, tt.gid)
		}
	}
}

// TestDropPrivilegesNonRootIsNoop pins the contract that an already
// unprivileged process is left untouched (the probe image runs as :nonroot).
// The actual root->65532 drop is exercised on the gateway host by the data unit
// (the process refuses to start if the drop cannot be verified).
func TestDropPrivilegesNonRootIsNoop(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a drop would change the test process credentials")
	}
	dropped, err := dropPrivileges(65532, 65532)
	if err != nil || dropped {
		t.Fatalf("dropPrivileges as non-root = (%v, %v), want (false, nil)", dropped, err)
	}
}
