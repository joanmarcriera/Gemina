package clientcore

import (
	"encoding/binary"
	"errors"

	"continuity-vpn/internal/protocol"
)

// Continuity VPN data-plane wire format. Unlike the fixed-size probe (CVP1), a
// data packet carries a variable payload (one tunnelled IP packet) after a
// fixed header. It deliberately carries no host identifiers — only a session +
// packet number so the peer can deduplicate copies that arrive over multiple
// paths.
//
//	offset 0  magic   "CVD1" (4 bytes)
//	offset 4  version 1 byte (dataVersion)
//	offset 5  flags   1 byte (reserved, must be 0)
//	offset 6  session 16 bytes (SessionID)
//	offset 22 number  8 bytes, big-endian (PacketNumber)
//	offset 30 payload variable
const (
	dataVersion    = 1
	dataHeaderSize = 6 + protocol.SessionIDSize + 8 // 30
	// maxPayload bounds a single tunnelled packet. 1500 (Ethernet MTU) plus a
	// little headroom; the caller fragments above this.
	maxPayload = 1600
)

var dataMagic = [4]byte{'C', 'V', 'D', '1'}

var (
	errShortData   = errors.New("data datagram shorter than header")
	errBadMagic    = errors.New("data datagram has wrong magic")
	errBadVersion  = errors.New("data datagram has unsupported version")
	errBadIdentity = errors.New("data datagram carries an invalid packet identity")
	errOversize    = errors.New("payload exceeds maximum size")
)

// encodeData frames a payload under the given identity. The same bytes are sent
// over every path; the peer dedups by identity.
func encodeData(id protocol.PacketID, payload []byte) ([]byte, error) {
	if !id.Valid() {
		return nil, errBadIdentity
	}
	if len(payload) > maxPayload {
		return nil, errOversize
	}
	out := make([]byte, dataHeaderSize+len(payload))
	copy(out[0:4], dataMagic[:])
	out[4] = dataVersion
	out[5] = 0 // flags reserved
	copy(out[6:6+protocol.SessionIDSize], id.Session[:])
	binary.BigEndian.PutUint64(out[6+protocol.SessionIDSize:], uint64(id.Number))
	copy(out[dataHeaderSize:], payload)
	return out, nil
}

// decodeData parses a data datagram, returning its payload and identity. Trailing
// interpretation is the caller's; the payload slice aliases the input buffer.
func decodeData(b []byte) ([]byte, protocol.PacketID, error) {
	if len(b) < dataHeaderSize {
		return nil, protocol.PacketID{}, errShortData
	}
	if [4]byte(b[0:4]) != dataMagic {
		return nil, protocol.PacketID{}, errBadMagic
	}
	if b[4] != dataVersion {
		return nil, protocol.PacketID{}, errBadVersion
	}

	var session protocol.SessionID
	copy(session[:], b[6:6+protocol.SessionIDSize])
	number := protocol.PacketNumber(binary.BigEndian.Uint64(b[6+protocol.SessionIDSize:]))
	id := protocol.PacketID{Session: session, Number: number}
	if !id.Valid() {
		return nil, protocol.PacketID{}, errBadIdentity
	}
	return b[dataHeaderSize:], id, nil
}
