package protocol

import (
	"strings"
	"testing"
)

func TestNewSessionIDRequiresExactSize(t *testing.T) {
	if _, err := NewSessionID(make([]byte, SessionIDSize-1)); err == nil {
		t.Fatal("NewSessionID accepted a short identifier")
	}

	if _, err := NewSessionID(make([]byte, SessionIDSize+1)); err == nil {
		t.Fatal("NewSessionID accepted a long identifier")
	}

	raw := []byte("1234567890abcdef")
	got, err := NewSessionID(raw)
	if err != nil {
		t.Fatalf("NewSessionID returned error: %v", err)
	}
	if got.String() != "31323334353637383930616263646566" {
		t.Fatalf("SessionID.String() = %q", got.String())
	}
}

func TestPacketIDValidity(t *testing.T) {
	session := filledSessionID(0x7a)

	tests := []struct {
		name string
		id   PacketID
		want bool
	}{
		{name: "valid", id: PacketID{Session: session, Number: 1}, want: true},
		{name: "zero session", id: PacketID{Number: 1}, want: false},
		{name: "zero number", id: PacketID{Session: session}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Valid(); got != tt.want {
				t.Fatalf("PacketID.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidPacketIDStringIsExplicit(t *testing.T) {
	if got := (PacketID{}).String(); got != "invalid-packet-id" {
		t.Fatalf("PacketID.String() = %q", got)
	}

	if got := (PacketID{Session: filledSessionID(0x01), Number: 42}).String(); !strings.HasSuffix(got, ":42") {
		t.Fatalf("PacketID.String() = %q, want packet number suffix", got)
	}
}

func filledSessionID(value byte) SessionID {
	var id SessionID
	for i := range id {
		id[i] = value
	}
	return id
}
