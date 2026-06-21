package transport

import (
	"errors"
	"testing"
)

func TestDialUDPRequiresInterface(t *testing.T) {
	_, err := PathDialer{}.DialUDP("127.0.0.1:9")
	if !errors.Is(err, ErrNoInterface) {
		t.Fatalf("DialUDP with empty interface error = %v, want ErrNoInterface", err)
	}
}

func TestDialUDPRejectsUnknownInterface(t *testing.T) {
	_, err := PathDialer{Interface: "definitelynotreal0"}.DialUDP("127.0.0.1:9")
	if err == nil {
		t.Fatal("DialUDP with unknown interface: expected error, got nil")
	}
	if errors.Is(err, ErrNoInterface) {
		t.Fatalf("unknown interface should not report ErrNoInterface, got %v", err)
	}
}
