package clientcore

import (
	"encoding/binary"
	"errors"
)

// A tiny unauthenticated ping/pong used only for latency and loss measurement
// (the `continuityctl benchmark` tool). It carries no identity and no payload —
// just a nonce the gateway echoes back. The pong is the same size as the ping,
// so it cannot be used for traffic amplification.
//
//	offset 0  magic "CVPP" (4 bytes)
//	offset 4  type  1 byte (1 = ping, 2 = pong)
//	offset 5  nonce 8 bytes, big-endian
const pingSize = 13

var pingMagic = [4]byte{'C', 'V', 'P', 'P'}

const (
	pingTypePing = 1
	pingTypePong = 2
)

var errNotPing = errors.New("not a ping/pong datagram")

func encodePing(typ byte, nonce uint64) []byte {
	out := make([]byte, pingSize)
	copy(out[0:4], pingMagic[:])
	out[4] = typ
	binary.BigEndian.PutUint64(out[5:], nonce)
	return out
}

// EncodePing frames a ping carrying nonce.
func EncodePing(nonce uint64) []byte { return encodePing(pingTypePing, nonce) }

// EncodePong frames the reply to a ping with the same nonce.
func EncodePong(nonce uint64) []byte { return encodePing(pingTypePong, nonce) }

// DecodePing parses a ping or pong, returning whether it is a pong and its nonce.
func DecodePing(b []byte) (isPong bool, nonce uint64, err error) {
	if len(b) != pingSize || [4]byte{b[0], b[1], b[2], b[3]} != pingMagic {
		return false, 0, errNotPing
	}
	switch b[4] {
	case pingTypePing:
		isPong = false
	case pingTypePong:
		isPong = true
	default:
		return false, 0, errNotPing
	}
	return isPong, binary.BigEndian.Uint64(b[5:]), nil
}
