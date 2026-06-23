package clientcore

import (
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"errors"

	"continuity-vpn/internal/protocol"
)

// Session-key agreement. The two endpoints each generate an X25519 key pair,
// exchange public keys, and derive the same 32-byte AES-256-GCM session key via
// X25519 ECDH followed by HKDF-SHA256, salted by the session id. This replaces
// the pre-shared key assumed by ADR-0006 with a real key agreement (ADR-0007),
// using only the Go standard library — no invented cryptography.
//
// The derivation is symmetric: ECDH(myPriv, peerPub) yields the same shared
// secret on both sides, and both feed it through HKDF with the same salt (the
// session id) and info label, so they arrive at the same key. The key does NOT
// depend on role: direction separation for nonces is handled in the AEAD layer,
// not the key.
//
// Scope note: this is the key-derivation half. Authenticating the public keys
// (so a client knows it is talking to the real gateway, defeating an active
// man-in-the-middle) requires binding the gateway's static key — e.g. pinning it
// or carrying it in the signed entitlement — which is left to the transport that
// exchanges these public keys. See ADR-0007.
const hkdfInfo = "continuity-vpn session key v1"

var (
	errBadPrivateKey = errors.New("x25519 private key must be 32 bytes")
	errBadPublicKey  = errors.New("x25519 public key must be 32 bytes")
)

// GenerateKeyPair returns a fresh X25519 private and public key, each 32 bytes.
func GenerateKeyPair() (priv, pub []byte, err error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return key.Bytes(), key.PublicKey().Bytes(), nil
}

// DeriveSessionKey performs X25519 ECDH between myPriv and peerPub and derives
// the 32-byte session key with HKDF-SHA256 salted by the session id. Both
// endpoints, given their own private key and the other's public key, derive an
// identical key.
func DeriveSessionKey(myPriv, peerPub []byte, id protocol.SessionID) ([]byte, error) {
	curve := ecdh.X25519()
	priv, err := curve.NewPrivateKey(myPriv)
	if err != nil {
		return nil, errBadPrivateKey
	}
	pub, err := curve.NewPublicKey(peerPub)
	if err != nil {
		return nil, errBadPublicKey
	}

	shared, err := priv.ECDH(pub)
	if err != nil {
		return nil, err
	}
	return hkdf.Key(sha256.New, shared, id[:], hkdfInfo, keySize)
}
