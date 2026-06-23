package clientcore

import (
	"encoding/binary"
	"errors"

	"continuity-vpn/internal/protocol"
)

// Continuity VPN data-plane wire format. A data packet carries a variable
// encrypted payload (one tunnelled IP packet) after a fixed header. The header
// is cleartext and authenticated (it is the AEAD additional data) so the gateway
// can deduplicate by identity without decrypting; it carries no host
// identifiers, only a session + packet number and a direction bit.
//
//	offset 0  magic   "CVD1" (4 bytes)
//	offset 4  version 1 byte (dataVersion)
//	offset 5  flags   1 byte (bit 0 = direction: 0 initiator, 1 responder)
//	offset 6  session 16 bytes (SessionID)
//	offset 22 number  8 bytes, big-endian (PacketNumber)
//	offset 30 payload variable (AES-256-GCM ciphertext||tag of the plaintext)
const (
	dataVersion    = 1
	dataHeaderSize = 6 + protocol.SessionIDSize + 8 // 30
	flagDirection  = 0x01
	// maxPayload bounds a single tunnelled packet's plaintext. 1500 (Ethernet
	// MTU) plus a little headroom; the caller fragments above this.
	maxPayload = 1600
)

var dataMagic = [4]byte{'C', 'V', 'D', '1'}

var (
	errShortData    = errors.New("data datagram shorter than header")
	errBadMagic     = errors.New("data datagram has wrong magic")
	errBadVersion   = errors.New("data datagram has unsupported version")
	errBadIdentity  = errors.New("data datagram carries an invalid packet identity")
	errBadDirection = errors.New("data datagram has the wrong direction for this endpoint")
	errOversize     = errors.New("payload exceeds maximum size")
)

// frameHeader builds the 30-byte cleartext header for an identity and direction.
// It doubles as the AEAD additional-authenticated-data, so any tampering with
// the identity or direction makes decryption fail.
func frameHeader(id protocol.PacketID, dir byte) []byte {
	out := make([]byte, dataHeaderSize)
	copy(out[0:4], dataMagic[:])
	out[4] = dataVersion
	if dir == dirResponder {
		out[5] = flagDirection
	}
	copy(out[6:6+protocol.SessionIDSize], id.Session[:])
	binary.BigEndian.PutUint64(out[6+protocol.SessionIDSize:], uint64(id.Number))
	return out
}

// SessionIDFromDatagram reads the session id from a CVD1 datagram so a gateway
// can pick the per-session key/state before decrypting. It validates the magic,
// version and that the id is non-zero.
func SessionIDFromDatagram(b []byte) (protocol.SessionID, error) {
	id, _, _, err := parseHeader(b)
	if err != nil {
		return protocol.SessionID{}, err
	}
	return id.Session, nil
}

// parseHeader validates and decodes a datagram's header, returning the identity,
// direction, and the offset at which the encrypted payload begins.
func parseHeader(b []byte) (protocol.PacketID, byte, int, error) {
	if len(b) < dataHeaderSize {
		return protocol.PacketID{}, 0, 0, errShortData
	}
	if [4]byte(b[0:4]) != dataMagic {
		return protocol.PacketID{}, 0, 0, errBadMagic
	}
	if b[4] != dataVersion {
		return protocol.PacketID{}, 0, 0, errBadVersion
	}

	dir := dirInitiator
	if b[5]&flagDirection != 0 {
		dir = dirResponder
	}

	var session protocol.SessionID
	copy(session[:], b[6:6+protocol.SessionIDSize])
	number := protocol.PacketNumber(binary.BigEndian.Uint64(b[6+protocol.SessionIDSize:]))
	id := protocol.PacketID{Session: session, Number: number}
	if !id.Valid() {
		return protocol.PacketID{}, 0, 0, errBadIdentity
	}
	return id, dir, dataHeaderSize, nil
}
