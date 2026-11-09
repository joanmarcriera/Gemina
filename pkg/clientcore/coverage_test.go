package clientcore

import (
	"errors"
	"testing"
	"time"

	"github.com/joanmarcriera/gemina/internal/protocol"
)

// TestCheckHandshakeFresh pins the replay-window boundary: a ClientHello is
// accepted up to and including tolerance of skew in either direction, and
// rejected with ErrStaleHandshake one second beyond it.
func TestCheckHandshakeFresh(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tol := DefaultHandshakeTolerance
	tests := []struct {
		name    string
		ts      int64
		tol     time.Duration
		wantErr error
	}{
		{name: "exact now", ts: now.Unix(), tol: tol},
		{name: "past at boundary", ts: now.Add(-tol).Unix(), tol: tol},
		{name: "future at boundary", ts: now.Add(tol).Unix(), tol: tol},
		{name: "past beyond boundary", ts: now.Add(-tol - time.Second).Unix(), tol: tol, wantErr: ErrStaleHandshake},
		{name: "future beyond boundary", ts: now.Add(tol + time.Second).Unix(), tol: tol, wantErr: ErrStaleHandshake},
		{name: "zero timestamp", ts: 0, tol: tol, wantErr: ErrStaleHandshake},
		{name: "negative timestamp", ts: -1, tol: tol, wantErr: ErrStaleHandshake},
		{name: "zero tolerance exact", ts: now.Unix(), tol: 0},
		{name: "zero tolerance one second off", ts: now.Unix() - 1, tol: 0, wantErr: ErrStaleHandshake},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckHandshakeFresh(tc.ts, now, tc.tol)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("CheckHandshakeFresh(%d) = %v, want %v", tc.ts, err, tc.wantErr)
			}
		})
	}
}

func TestDatagramKindString(t *testing.T) {
	tests := []struct {
		kind DatagramKind
		want string
	}{
		{KindUnknown, "unknown"},
		{KindData, "data"},
		{KindClientHello, "client-hello"},
		{KindServerHello, "server-hello"},
		{KindPing, "ping"},
		{DatagramKind(99), "unknown"},
		{DatagramKind(-1), "unknown"},
	}
	for _, tc := range tests {
		if got := tc.kind.String(); got != tc.want {
			t.Errorf("DatagramKind(%d).String() = %q, want %q", int(tc.kind), got, tc.want)
		}
	}
}

// TestAssignedIPv4FromServerHello pins that the gateway-leased tunnel address
// carried in the ServerHello is surfaced on the completed client session, and
// that the zero value ("unassigned") round-trips as zero. A responder session
// built directly has no lease.
func TestAssignedIPv4FromServerHello(t *testing.T) {
	idPriv, idPub, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}
	for _, want := range [][4]byte{{10, 77, 0, 2}, {}} {
		hello, hs, err := BeginClientHandshake(idPub, "tok")
		if err != nil {
			t.Fatalf("BeginClientHandshake: %v", err)
		}
		id, _, _, _, err := DecodeClientHello(hello)
		if err != nil {
			t.Fatalf("DecodeClientHello: %v", err)
		}
		_, gwPub, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair: %v", err)
		}
		sh, err := EncodeServerHello(id, gwPub, SignHandshake(idPriv, gwPub, id), want)
		if err != nil {
			t.Fatalf("EncodeServerHello: %v", err)
		}
		s, err := hs.Complete(sh, 64)
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if got := s.AssignedIPv4(); got != want {
			t.Fatalf("AssignedIPv4 = %v, want %v", got, want)
		}
	}

	var id protocol.SessionID
	id[0] = 1
	resp, err := NewSession(id, make([]byte, 32), RoleResponder, 64)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if got := resp.AssignedIPv4(); got != [4]byte{} {
		t.Fatalf("responder AssignedIPv4 = %v, want zero", got)
	}
}
