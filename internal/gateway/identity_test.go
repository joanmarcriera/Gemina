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
