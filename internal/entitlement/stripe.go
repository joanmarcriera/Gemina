package entitlement

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// defaultStripeBaseURL is Stripe's REST API root. It is overridable via
// WithBaseURL so tests can point the provider at an httptest.Server.
const defaultStripeBaseURL = "https://api.stripe.com"

// defaultWebhookTolerance is the maximum allowed difference between the
// signed timestamp and the current time. It mirrors Stripe's own default and
// guards against replay of captured webhooks.
const defaultWebhookTolerance = 5 * time.Minute

// StripeProvider is a PaymentProvider backed by Stripe's REST API, implemented
// with the Go standard library only (no stripe-go SDK). It creates hosted
// checkout sessions and verifies inbound webhook signatures using Stripe's
// documented HMAC-SHA256 scheme.
//
// Secrets (the API key and the webhook signing secret) are held in unexported
// fields and are never logged or embedded in error messages.
type StripeProvider struct {
	apiKey    string
	whSecret  []byte
	baseURL   string
	client    *http.Client
	clock     Clock
	tolerance time.Duration
}

// StripeOption configures a StripeProvider at construction.
type StripeOption func(*StripeProvider)

// WithHTTPClient injects the HTTP client used for outbound Stripe calls,
// chiefly so tests can supply an httptest.Server's client.
func WithHTTPClient(c *http.Client) StripeOption {
	return func(p *StripeProvider) {
		if c != nil {
			p.client = c
		}
	}
}

// WithBaseURL overrides the Stripe API base URL, chiefly for tests. A trailing
// slash is trimmed so path joining stays predictable.
func WithBaseURL(base string) StripeOption {
	return func(p *StripeProvider) {
		if base != "" {
			p.baseURL = strings.TrimRight(base, "/")
		}
	}
}

// WithClock injects the Clock used for webhook timestamp tolerance checks,
// keeping verification deterministic in tests.
func WithClock(c Clock) StripeOption {
	return func(p *StripeProvider) {
		if c != nil {
			p.clock = c
		}
	}
}

// WithWebhookTolerance overrides the webhook timestamp tolerance window.
func WithWebhookTolerance(d time.Duration) StripeOption {
	return func(p *StripeProvider) {
		if d > 0 {
			p.tolerance = d
		}
	}
}

// NewStripeProvider constructs a StripeProvider from the secret API key and the
// webhook signing secret. The HTTP client, base URL, clock and tolerance all
// have sensible production defaults and are overridable via options for tests.
func NewStripeProvider(apiKey, webhookSecret string, opts ...StripeOption) *StripeProvider {
	p := &StripeProvider{
		apiKey:    apiKey,
		whSecret:  []byte(webhookSecret),
		baseURL:   defaultStripeBaseURL,
		client:    http.DefaultClient,
		clock:     SystemClock{},
		tolerance: defaultWebhookTolerance,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// checkoutMode maps a tier to the Stripe Checkout mode. Hosted is a recurring
// subscription; other (future) one-off tiers would use payment mode.
func checkoutMode(tier Tier) string {
	if tier == TierHosted {
		return "subscription"
	}
	return "payment"
}

// CreateCheckout starts a Stripe Checkout session for the subject and tier and
// returns the normalised CheckoutSession (id + hosted URL). Non-2xx responses
// are mapped to *ProviderError.
func (p *StripeProvider) CreateCheckout(subject string, tier Tier) (CheckoutSession, error) {
	if subject == "" {
		return CheckoutSession{}, providerErr("checkout", errors.New("empty subject"))
	}
	if !validTier(tier) {
		return CheckoutSession{}, providerErr("checkout", fmt.Errorf("unknown tier %q", tier))
	}

	form := url.Values{}
	form.Set("mode", checkoutMode(tier))
	form.Set("client_reference_id", subject)
	form.Set("metadata[subject]", subject)
	form.Set("metadata[tier]", string(tier))

	req, err := http.NewRequest(http.MethodPost, p.baseURL+"/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return CheckoutSession{}, providerErr("checkout", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return CheckoutSession{}, providerErr("checkout", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return CheckoutSession{}, providerErr("checkout", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Surface the status only; the body may echo request fields, so it is
		// not included to avoid leaking anything sensitive into logs.
		return CheckoutSession{}, providerErr("checkout", fmt.Errorf("stripe responded %d", resp.StatusCode))
	}

	var out struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return CheckoutSession{}, providerErr("checkout", fmt.Errorf("decode response: %w", err))
	}
	if out.ID == "" {
		return CheckoutSession{}, providerErr("checkout", errors.New("stripe response missing session id"))
	}

	return CheckoutSession{
		ID:      out.ID,
		Subject: subject,
		Tier:    tier,
		URL:     out.URL,
	}, nil
}

// stripeEvent is the minimal shape of a Stripe webhook event we consume.
type stripeEvent struct {
	Type string `json:"type"`
	Data struct {
		Object struct {
			ClientReferenceID string `json:"client_reference_id"`
			Metadata          struct {
				Subject string `json:"subject"`
				Tier    string `json:"tier"`
			} `json:"metadata"`
		} `json:"object"`
	} `json:"data"`
}

// VerifyWebhook authenticates payload against the Stripe-Signature header value
// and returns the normalised PaymentEvent for a completed checkout. It rejects
// bad signatures, stale or future timestamps (outside the tolerance window),
// malformed headers, and unhandled event types.
func (p *StripeProvider) VerifyWebhook(payload []byte, signature string) (PaymentEvent, error) {
	ts, sigs, err := parseStripeSignature(signature)
	if err != nil {
		return PaymentEvent{}, providerErr("webhook", err)
	}

	// Reject timestamps outside the tolerance window (in either direction) to
	// blunt replay of captured webhooks.
	skew := p.clock.Now().Sub(time.Unix(ts, 0))
	if skew < 0 {
		skew = -skew
	}
	if skew > p.tolerance {
		return PaymentEvent{}, providerErr("webhook", errors.New("timestamp outside tolerance"))
	}

	// signedPayload is "<t>.<rawPayload>" per Stripe's scheme.
	mac := hmac.New(sha256.New, p.whSecret)
	fmt.Fprintf(mac, "%d.", ts)
	mac.Write(payload)
	want := mac.Sum(nil)

	if !anyHexEqual(sigs, want) {
		return PaymentEvent{}, providerErr("webhook", errors.New("signature mismatch"))
	}

	var ev stripeEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return PaymentEvent{}, providerErr("webhook", fmt.Errorf("decode event: %w", err))
	}
	if ev.Type != "checkout.session.completed" {
		return PaymentEvent{}, providerErr("webhook", fmt.Errorf("unhandled event type %q", ev.Type))
	}

	obj := ev.Data.Object
	subject := obj.ClientReferenceID
	if subject == "" {
		subject = obj.Metadata.Subject
	}
	if subject == "" {
		return PaymentEvent{}, providerErr("webhook", errors.New("event missing subject"))
	}

	tier := Tier(obj.Metadata.Tier)
	if !validTier(tier) {
		return PaymentEvent{}, providerErr("webhook", fmt.Errorf("unknown tier %q", tier))
	}

	return PaymentEvent{Subject: subject, Tier: tier, Paid: true}, nil
}

// parseStripeSignature parses a Stripe-Signature header value of the form
// "t=<unix>,v1=<hex>[,v1=<hex>...]" and returns the timestamp and the list of
// v1 signatures. Other scheme keys (e.g. v0) are ignored.
func parseStripeSignature(header string) (ts int64, v1s []string, err error) {
	if header == "" {
		return 0, nil, errors.New("empty signature header")
	}
	var haveTS bool
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			ts, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return 0, nil, errors.New("malformed timestamp")
			}
			haveTS = true
		case "v1":
			if v != "" {
				v1s = append(v1s, v)
			}
		}
	}
	if !haveTS {
		return 0, nil, errors.New("missing timestamp")
	}
	if len(v1s) == 0 {
		return 0, nil, errors.New("missing v1 signature")
	}
	return ts, v1s, nil
}

// anyHexEqual reports whether any hex-encoded candidate decodes to want, using
// a constant-time comparison to guard against signature-timing attacks.
func anyHexEqual(candidates []string, want []byte) bool {
	var matched bool
	for _, c := range candidates {
		got, err := hex.DecodeString(c)
		if err != nil {
			continue
		}
		// Do not short-circuit: compare every candidate in constant time.
		if hmac.Equal(got, want) {
			matched = true
		}
	}
	return matched
}
