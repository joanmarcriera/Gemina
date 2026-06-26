package entitlement

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
