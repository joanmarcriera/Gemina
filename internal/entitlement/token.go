package entitlement

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Tier names the access level an entitlement grants. Tiers carry no pricing or
// billing detail; they only distinguish what a verified client may use.
type Tier string

const (
	// TierHosted is the paid hosted-gateway tier. Clients must present a valid
	// entitlement token to be admitted by a hosted gateway.
	TierHosted Tier = "hosted"
	// TierSelfHosted is the implicit tier granted by a self-hosted (open-mode)
	// gateway. It is never carried in a signed token; it is what Admit returns
	// when the entitlement gate is disabled.
	TierSelfHosted Tier = "self-hosted"
)

// validTier reports whether t is a tier a token may legitimately carry.
func validTier(t Tier) bool {
	return t == TierHosted
}

// Claims are the assertions carried by an entitlement token. The token carries
// no personally-identifying data: Subject is an opaque, provider-issued
// identifier, never an email address or name.
type Claims struct {
	// Subject is an opaque subject identifier (e.g. an internal account or
	// customer reference). It must not encode personal data.
	Subject string
	// Tier is the access level granted.
	Tier Tier
	// Expiry is the expiry instant as Unix seconds. A token is valid up to and
	// including this second.
	Expiry int64
}

// Clock supplies the current time. Injecting a Clock keeps verification
// deterministic in tests; production code passes a real-time clock.
type Clock interface {
	Now() time.Time
}

// SystemClock is a Clock backed by time.Now.
type SystemClock struct{}

// Now returns the current wall-clock time.
func (SystemClock) Now() time.Time { return time.Now() }

// Token errors. Callers can match these with errors.Is to distinguish the
// rejection reason without inspecting message text.
var (
	ErrMalformedToken = errors.New("entitlement: malformed token")
	ErrBadSignature   = errors.New("entitlement: signature mismatch")
	ErrExpired        = errors.New("entitlement: token expired")
	ErrInvalidClaims  = errors.New("entitlement: invalid claims")
	ErrNoKey          = errors.New("entitlement: empty signing key")
)

// b64 is URL-safe base64 without padding, keeping tokens free of +, / and =.
var b64 = base64.RawURLEncoding

// Issue produces a signed, URL-safe entitlement token for the given claims.
//
// The token format is "payload.signature" where payload is the base64url
// encoding of "subject|tier|expiry" and signature is the base64url HMAC-SHA256
// of the payload under key. The "|" separator is rejected inside Subject so the
// payload cannot be forged by field injection.
func Issue(c Claims, key []byte) (string, error) {
	if len(key) == 0 {
		return "", ErrNoKey
	}
	if c.Subject == "" || strings.Contains(c.Subject, "|") {
		return "", fmt.Errorf("%w: subject", ErrInvalidClaims)
	}
	if !validTier(c.Tier) {
		return "", fmt.Errorf("%w: tier %q", ErrInvalidClaims, c.Tier)
	}
	if c.Expiry <= 0 {
		return "", fmt.Errorf("%w: expiry", ErrInvalidClaims)
	}

	raw := c.Subject + "|" + string(c.Tier) + "|" + strconv.FormatInt(c.Expiry, 10)
	payload := b64.EncodeToString([]byte(raw))
	sig := sign(payload, key)
	return payload + "." + sig, nil
}

// Verify checks a token's signature and expiry against key, using now for the
// expiry comparison, and returns its claims. It rejects tampered tokens, tokens
// signed with the wrong key, and expired tokens.
func Verify(token string, key []byte, now Clock) (Claims, error) {
	if len(key) == 0 {
		return Claims{}, ErrNoKey
	}
	payload, gotSig, ok := strings.Cut(token, ".")
	if !ok || payload == "" || gotSig == "" {
		return Claims{}, ErrMalformedToken
	}

	wantSig := sign(payload, key)
	// Constant-time compare guards against signature-timing attacks.
	if !hmac.Equal([]byte(gotSig), []byte(wantSig)) {
		return Claims{}, ErrBadSignature
	}

	raw, err := b64.DecodeString(payload)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: payload encoding", ErrMalformedToken)
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return Claims{}, fmt.Errorf("%w: payload fields", ErrMalformedToken)
	}
	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: expiry", ErrMalformedToken)
	}
	c := Claims{Subject: parts[0], Tier: Tier(parts[1]), Expiry: exp}

	if !validTier(c.Tier) {
		return Claims{}, fmt.Errorf("%w: tier %q", ErrInvalidClaims, c.Tier)
	}
	// Valid up to and including the expiry second.
	if now.Now().Unix() > c.Expiry {
		return Claims{}, ErrExpired
	}
	return c, nil
}

// sign returns the base64url HMAC-SHA256 of payload under key.
func sign(payload string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return b64.EncodeToString(mac.Sum(nil))
}
