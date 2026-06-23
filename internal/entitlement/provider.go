package entitlement

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// CheckoutSession is the result of starting a purchase. A real provider returns
// a hosted checkout URL the client opens to pay; the ID correlates the later
// webhook back to this session.
type CheckoutSession struct {
	ID      string
	Subject string
	Tier    Tier
	URL     string
}

// PaymentEvent is the normalised outcome of a provider webhook, stripped of
// provider-specific shape. Paid is true when the subscription/payment is active
// and the subject should be granted an entitlement.
type PaymentEvent struct {
	Subject string
	Tier    Tier
	Paid    bool
}

// PaymentProvider abstracts a real payment backend (Stripe, App Store / Play
// IAP, …) without binding to one. Implementations must perform their own
// signature verification in VerifyWebhook so the service never trusts raw
// webhook bytes.
type PaymentProvider interface {
	// CreateCheckout starts a purchase for an opaque subject and a tier.
	CreateCheckout(subject string, tier Tier) (CheckoutSession, error)
	// VerifyWebhook authenticates a webhook payload against its signature and
	// returns the normalised event. It must reject tampered or unsigned
	// payloads.
	VerifyWebhook(payload []byte, signature string) (PaymentEvent, error)
}

// ProviderError is a typed error from a PaymentProvider, letting callers
// distinguish provider-level failures from token errors via errors.As.
type ProviderError struct {
	Op  string
	Err error
}

func (e *ProviderError) Error() string { return "entitlement: provider " + e.Op + ": " + e.Err.Error() }
func (e *ProviderError) Unwrap() error { return e.Err }

func providerErr(op string, err error) error { return &ProviderError{Op: op, Err: err} }

// FakeProvider is a deterministic, in-memory PaymentProvider for tests and
// local development. It performs no network calls and holds no real secrets:
// the webhook secret is supplied by the caller and used only to HMAC the
// in-memory payloads it generates, mirroring how a real provider signs
// webhooks.
type FakeProvider struct {
	secret   []byte
	sessions map[string]CheckoutSession
}

// NewFakeProvider returns a FakeProvider whose webhooks are signed with secret.
func NewFakeProvider(secret string) *FakeProvider {
	return &FakeProvider{
		secret:   []byte(secret),
		sessions: make(map[string]CheckoutSession),
	}
}

// CreateCheckout records and returns a deterministic checkout session. The
// session ID is derived from subject and tier so repeated calls (and separate
// provider instances) yield the same ID, which keeps tests stable.
func (p *FakeProvider) CreateCheckout(subject string, tier Tier) (CheckoutSession, error) {
	if subject == "" {
		return CheckoutSession{}, providerErr("checkout", errors.New("empty subject"))
	}
	if !validTier(tier) {
		return CheckoutSession{}, providerErr("checkout", fmt.Errorf("unknown tier %q", tier))
	}
	id := "cs_" + hex.EncodeToString(digest(p.secret, subject+"|"+string(tier)))[:24]
	sess := CheckoutSession{
		ID:      id,
		Subject: subject,
		Tier:    tier,
		URL:     "https://checkout.example.test/" + id,
	}
	p.sessions[id] = sess
	return sess, nil
}

// SignedWebhook returns a payload and matching signature for a previously
// created session, simulating the provider posting a "payment succeeded"
// webhook. It is a test/dev helper, not part of PaymentProvider.
func (p *FakeProvider) SignedWebhook(sessionID string) (payload []byte, signature string) {
	sess, ok := p.sessions[sessionID]
	if !ok {
		// Produce a payload that will fail VerifyWebhook rather than panic.
		payload = []byte("unknown:" + sessionID)
		return payload, hex.EncodeToString(digest(p.secret, string(payload)))
	}
	payload = []byte("paid|" + sess.Subject + "|" + string(sess.Tier))
	signature = hex.EncodeToString(digest(p.secret, string(payload)))
	return payload, signature
}

// VerifyWebhook authenticates payload against signature using the provider
// secret and parses the normalised event.
func (p *FakeProvider) VerifyWebhook(payload []byte, signature string) (PaymentEvent, error) {
	want := digest(p.secret, string(payload))
	got, err := hex.DecodeString(signature)
	if err != nil {
		return PaymentEvent{}, providerErr("webhook", errors.New("malformed signature"))
	}
	if !hmac.Equal(got, want) {
		return PaymentEvent{}, providerErr("webhook", errors.New("signature mismatch"))
	}

	fields := strings.SplitN(string(payload), "|", 3)
	if len(fields) != 3 || fields[0] != "paid" {
		return PaymentEvent{}, providerErr("webhook", errors.New("unrecognised event"))
	}
	tier := Tier(fields[2])
	if !validTier(tier) {
		return PaymentEvent{}, providerErr("webhook", fmt.Errorf("unknown tier %q", tier))
	}
	return PaymentEvent{Subject: fields[1], Tier: tier, Paid: true}, nil
}

// digest returns HMAC-SHA256(msg) under key.
func digest(key []byte, msg string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msg))
	return mac.Sum(nil)
}
