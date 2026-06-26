package clientcore

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

// Packet encryption for the data plane. Each session has a 32-byte key and uses
// AES-256-GCM (an authenticated cipher from the standard library — no invented
// cryptography). The nonce is derived deterministically from the packet identity
// plus a direction byte, and the frame header is the additional authenticated
// data, so:
//
//   - Both duplicate copies of one logical packet share an identity, hence the
//     same nonce and identical ciphertext — dual-path duplication and dedup by
//     identity keep working untouched.
//   - Nonces never repeat under a key: the packet number is unique and monotonic
//     per session per direction, and the direction byte keeps the two endpoints
//     (which share the session key) from colliding on number 1, 2, 3, …
//   - Anti-replay is provided by the existing dedup window; the AEAD provides
//     confidentiality and integrity, binding each ciphertext to its identity.
//
// Key agreement (a handshake to establish the per-session key) is out of scope
// here: the key is supplied to the session (a pre-shared key for now). See
// ADR-0006.
const (
	keySize   = 32 // AES-256
	nonceSize = 12 // GCM standard nonce

	dirInitiator byte = 0 // client -> gateway
	dirResponder byte = 1 // gateway -> client
)

var errKeySize = errors.New("session key must be 32 bytes")

type sealer struct {
	aead cipher.AEAD
}

func newSealer(key []byte) (*sealer, error) {
	if len(key) != keySize {
		return nil, errKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &sealer{aead: aead}, nil
}

// nonce derives the 12-byte GCM nonce from the direction and packet number:
// byte 0 is the direction, bytes 1..3 are zero, bytes 4..11 are the big-endian
// packet number. Unique per (key, direction, number).
func nonce(dir byte, number protocol.PacketNumber) []byte {
	var n [nonceSize]byte
	n[0] = dir
	binary.BigEndian.PutUint64(n[4:], uint64(number))
	return n[:]
}

// seal encrypts plaintext for the given direction and identity, authenticating
// aad (the frame header). Returns ciphertext||tag.
func (s *sealer) seal(dir byte, id protocol.PacketID, aad, plaintext []byte) []byte {
	return s.aead.Seal(nil, nonce(dir, id.Number), plaintext, aad)
}

// open authenticates and decrypts ciphertext for the given direction and
// identity. It fails if the ciphertext, the aad, the direction or the identity
// do not match what was sealed.
func (s *sealer) open(dir byte, id protocol.PacketID, aad, ciphertext []byte) ([]byte, error) {
	return s.aead.Open(nil, nonce(dir, id.Number), ciphertext, aad)
}
