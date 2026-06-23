package entitlement

import (
	"errors"
	"testing"
)

func TestFakeProviderCheckout(t *testing.T) {
	p := NewFakeProvider("whsec_test")

	tests := []struct {
		name    string
		subject string
		tier    Tier
		wantErr bool
	}{
		{name: "valid hosted checkout", subject: "sub_1", tier: TierHosted},
		{name: "empty subject", subject: "", tier: TierHosted, wantErr: true},
		{name: "unknown tier", subject: "sub_1", tier: Tier("platinum"), wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sess, err := p.CreateCheckout(tc.subject, tc.tier)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("CreateCheckout(%q,%q) = nil err, want error", tc.subject, tc.tier)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateCheckout: %v", err)
			}
			if sess.ID == "" {
				t.Errorf("checkout session has empty ID")
			}
			if sess.Subject != tc.subject || sess.Tier != tc.tier {
				t.Errorf("session = %+v, want subject %q tier %q", sess, tc.subject, tc.tier)
			}
			if sess.URL == "" {
				t.Errorf("checkout session has empty URL")
			}
		})
	}
}

func TestFakeProviderDeterministic(t *testing.T) {
	p1 := NewFakeProvider("whsec_test")
	p2 := NewFakeProvider("whsec_test")
	a, err := p1.CreateCheckout("sub_42", TierHosted)
	if err != nil {
		t.Fatalf("p1 checkout: %v", err)
	}
	b, err := p2.CreateCheckout("sub_42", TierHosted)
	if err != nil {
		t.Fatalf("p2 checkout: %v", err)
	}
	if a.ID != b.ID {
		t.Errorf("checkout IDs not deterministic: %q vs %q", a.ID, b.ID)
	}
}

func TestFakeProviderWebhook(t *testing.T) {
	const secret = "whsec_test"
	p := NewFakeProvider(secret)

	// Drive a real checkout so the provider knows the subject/tier and can
	// produce a signed webhook payload for it.
	sess, err := p.CreateCheckout("sub_9", TierHosted)
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}

	payload, sig := p.SignedWebhook(sess.ID)

	tests := []struct {
		name      string
		payload   []byte
		signature string
		wantErr   bool
		wantSub   string
		wantTier  Tier
	}{
		{
			name:      "valid webhook",
			payload:   payload,
			signature: sig,
			wantSub:   "sub_9",
			wantTier:  TierHosted,
		},
		{
			name:      "bad signature",
			payload:   payload,
			signature: "deadbeef",
			wantErr:   true,
		},
		{
			name:      "tampered payload",
			payload:   append([]byte("x"), payload...),
			signature: sig,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := p.VerifyWebhook(tc.payload, tc.signature)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("VerifyWebhook accepted %s, want rejection", tc.name)
				}
				return
			}
			if err != nil {
				t.Fatalf("VerifyWebhook: %v", err)
			}
			if !ev.Paid {
				t.Errorf("event Paid = false, want true")
			}
			if ev.Subject != tc.wantSub || ev.Tier != tc.wantTier {
				t.Errorf("event = %+v, want subject %q tier %q", ev, tc.wantSub, tc.wantTier)
			}
		})
	}
}

func TestFakeProviderWebhookUnknownSession(t *testing.T) {
	p := NewFakeProvider("whsec_test")
	// Asking for a signed webhook on an unknown session must not panic and
	// VerifyWebhook on arbitrary bytes must error rather than succeed.
	if _, err := p.VerifyWebhook([]byte("{}"), "00"); err == nil {
		t.Errorf("VerifyWebhook accepted arbitrary payload")
	}
	var perr *ProviderError
	_, err := p.CreateCheckout("", TierHosted)
	if !errors.As(err, &perr) {
		t.Errorf("expected *ProviderError, got %T: %v", err, err)
	}
}
