package protocol

import (
	"encoding/hex"
	"fmt"
)

const SessionIDSize = 16

type SessionID [SessionIDSize]byte

type PacketNumber uint64

type PacketID struct {
	Session SessionID
	Number  PacketNumber
}

func NewSessionID(raw []byte) (SessionID, error) {
	var id SessionID
	if len(raw) != SessionIDSize {
		return id, fmt.Errorf("session id must be %d bytes, got %d", SessionIDSize, len(raw))
	}
	copy(id[:], raw)
	return id, nil
}

func (id SessionID) IsZero() bool {
	for _, b := range id {
		if b != 0 {
			return false
		}
	}
	return true
}

func (id SessionID) String() string {
	return hex.EncodeToString(id[:])
}

func (id PacketID) Valid() bool {
	return !id.Session.IsZero() && id.Number > 0
}

func (id PacketID) String() string {
	if !id.Valid() {
		return "invalid-packet-id"
	}
	return fmt.Sprintf("%s:%d", id.Session, id.Number)
}
