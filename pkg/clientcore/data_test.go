package clientcore

import (
	"testing"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

func TestSessionIDFromDatagram(t *testing.T) {
	id := sessionID(0x5C)
	client, err := NewSession(id, testKey(), RoleInitiator, 16)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	wire, err := client.Outbound([]byte("payload"))
	if err != nil {
		t.Fatalf("outbound: %v", err)
	}

	got, err := SessionIDFromDatagram(wire)
	if err != nil {
		t.Fatalf("SessionIDFromDatagram: %v", err)
	}
	if got != id {
		t.Fatalf("session id = %s, want %s", got, id)
	}
}

func TestSessionIDFromDatagramRejectsJunk(t *testing.T) {
	if _, err := SessionIDFromDatagram([]byte("short")); err == nil {
		t.Fatal("expected error for short datagram")
	}
	bad := make([]byte, dataHeaderSize)
	copy(bad, []byte("NOPE"))
	if _, err := SessionIDFromDatagram(bad); err == nil {
		t.Fatal("expected error for bad magic")
	}
	// A zero session id is invalid.
	zero := make([]byte, dataHeaderSize)
	copy(zero[0:4], dataMagic[:])
	zero[4] = dataVersion
	if _, err := SessionIDFromDatagram(zero); err == nil {
		t.Fatal("expected error for zero session id")
	}
	_ = protocol.SessionID{}
}
