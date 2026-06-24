package gateway

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
)

// LoadOrCreateIdentity loads the gateway's long-term Ed25519 identity from path,
// or generates and persists one if the file does not exist. The returned bool is
// true when a new identity was created. The private key file is written with
// owner-only permissions; clients pin the corresponding public key.
func LoadOrCreateIdentity(path string) (ed25519.PrivateKey, bool, error) {
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if len(data) != ed25519.PrivateKeySize {
			return nil, false, fmt.Errorf("identity file %s is %d bytes, want %d", path, len(data), ed25519.PrivateKeySize)
		}
		return ed25519.PrivateKey(data), false, nil
	case errors.Is(err, os.ErrNotExist):
		_, priv, genErr := ed25519.GenerateKey(rand.Reader)
		if genErr != nil {
			return nil, false, genErr
		}
		if writeErr := os.WriteFile(path, priv, 0o600); writeErr != nil {
			return nil, false, writeErr
		}
		return priv, true, nil
	default:
		return nil, false, err
	}
}
