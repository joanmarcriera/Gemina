package clientcore

import "testing"

func TestGatewaySignatureVerifies(t *testing.T) {
	idPriv, idPub, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	_, ephPub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("ephemeral: %v", err)
	}
	id := sessionID(0x77)

	sig := SignHandshake(idPriv, ephPub, id)
	if err := VerifyHandshake(idPub, ephPub, sig, id); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
}

func TestVerifyRejectsTamperedEphemeralKey(t *testing.T) {
	idPriv, idPub, _ := GenerateIdentity()
	_, ephPub, _ := GenerateKeyPair()
	id := sessionID(0x77)
	sig := SignHandshake(idPriv, ephPub, id)

	tampered := append([]byte(nil), ephPub...)
	tampered[0] ^= 0x01
	if err := VerifyHandshake(idPub, tampered, sig, id); err == nil {
		t.Fatal("accepted a tampered ephemeral key")
	}
}

func TestVerifyRejectsWrongIdentity(t *testing.T) {
	idPriv, _, _ := GenerateIdentity()
	_, otherPub, _ := GenerateIdentity()
	_, ephPub, _ := GenerateKeyPair()
	id := sessionID(0x77)
	sig := SignHandshake(idPriv, ephPub, id)

	if err := VerifyHandshake(otherPub, ephPub, sig, id); err == nil {
		t.Fatal("accepted a signature from a different identity")
	}
}

func TestVerifyRejectsWrongSession(t *testing.T) {
	idPriv, idPub, _ := GenerateIdentity()
	_, ephPub, _ := GenerateKeyPair()
	sig := SignHandshake(idPriv, ephPub, sessionID(0x01))

	if err := VerifyHandshake(idPub, ephPub, sig, sessionID(0x02)); err == nil {
		t.Fatal("accepted a signature bound to a different session")
	}
}

// The core MITM defence: an attacker who substitutes its own ephemeral key (and
// can only sign with its OWN identity) is rejected by a client that pins the
// real gateway identity.
func TestMITMSubstitutedKeyRejected(t *testing.T) {
	_, realGatewayPub, _ := GenerateIdentity() // client pins this
	attackerPriv, _, _ := GenerateIdentity()
	_, attackerEph, _ := GenerateKeyPair()
	id := sessionID(0x99)

	attackerSig := SignHandshake(attackerPriv, attackerEph, id)
	if err := VerifyHandshake(realGatewayPub, attackerEph, attackerSig, id); err == nil {
		t.Fatal("MITM substituted key accepted — pinning defeated")
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	idPriv, idPub, _ := GenerateIdentity()
	_, ephPub, _ := GenerateKeyPair()
	id := sessionID(0x77)
	sig := SignHandshake(idPriv, ephPub, id)

	if err := VerifyHandshake([]byte("short"), ephPub, sig, id); err == nil {
		t.Fatal("accepted a malformed identity public key")
	}
	if err := VerifyHandshake(idPub, ephPub, []byte("short"), id); err == nil {
		t.Fatal("accepted a malformed signature")
	}
}
