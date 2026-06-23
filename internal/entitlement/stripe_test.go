package entitlement

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// stripeSignature builds a valid Stripe-Signature header value for payload at
// time t under secret, mirroring Stripe's t=<unix>,v1=<hex hmac-sha256> scheme
// over "<t>.<payload>".
func stripeSignature(secret string, t time.Time, payload []byte) string {
	ts := t.Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.%s", ts, payload)
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func TestStripeCreateCheckout(t *testing.T) {
	const (
		secret  = "sk_test_abc123"
		whsec   = "whsec_test"
		subject = "acct_42"
	)

	var (
		gotMethod string
		gotPath   string
		gotAuth   string
		gotCT     string
		gotForm   url.Values
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotForm, _ = url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"id":"cs_test_123","url":"https://checkout.stripe.test/c/cs_test_123","object":"checkout.session"}`)
	}))
	defer srv.Close()

	p := NewStripeProvider(secret, whsec,
		WithHTTPClient(srv.Client()),
		WithBaseURL(srv.URL),
	)

	sess, err := p.CreateCheckout(subject, TierHosted)
	if err != nil {
		t.Fatalf("CreateCheckout: unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/v1/checkout/sessions" {
		t.Errorf("path = %q, want /v1/checkout/sessions", gotPath)
	}
	if gotAuth != "Bearer "+secret {
		t.Errorf("auth = %q, want Bearer <secret>", gotAuth)
	}
	if !strings.HasPrefix(gotCT, "application/x-www-form-urlencoded") {
		t.Errorf("content-type = %q, want form-urlencoded", gotCT)
	}
	if got := gotForm.Get("client_reference_id"); got != subject {
		t.Errorf("client_reference_id = %q, want %q", got, subject)
	}
	if got := gotForm.Get("metadata[subject]"); got != subject {
		t.Errorf("metadata[subject] = %q, want %q", got, subject)
	}
	if got := gotForm.Get("metadata[tier]"); got != string(TierHosted) {
		t.Errorf("metadata[tier] = %q, want %q", got, TierHosted)
	}
	if got := gotForm.Get("mode"); got == "" {
		t.Errorf("mode missing from form")
	}

	if sess.ID != "cs_test_123" {
		t.Errorf("session ID = %q, want cs_test_123", sess.ID)
	}
	if sess.URL != "https://checkout.stripe.test/c/cs_test_123" {
		t.Errorf("session URL = %q", sess.URL)
	}
	if sess.Subject != subject || sess.Tier != TierHosted {
		t.Errorf("session subject/tier = %q/%q", sess.Subject, sess.Tier)
	}
}

func TestStripeCreateCheckoutErrors(t *testing.T) {
	t.Run("non-2xx maps to ProviderError", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error":{"message":"bad request"}}`)
		}))
		defer srv.Close()

		p := NewStripeProvider("sk", "whsec", WithHTTPClient(srv.Client()), WithBaseURL(srv.URL))
		_, err := p.CreateCheckout("acct_1", TierHosted)
		var pe *ProviderError
		if !errors.As(err, &pe) {
			t.Fatalf("error = %v, want *ProviderError", err)
		}
	})

	t.Run("empty subject", func(t *testing.T) {
		p := NewStripeProvider("sk", "whsec")
		_, err := p.CreateCheckout("", TierHosted)
		var pe *ProviderError
		if !errors.As(err, &pe) {
			t.Fatalf("error = %v, want *ProviderError", err)
		}
	})

	t.Run("invalid tier", func(t *testing.T) {
		p := NewStripeProvider("sk", "whsec")
		_, err := p.CreateCheckout("acct_1", Tier("bogus"))
		var pe *ProviderError
		if !errors.As(err, &pe) {
			t.Fatalf("error = %v, want *ProviderError", err)
		}
	})
}

// completedEvent is a minimal checkout.session.completed payload.
func completedEvent(subject, tier string) []byte {
	return []byte(fmt.Sprintf(
		`{"type":"checkout.session.completed","data":{"object":{"client_reference_id":%q,"metadata":{"subject":%q,"tier":%q}}}}`,
		subject, subject, tier,
	))
}

func TestStripeVerifyWebhook(t *testing.T) {
	const whsec = "whsec_test_secret"
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	newProvider := func() *StripeProvider {
		return NewStripeProvider("sk", whsec, WithClock(fixedClock{now}))
	}

	t.Run("happy path", func(t *testing.T) {
		p := newProvider()
		payload := completedEvent("acct_99", string(TierHosted))
		sig := stripeSignature(whsec, now, payload)
		ev, err := p.VerifyWebhook(payload, sig)
		if err != nil {
			t.Fatalf("VerifyWebhook: %v", err)
		}
		if !ev.Paid {
			t.Errorf("Paid = false, want true")
		}
		if ev.Subject != "acct_99" {
			t.Errorf("Subject = %q, want acct_99", ev.Subject)
		}
		if ev.Tier != TierHosted {
			t.Errorf("Tier = %q, want %q", ev.Tier, TierHosted)
		}
	})

	t.Run("subject falls back to metadata when client_reference_id empty", func(t *testing.T) {
		p := newProvider()
		payload := []byte(fmt.Sprintf(
			`{"type":"checkout.session.completed","data":{"object":{"metadata":{"subject":"acct_meta","tier":%q}}}}`,
			TierHosted))
		sig := stripeSignature(whsec, now, payload)
		ev, err := p.VerifyWebhook(payload, sig)
		if err != nil {
			t.Fatalf("VerifyWebhook: %v", err)
		}
		if ev.Subject != "acct_meta" {
			t.Errorf("Subject = %q, want acct_meta", ev.Subject)
		}
	})

	rejectCases := []struct {
		name      string
		payload   []byte
		signature string
	}{
		{
			name:      "wrong signature",
			payload:   completedEvent("acct_1", string(TierHosted)),
			signature: stripeSignature("whsec_wrong", now, completedEvent("acct_1", string(TierHosted))),
		},
		{
			name:      "tampered payload",
			payload:   completedEvent("attacker", string(TierHosted)),
			signature: stripeSignature(whsec, now, completedEvent("acct_1", string(TierHosted))),
		},
		{
			name:      "stale timestamp outside tolerance",
			payload:   completedEvent("acct_1", string(TierHosted)),
			signature: stripeSignature(whsec, now.Add(-10*time.Minute), completedEvent("acct_1", string(TierHosted))),
		},
		{
			name:      "future timestamp outside tolerance",
			payload:   completedEvent("acct_1", string(TierHosted)),
			signature: stripeSignature(whsec, now.Add(10*time.Minute), completedEvent("acct_1", string(TierHosted))),
		},
		{
			name:      "malformed header no scheme",
			payload:   completedEvent("acct_1", string(TierHosted)),
			signature: "garbage",
		},
		{
			name:      "malformed header missing v1",
			payload:   completedEvent("acct_1", string(TierHosted)),
			signature: "t=" + strconv.FormatInt(now.Unix(), 10),
		},
		{
			name:      "non-completed event type",
			payload:   []byte(`{"type":"customer.subscription.deleted","data":{"object":{}}}`),
			signature: stripeSignature(whsec, now, []byte(`{"type":"customer.subscription.deleted","data":{"object":{}}}`)),
		},
		{
			name:      "completed event with unknown tier",
			payload:   completedEvent("acct_1", "platinum"),
			signature: stripeSignature(whsec, now, completedEvent("acct_1", "platinum")),
		},
	}

	for _, tc := range rejectCases {
		t.Run(tc.name, func(t *testing.T) {
			p := newProvider()
			if _, err := p.VerifyWebhook(tc.payload, tc.signature); err == nil {
				t.Fatalf("VerifyWebhook accepted %s; want error", tc.name)
			} else {
				var pe *ProviderError
				if !errors.As(err, &pe) {
					t.Errorf("error = %v, want *ProviderError", err)
				}
			}
		})
	}

	t.Run("timestamp within tolerance accepted", func(t *testing.T) {
		p := newProvider()
		payload := completedEvent("acct_1", string(TierHosted))
		// 4 minutes is within the default 5-minute tolerance.
		sig := stripeSignature(whsec, now.Add(-4*time.Minute), payload)
		if _, err := p.VerifyWebhook(payload, sig); err != nil {
			t.Fatalf("VerifyWebhook rejected in-tolerance event: %v", err)
		}
	})
}

// TestStripeServiceRoundTrip drives a StripeProvider-produced webhook through
// the real entitlement.Service and asserts a token is minted and admits.
func TestStripeServiceRoundTrip(t *testing.T) {
	const whsec = "whsec_roundtrip"
	now := time.Date(2026, 6, 24, 9, 0, 0, 0, time.UTC)
	key := []byte("signing-key-roundtrip")

	p := NewStripeProvider("sk", whsec, WithClock(fixedClock{now}))

	svc := &Service{
		Mode:     ModeHosted,
		Key:      key,
		Provider: p,
		TokenTTL: time.Hour,
		Clock:    fixedClock{now},
	}

	payload := completedEvent("acct_roundtrip", string(TierHosted))
	sig := stripeSignature(whsec, now, payload)

	tok, err := svc.OnPaymentEvent(payload, sig)
	if err != nil {
		t.Fatalf("OnPaymentEvent: %v", err)
	}
	if tok == "" {
		t.Fatal("OnPaymentEvent returned empty token")
	}

	claims, err := svc.Admit(tok)
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if claims.Subject != "acct_roundtrip" {
		t.Errorf("claims.Subject = %q, want acct_roundtrip", claims.Subject)
	}
	if claims.Tier != TierHosted {
		t.Errorf("claims.Tier = %q, want %q", claims.Tier, TierHosted)
	}
}
