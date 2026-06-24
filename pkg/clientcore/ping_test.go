package clientcore

import "testing"

func TestPingRoundTrip(t *testing.T) {
	ping := EncodePing(0x0123456789ABCDEF)
	if got := ClassifyDatagram(ping); got != KindPing {
		t.Fatalf("ping classified as %v", got)
	}
	isPong, nonce, err := DecodePing(ping)
	if err != nil {
		t.Fatalf("decode ping: %v", err)
	}
	if isPong || nonce != 0x0123456789ABCDEF {
		t.Fatalf("ping decode = pong:%v nonce:%x", isPong, nonce)
	}

	pong := EncodePong(nonce)
	isPong, gotNonce, err := DecodePing(pong)
	if err != nil {
		t.Fatalf("decode pong: %v", err)
	}
	if !isPong || gotNonce != nonce {
		t.Fatalf("pong decode = pong:%v nonce:%x", isPong, gotNonce)
	}
}

func TestDecodePingRejectsJunk(t *testing.T) {
	if _, _, err := DecodePing([]byte("nope")); err == nil {
		t.Fatal("accepted junk as ping")
	}
}
