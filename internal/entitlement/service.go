package entitlement

import (
	"errors"
	"fmt"
	"time"
)

// Mode selects whether the entitlement gate is enforced.
//
// The client and gateway are open source and self-hostable for free; revenue
// comes only from an optional paid hosted gateway. Mode is what keeps the
// open-source / self-host path uncrippled: a self-hosted gateway runs with the
// gate disabled and needs no token, while the paid hosted gateway enforces it.
type Mode uint8

const (
	// ModeOpen disables the entitlement gate. Admit always succeeds and no
	// token is required. This is the mode a self-hosted gateway runs in.
	ModeOpen Mode = iota
	// ModeHosted enforces the gate. Admit requires a valid, unexpired token
	// for the hosted tier. This is the mode the paid hosted gateway runs in.
	ModeHosted
)

// ErrNotConfigured is returned when a hosted-mode service is missing its
// signing key or provider.
var ErrNotConfigured = errors.New("entitlement: service not configured for hosted mode")

// Service ties the token codec and payment provider together. The hosted
// gateway calls Admit to decide whether to admit a client; a billing webhook
// handler calls OnPaymentEvent to mint an entitlement token after a verified
// payment.
type Service struct {
	// Mode selects whether the gate is enforced. The zero value is ModeOpen,
	// so a service constructed without configuration safely defaults to the
	// free self-host behaviour.
	Mode Mode
	// Key signs and verifies entitlement tokens. Required in hosted mode.
	Key []byte
	// Provider verifies payment webhooks. Required in hosted mode.
	Provider PaymentProvider
	// TokenTTL is how long an issued entitlement token remains valid.
	TokenTTL time.Duration
	// Clock supplies the current time for issuance and verification. A nil
	// Clock falls back to SystemClock.
	Clock Clock
}

func (s *Service) clock() Clock {
	if s.Clock == nil {
		return SystemClock{}
	}
	return s.Clock
}

// OnPaymentEvent verifies a provider webhook and, if it represents a completed
// payment, issues an entitlement token for the subject. It is only meaningful
// in hosted mode.
func (s *Service) OnPaymentEvent(payload []byte, signature string) (string, error) {
	if s.Provider == nil || len(s.Key) == 0 {
		return "", ErrNotConfigured
	}
	ev, err := s.Provider.VerifyWebhook(payload, signature)
	if err != nil {
		return "", err
	}
	if !ev.Paid {
		return "", fmt.Errorf("%w: event not paid", ErrInvalidClaims)
	}
	claims := Claims{
		Subject: ev.Subject,
		Tier:    ev.Tier,
		Expiry:  s.clock().Now().Add(s.TokenTTL).Unix(),
	}
	return Issue(claims, s.Key)
}

// Admit decides whether a client may use the gateway.
//
// In ModeOpen the gate is disabled: Admit ignores the token and returns
// self-hosted claims so the free self-host path is never blocked. In ModeHosted
// the token's signature, tier and expiry are checked; a missing, tampered,
// wrong-key or expired token is rejected.
func (s *Service) Admit(token string) (Claims, error) {
	if s.Mode == ModeOpen {
		return Claims{Tier: TierSelfHosted}, nil
	}
	if len(s.Key) == 0 {
		return Claims{}, ErrNotConfigured
	}
	if token == "" {
		return Claims{}, fmt.Errorf("%w: token required in hosted mode", ErrMalformedToken)
	}
	claims, err := Verify(token, s.Key, s.clock())
	if err != nil {
		return Claims{}, err
	}
	if claims.Tier != TierHosted {
		return Claims{}, fmt.Errorf("%w: tier %q not admitted", ErrInvalidClaims, claims.Tier)
	}
	return claims, nil
}
