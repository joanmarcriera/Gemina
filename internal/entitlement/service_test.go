package entitlement

import (
	"testing"
	"time"
)

func newTestService(t *testing.T, mode Mode, now time.Time) (*Service, *FakeProvider) {
	t.Helper()
	p := NewFakeProvider("whsec_test")
	svc := &Service{
		Mode:     mode,
		Key:      []byte("server-secret-key-for-tests"),
		Provider: p,
		TokenTTL: 30 * 24 * time.Hour,
		Clock:    fixedClock{now},
	}
	return svc, p
}

func TestServiceIssueThenAdmit(t *testing.T) {
	now := mustTime(t, "2026-01-01T00:00:00Z")
	svc, p := newTestService(t, ModeHosted, now)

	sess, err := p.CreateCheckout("sub_77", TierHosted)
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}
	payload, sig := p.SignedWebhook(sess.ID)

	// A verified payment event yields an entitlement token.
	tok, err := svc.OnPaymentEvent(payload, sig)
	if err != nil {
		t.Fatalf("OnPaymentEvent: %v", err)
	}
	if tok == "" {
		t.Fatalf("OnPaymentEvent returned empty token")
	}

	// The hosted gateway admits a client presenting that token.
	claims, err := svc.Admit(tok)
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if claims.Subject != "sub_77" || claims.Tier != TierHosted {
		t.Errorf("admitted claims = %+v, want subject sub_77 tier hosted", claims)
	}
}

func TestServiceHostedModeRejects(t *testing.T) {
	now := mustTime(t, "2026-01-01T00:00:00Z")
	svc, _ := newTestService(t, ModeHosted, now)

	tests := []struct {
		name  string
		token string
	}{
		{name: "missing token", token: ""},
		{name: "garbage token", token: "not.a.token"},
		{name: "wrong-key token", token: mustIssueWith(t, []byte("attacker-key-attacker-key--"), now)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Admit(tc.token); err == nil {
				t.Errorf("Admit accepted %s in hosted mode, want rejection", tc.name)
			}
		})
	}
}

func TestServiceHostedModeRejectsExpired(t *testing.T) {
	issued := mustTime(t, "2026-01-01T00:00:00Z")
	svc, _ := newTestService(t, ModeHosted, issued)

	claims := Claims{Subject: "sub_x", Tier: TierHosted, Expiry: issued.Add(time.Hour).Unix()}
	tok, err := Issue(claims, svc.Key)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// Advance the service clock beyond expiry.
	svc.Clock = fixedClock{issued.Add(2 * time.Hour)}
	if _, err := svc.Admit(tok); err == nil {
		t.Errorf("Admit accepted expired token in hosted mode")
	}
}

func TestServiceOpenModeAdmitsWithoutToken(t *testing.T) {
	now := mustTime(t, "2026-01-01T00:00:00Z")
	svc, _ := newTestService(t, ModeOpen, now)

	// Self-hosted: no token required, Admit always succeeds.
	claims, err := svc.Admit("")
	if err != nil {
		t.Fatalf("Admit in open mode: %v", err)
	}
	if claims.Tier != TierSelfHosted {
		t.Errorf("open-mode claims tier = %q, want self-hosted", claims.Tier)
	}

	// Even a garbage token is fine in open mode; the gate is disabled.
	if _, err := svc.Admit("garbage"); err != nil {
		t.Errorf("Admit garbage token in open mode: %v", err)
	}
}

func mustIssueWith(t *testing.T, key []byte, now time.Time) string {
	t.Helper()
	claims := Claims{Subject: "sub", Tier: TierHosted, Expiry: now.Add(time.Hour).Unix()}
	tok, err := Issue(claims, key)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	return tok
}
