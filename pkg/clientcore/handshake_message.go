package clientcore

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"time"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

// DefaultHandshakeTolerance bounds how far a ClientHello timestamp may be from
// the gateway's clock. It limits how long a captured ClientHello (with a still
// valid token) can be replayed to open sessions, independent of the token's TTL.
const DefaultHandshakeTolerance = 60 * time.Second

// ErrStaleHandshake is returned for a ClientHello whose timestamp is outside the
// tolerance window in either direction.
var ErrStaleHandshake = errors.New("handshake timestamp outside tolerance window")

// CheckHandshakeFresh reports whether a ClientHello timestamp is within tolerance
// of now (in either direction).
func CheckHandshakeFresh(timestamp int64, now time.Time, tolerance time.Duration) error {
	skew := now.Sub(time.Unix(timestamp, 0))
	if skew < 0 {
		skew = -skew
	}
	if skew > tolerance {
		return ErrStaleHandshake
	}
	return nil
}

// randomSessionID fills id with cryptographically-random bytes, ensuring it is
// non-zero (a zero session id is invalid).
func randomSessionID(id *protocol.SessionID) error {
	if _, err := rand.Read(id[:]); err != nil {
		return err
	}
	id[0] |= 1
	return nil
}

// On-wire handshake messages (CVH1). A two-message exchange establishes a
// session before any data flows:
//
//	ClientHello (client -> gateway): session id, the client's ephemeral X25519
//	  public key, and the client's entitlement token.
//	ServerHello (gateway -> client): the gateway's ephemeral X25519 public key
//	  and an Ed25519 signature over it (bound to the session) under the gateway's
//	  pinned identity.
//
// Both sides then derive the session key by X25519 ECDH of their own ephemeral
// private key with the peer's ephemeral public key (DeriveSessionKey). The client
// authenticates the gateway by verifying the signature against the pinned
// identity (ADR-0007); the gateway authenticates/admits the client by its token.
const (
	handshakeVersion = 1
	msgClientHello   = 1
	msgServerHello   = 2

	x25519PubSize    = 32
	maxTokenLen      = 4096
	assignedIPv4Size = 4

	// ClientHello: header + session id + 8-byte timestamp + ephemeral key +
	// 2-byte token length, then the token.
	clientHelloFixed = 6 + protocol.SessionIDSize + 8 + x25519PubSize + 2 // before token
	// ServerHello: header + session id + ephemeral key + signature + the
	// gateway-assigned tunnel IPv4 (4 bytes, appended after the signature).
	// The IPv4 is NOT covered by the gateway signature (which binds only the
	// ephemeral key to the session, ADR-0007); tampering with it can only break
	// the client's own tunnel-IP config (a denial of service a network attacker
	// can already cause), never the AEAD-authenticated data plane. A zero value
	// means "unassigned" (e.g. the gateway exit is disabled).
	serverHelloLen = 6 + protocol.SessionIDSize + x25519PubSize + ed25519.SignatureSize + assignedIPv4Size
)

var handshakeMagic = [4]byte{'C', 'V', 'H', '1'}

var (
	errShortHandshake = errors.New("handshake message shorter than its header")
	errHandshakeMagic = errors.New("handshake message has wrong magic")
	errHandshakeVer   = errors.New("handshake message has unsupported version")
	errHandshakeType  = errors.New("handshake message has the wrong type")
	errHandshakeField = errors.New("handshake message has a malformed field")
)

func putHandshakeHeader(out []byte, msgType byte, id protocol.SessionID) {
	copy(out[0:4], handshakeMagic[:])
	out[4] = handshakeVersion
	out[5] = msgType
	copy(out[6:6+protocol.SessionIDSize], id[:])
}

func parseHandshakeHeader(b []byte, msgType byte) (protocol.SessionID, error) {
	if len(b) < 6+protocol.SessionIDSize {
		return protocol.SessionID{}, errShortHandshake
	}
	if [4]byte(b[0:4]) != handshakeMagic {
		return protocol.SessionID{}, errHandshakeMagic
	}
	if b[4] != handshakeVersion {
		return protocol.SessionID{}, errHandshakeVer
	}
	if b[5] != msgType {
		return protocol.SessionID{}, errHandshakeType
	}
	var id protocol.SessionID
	copy(id[:], b[6:6+protocol.SessionIDSize])
	return id, nil
}

// EncodeClientHello frames a ClientHello. timestamp is the client's Unix-seconds
// clock, which the gateway checks for freshness to bound replay.
func EncodeClientHello(id protocol.SessionID, timestamp int64, ephemeralPub []byte, token string) ([]byte, error) {
	if len(ephemeralPub) != x25519PubSize {
		return nil, errHandshakeField
	}
	if len(token) > maxTokenLen {
		return nil, errHandshakeField
	}
	out := make([]byte, clientHelloFixed+len(token))
	putHandshakeHeader(out, msgClientHello, id)
	tsStart := 6 + protocol.SessionIDSize
	binary.BigEndian.PutUint64(out[tsStart:], uint64(timestamp))
	ephStart := tsStart + 8
	copy(out[ephStart:], ephemeralPub)
	binary.BigEndian.PutUint16(out[ephStart+x25519PubSize:], uint16(len(token)))
	copy(out[clientHelloFixed:], token)
	return out, nil
}

// DecodeClientHello parses a ClientHello.
func DecodeClientHello(b []byte) (id protocol.SessionID, timestamp int64, ephemeralPub []byte, token string, err error) {
	id, err = parseHandshakeHeader(b, msgClientHello)
	if err != nil {
		return protocol.SessionID{}, 0, nil, "", err
	}
	if len(b) < clientHelloFixed {
		return protocol.SessionID{}, 0, nil, "", errShortHandshake
	}
	tsStart := 6 + protocol.SessionIDSize
	timestamp = int64(binary.BigEndian.Uint64(b[tsStart:]))
	ephStart := tsStart + 8
	eph := b[ephStart : ephStart+x25519PubSize]
	tokenLen := int(binary.BigEndian.Uint16(b[ephStart+x25519PubSize:]))
	if len(b) != clientHelloFixed+tokenLen || tokenLen > maxTokenLen {
		return protocol.SessionID{}, 0, nil, "", errHandshakeField
	}
	return id, timestamp, append([]byte(nil), eph...), string(b[clientHelloFixed:]), nil
}

// EncodeServerHello frames a ServerHello. assignedIPv4 is the gateway-leased
// tunnel address for the client (zero = unassigned).
func EncodeServerHello(id protocol.SessionID, ephemeralPub, sig []byte, assignedIPv4 [4]byte) ([]byte, error) {
	if len(ephemeralPub) != x25519PubSize || len(sig) != ed25519.SignatureSize {
		return nil, errHandshakeField
	}
	out := make([]byte, serverHelloLen)
	putHandshakeHeader(out, msgServerHello, id)
	ephStart := 6 + protocol.SessionIDSize
	copy(out[ephStart:], ephemeralPub)
	sigStart := ephStart + x25519PubSize
	copy(out[sigStart:], sig)
	copy(out[sigStart+ed25519.SignatureSize:], assignedIPv4[:])
	return out, nil
}

// DecodeServerHello parses a ServerHello, returning the gateway's ephemeral key,
// its signature, and the assigned tunnel IPv4 (zero = unassigned).
func DecodeServerHello(b []byte) (id protocol.SessionID, ephemeralPub, sig []byte, assignedIPv4 [4]byte, err error) {
	id, err = parseHandshakeHeader(b, msgServerHello)
	if err != nil {
		return protocol.SessionID{}, nil, nil, [4]byte{}, err
	}
	if len(b) != serverHelloLen {
		return protocol.SessionID{}, nil, nil, [4]byte{}, errHandshakeField
	}
	ephStart := 6 + protocol.SessionIDSize
	eph := b[ephStart : ephStart+x25519PubSize]
	sigStart := ephStart + x25519PubSize
	signature := b[sigStart : sigStart+ed25519.SignatureSize]
	var ip [4]byte
	copy(ip[:], b[sigStart+ed25519.SignatureSize:])
	return id, append([]byte(nil), eph...), append([]byte(nil), signature...), ip, nil
}

// ClientHandshake is the client's in-flight handshake state between sending a
// ClientHello and receiving the ServerHello.
type ClientHandshake struct {
	sessionID       protocol.SessionID
	ephemeralPriv   []byte
	gatewayIdentity ed25519.PublicKey
}

// BeginClientHandshake starts a handshake to a gateway whose Ed25519 identity is
// pinned in gatewayIdentity, presenting the entitlement token. It returns the
// ClientHello bytes to send and the state needed to Complete the handshake.
func BeginClientHandshake(gatewayIdentity ed25519.PublicKey, token string) ([]byte, *ClientHandshake, error) {
	if len(gatewayIdentity) != ed25519.PublicKeySize {
		return nil, nil, errBadIdentityKey
	}
	var id protocol.SessionID
	if err := randomSessionID(&id); err != nil {
		return nil, nil, err
	}
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}
	hello, err := EncodeClientHello(id, time.Now().Unix(), pub, token)
	if err != nil {
		return nil, nil, err
	}
	return hello, &ClientHandshake{sessionID: id, ephemeralPriv: priv, gatewayIdentity: gatewayIdentity}, nil
}

// Complete consumes the gateway's ServerHello: it checks the session matches,
// verifies the gateway signature against the pinned identity (defeating an active
// MITM), derives the session key, and returns a ready initiator Session with the
// given inbound dedup capacity.
func (hs *ClientHandshake) Complete(serverHello []byte, dedupCapacity int) (*Session, error) {
	id, gatewayEph, sig, assignedIPv4, err := DecodeServerHello(serverHello)
	if err != nil {
		return nil, err
	}
	if id != hs.sessionID {
		return nil, errHandshakeField
	}
	if err := VerifyHandshake(hs.gatewayIdentity, gatewayEph, sig, id); err != nil {
		return nil, err
	}
	key, err := DeriveSessionKey(hs.ephemeralPriv, gatewayEph, id)
	if err != nil {
		return nil, err
	}
	session, err := NewSession(id, key, RoleInitiator, dedupCapacity)
	if err != nil {
		return nil, err
	}
	session.assignedIPv4 = assignedIPv4
	return session, nil
}
