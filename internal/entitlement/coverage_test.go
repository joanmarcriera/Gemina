package entitlement

import (
	"errors"
	"io"
	"testing"
	"time"
)

// TestWithWebhookTolerance pins that the option widens or narrows the webhook
// replay window and that a non-positive value is ignored (keeping the 5-minute
// default) rather than disabling or inverting the check.
func TestWithWebhookTolerance(t *testing.T) {
	const whsec = "whsec_tol"
	now := time.Unix(1_800_000_000, 0)
	payload := []byte(`{"type":"checkout.session.completed","data":{"object":{"client_reference_id":"acct_1","metadata":{"tier":"hosted"}}}}`)

	tests := []struct {
		name    string
		opts    []StripeOption
		age     time.Duration // how old the signature timestamp is
		wantErr bool
	}{
		{name: "default accepts 4m", age: 4 * time.Minute},
		{name: "default rejects 6m", age: 6 * time.Minute, wantErr: true},
		{name: "widened accepts 9m", opts: []StripeOption{WithWebhookTolerance(10 * time.Minute)}, age: 9 * time.Minute},
		{name: "narrowed rejects 31s", opts: []StripeOption{WithWebhookTolerance(30 * time.Second)}, age: 31 * time.Second, wantErr: true},
		{name: "narrowed accepts future 29s", opts: []StripeOption{WithWebhookTolerance(30 * time.Second)}, age: -29 * time.Second},
		{name: "narrowed rejects future 31s", opts: []StripeOption{WithWebhookTolerance(30 * time.Second)}, age: -31 * time.Second, wantErr: true},
		{name: "zero ignored keeps default", opts: []StripeOption{WithWebhookTolerance(0)}, age: 4 * time.Minute},
		{name: "zero ignored still rejects 6m", opts: []StripeOption{WithWebhookTolerance(0)}, age: 6 * time.Minute, wantErr: true},
		{name: "negative ignored keeps default", opts: []StripeOption{WithWebhookTolerance(-time.Hour)}, age: 6 * time.Minute, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := append([]StripeOption{WithClock(fixedClock{t: now})}, tc.opts...)
			p := NewStripeProvider("sk", whsec, opts...)
			sig := stripeSignature(whsec, now.Add(-tc.age), payload)
			_, err := p.VerifyWebhook(payload, sig)
			if (err != nil) != tc.wantErr {
				t.Fatalf("VerifyWebhook err = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				var pe *ProviderError
				if !errors.As(err, &pe) || pe.Op != "webhook" {
					t.Fatalf("err = %#v, want *ProviderError{Op: webhook}", err)
				}
			}
		})
	}
}

// TestProviderErrorFormatAndUnwrap pins the message shape and that the cause
// stays reachable through errors.Is / errors.As.
func TestProviderErrorFormatAndUnwrap(t *testing.T) {
	cause := io.ErrUnexpectedEOF
	err := providerErr("checkout", cause)

	if got, want := err.Error(), "entitlement: provider checkout: "+cause.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is(err, cause) = false; Unwrap broken")
	}
	var pe *ProviderError
	if !errors.As(err, &pe) || pe.Op != "checkout" || pe.Unwrap() != cause {
		t.Fatalf("errors.As gave %#v", pe)
	}
	if errors.Is(err, ErrBadSignature) {
		t.Fatal("provider error must not match token errors")
	}
}

func TestSystemClockNow(t *testing.T) {
	before := time.Now()
	got := SystemClock{}.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("SystemClock.Now() = %v, not within [%v, %v]", got, before, after)
	}
	var _ Clock = SystemClock{}
}
