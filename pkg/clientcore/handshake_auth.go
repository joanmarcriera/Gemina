package clientcore

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

// Gateway authentication for the key agreement (ADR-0007). DeriveSessionKey gives
// confidentiality and forward secrecy but, on its own, no protection against an
// active man-in-the-middle who substitutes its own ephemeral key. We close that
// gap the way TLS 1.3 authenticates a server: the gateway holds a long-term
// Ed25519 identity, the client pins its public key, and the gateway signs its
// ephemeral X25519 key (bound to the session) with that identity. A client that
// pins the real gateway key rejects any substituted key, because an attacker can
// only sign with its own identity.
//
// This authenticates the gateway to the client. The client is authenticated to
// the gateway by its entitlement token (internal/entitlement), so no client
// signature is needed here. Wiring the on-wire handshake message that carries the
// ephemeral key + signature remains an integration step.
//
// handshakeContext is the domain-separation label prefixed to every signed
// transcript so a signature cannot be repurposed in another context.
const handshakeContext = "continuity-vpn handshake v1"

var (
	errBadIdentityKey  = errors.New("ed25519 identity public key has the wrong size")
	errBadSignature    = errors.New("handshake signature does not verify")
	errMalformedSigLen = errors.New("handshake signature has the wrong size")
)

// GenerateIdentity returns a long-term Ed25519 identity for a gateway. The
// public half is pinned by clients; the private half stays on the gateway.
func GenerateIdentity() (priv ed25519.PrivateKey, pub ed25519.PublicKey, err error) {
	pub, priv, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

// transcript is the exact byte string signed and verified: a fixed context
// label, the session id, and the ephemeral public key. Binding the session id
// stops a signature being replayed into a different session.
func transcript(ephemeralPub []byte, id protocol.SessionID) []byte {
	msg := make([]byte, 0, len(handshakeContext)+protocol.SessionIDSize+len(ephemeralPub))
	msg = append(msg, handshakeContext...)
	msg = append(msg, id[:]...)
	msg = append(msg, ephemeralPub...)
	return msg
}

// SignHandshake signs the gateway's ephemeral X25519 public key, bound to the
// session, with the gateway's Ed25519 identity.
func SignHandshake(identityPriv ed25519.PrivateKey, ephemeralPub []byte, id protocol.SessionID) []byte {
	return ed25519.Sign(identityPriv, transcript(ephemeralPub, id))
}

// VerifyHandshake checks that ephemeralPub was signed for this session by the
// holder of identityPub (the pinned gateway identity). It returns nil only on a
// valid signature.
func VerifyHandshake(identityPub ed25519.PublicKey, ephemeralPub, sig []byte, id protocol.SessionID) error {
	if len(identityPub) != ed25519.PublicKeySize {
		return errBadIdentityKey
	}
	if len(sig) != ed25519.SignatureSize {
		return errMalformedSigLen
	}
	if !ed25519.Verify(identityPub, transcript(ephemeralPub, id), sig) {
		return errBadSignature
	}
	return nil
}
