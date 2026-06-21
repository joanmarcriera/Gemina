package protocol

import (
	"encoding/binary"
	"errors"
)

// Stage-1 probe wire format. A fixed-size datagram carrying a packet identity
// and the coarse path tag the client sent it over, so the gateway can dedup by
// identity and attribute copies to a path without inspecting (or logging) the
// source address. The format deliberately carries no host identifiers.
//
//	offset 0  magic   "CVP1" (4 bytes)
//	offset 4  version 1 byte (ProbeVersion)
//	offset 5  path    1 byte (PathTag)
//	offset 6  session 16 bytes (SessionID)
//	offset 22 number  8 bytes, big-endian (PacketNumber)
//	total     30 bytes (ProbeWireSize)
const (
	ProbeVersion       = 1
	ProbeSessionOffset = 6
	ProbeWireSize      = ProbeSessionOffset + SessionIDSize + 8
)

var probeMagic = [4]byte{'C', 'V', 'P', '1'}

var (
	ErrShortProbe         = errors.New("probe datagram shorter than wire size")
	ErrBadMagic           = errors.New("probe datagram has wrong magic")
	ErrUnsupportedVersion = errors.New("probe datagram has unsupported version")
	ErrInvalidProbe       = errors.New("probe carries an invalid packet identity")
)

// PathTag is the coarse, non-identifying uplink label a probe was sent over.
type PathTag uint8

const (
	PathUnknown PathTag = iota
	PathWiFi
	PathAndroidUSBTether
)

func (tag PathTag) String() string {
	switch tag {
	case PathWiFi:
		return "wi-fi"
	case PathAndroidUSBTether:
		return "android-usb-tether"
	default:
		return "unknown"
	}
}

// ProbePacket is one Stage-1 probe: a packet identity plus the path tag it was
// sent over.
type ProbePacket struct {
	ID   PacketID
	Path PathTag
}

// MarshalBinary encodes the probe to its fixed wire form. It refuses to encode
// an invalid packet identity so malformed probes never enter the transport.
func (p ProbePacket) MarshalBinary() ([]byte, error) {
	if !p.ID.Valid() {
		return nil, ErrInvalidProbe
	}
	out := make([]byte, ProbeWireSize)
	copy(out[0:4], probeMagic[:])
	out[4] = ProbeVersion
	out[5] = byte(p.Path)
	copy(out[ProbeSessionOffset:ProbeSessionOffset+SessionIDSize], p.ID.Session[:])
	binary.BigEndian.PutUint64(out[ProbeSessionOffset+SessionIDSize:], uint64(p.ID.Number))
	return out, nil
}

// UnmarshalProbe decodes a probe from a datagram. Trailing bytes past the fixed
// wire size are ignored so future fields cannot break old receivers.
func UnmarshalProbe(b []byte) (ProbePacket, error) {
	if len(b) < ProbeWireSize {
		return ProbePacket{}, ErrShortProbe
	}
	if [4]byte(b[0:4]) != probeMagic {
		return ProbePacket{}, ErrBadMagic
	}
	if b[4] != ProbeVersion {
		return ProbePacket{}, ErrUnsupportedVersion
	}

	var session SessionID
	copy(session[:], b[ProbeSessionOffset:ProbeSessionOffset+SessionIDSize])
	number := PacketNumber(binary.BigEndian.Uint64(b[ProbeSessionOffset+SessionIDSize:]))

	packet := ProbePacket{ID: PacketID{Session: session, Number: number}, Path: PathTag(b[5])}
	if !packet.ID.Valid() {
		return ProbePacket{}, ErrInvalidProbe
	}
	return packet, nil
}
