package clientcore

import (
	"bytes"
	"testing"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

func testKey() []byte {
	return bytes.Repeat([]byte{0x2A}, keySize)
}

func testID(number protocol.PacketNumber) protocol.PacketID {
	var s protocol.SessionID
	copy(s[:], bytes.Repeat([]byte{0xCD}, protocol.SessionIDSize))
	return protocol.PacketID{Session: s, Number: number}
}

func TestSealerRoundTrip(t *testing.T) {
	s, err := newSealer(testKey())
	if err != nil {
		t.Fatalf("newSealer: %v", err)
	}
	id := testID(7)
	aad := []byte("header-bytes")
	plaintext := []byte("consistent network experience")

	ct := s.seal(dirInitiator, id, aad, plaintext)
	if bytes.Contains(ct, plaintext) {
		t.Fatal("ciphertext still contains the plaintext")
	}

	got, err := s.open(dirInitiator, id, aad, ct)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round-trip = %q, want %q", got, plaintext)
	}
}

func TestSealerRejectsTamper(t *testing.T) {
	s, _ := newSealer(testKey())
	id := testID(7)
	aad := []byte("aad")
	ct := s.seal(dirInitiator, id, aad, []byte("secret payload"))

	// Flip a ciphertext byte.
	tampered := append([]byte(nil), ct...)
	tampered[0] ^= 0x80
	if _, err := s.open(dirInitiator, id, aad, tampered); err == nil {
		t.Fatal("open accepted a tampered ciphertext")
	}

	// Tamper the authenticated header (identity binding).
	if _, err := s.open(dirInitiator, id, []byte("different-aad"), ct); err == nil {
		t.Fatal("open accepted mismatched AAD")
	}

	// A different packet number (nonce) must not verify.
	if _, err := s.open(dirInitiator, testID(8), aad, ct); err == nil {
		t.Fatal("open accepted a wrong-nonce ciphertext")
	}
}

func TestSealerRejectsWrongKey(t *testing.T) {
	a, _ := newSealer(testKey())
	other := bytes.Repeat([]byte{0x99}, keySize)
	b, _ := newSealer(other)

	id := testID(1)
	ct := a.seal(dirInitiator, id, nil, []byte("hi"))
	if _, err := b.open(dirInitiator, id, nil, ct); err == nil {
		t.Fatal("open accepted ciphertext sealed under a different key")
	}
}

func TestSealerDirectionPreventsNonceReuse(t *testing.T) {
	s, _ := newSealer(testKey())
	id := testID(1) // same identity, same key...

	initiator := s.seal(dirInitiator, id, nil, []byte("same-plaintext"))
	responder := s.seal(dirResponder, id, nil, []byte("same-plaintext"))

	// ...different direction must yield different ciphertext (distinct nonce),
	// so client#1 and gateway#1 never reuse a (key, nonce) pair.
	if bytes.Equal(initiator, responder) {
		t.Fatal("same identity in both directions produced identical ciphertext (nonce reuse)")
	}
}

func TestNewSealerRejectsBadKeyLength(t *testing.T) {
	if _, err := newSealer([]byte("short")); err == nil {
		t.Fatal("newSealer accepted a short key")
	}
}
