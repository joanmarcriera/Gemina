package clientcore

import "testing"

func TestClassifyDatagram(t *testing.T) {
	_, eph, _ := GenerateKeyPair()

	clientHello, _ := EncodeClientHello(sessionID(1), 0, eph, "token")
	if got := ClassifyDatagram(clientHello); got != KindClientHello {
		t.Errorf("client hello classified as %v", got)
	}

	serverHello, _ := EncodeServerHello(sessionID(1), eph, make([]byte, 64))
	if got := ClassifyDatagram(serverHello); got != KindServerHello {
		t.Errorf("server hello classified as %v", got)
	}

	s, _ := NewSession(sessionID(1), testKey(), RoleInitiator, 16)
	data, _ := s.Outbound([]byte("payload"))
	if got := ClassifyDatagram(data); got != KindData {
		t.Errorf("data classified as %v", got)
	}

	for _, junk := range [][]byte{nil, []byte("zz"), []byte("NOPE-not-a-real-frame")} {
		if got := ClassifyDatagram(junk); got != KindUnknown {
			t.Errorf("junk %q classified as %v", junk, got)
		}
	}
}
