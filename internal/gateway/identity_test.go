package gateway

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateIdentityCreatesThenReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity.key")

	priv, created, err := LoadOrCreateIdentity(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !created {
		t.Fatal("first call should have created the identity")
	}
	if len(priv) != ed25519.PrivateKeySize {
		t.Fatalf("key size = %d", len(priv))
	}

	// Reloading returns the same key and does not recreate it.
	priv2, created2, err := LoadOrCreateIdentity(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if created2 {
		t.Fatal("second call should have loaded, not created")
	}
	if !priv.Equal(priv2) {
		t.Fatal("reloaded identity differs from the created one")
	}
}

func TestLoadOrCreateIdentityRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.key")
	if err := os.WriteFile(path, []byte("too short"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadOrCreateIdentity(path); err == nil {
		t.Fatal("expected error for a wrong-size identity file")
	}
}

func TestLoadOrCreateIdentityRefusesDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "elsewhere.key")
	path := filepath.Join(dir, "identity.key")
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if _, _, err := LoadOrCreateIdentity(path); err == nil {
		t.Fatal("expected error creating identity through a dangling symlink")
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("symlink target must not be created, lstat err = %v", err)
	}
}

func TestLoadOrCreateIdentityCreatesOwnerOnlyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity.key")
	if _, _, err := LoadOrCreateIdentity(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Fatalf("identity mode = %o, want no group/other bits", perm)
	}
}
