package entitlement

import (
	"strings"
	"testing"
	"time"
)

// fixedClock is an injectable clock for deterministic expiry tests.
type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse time %q: %v", s, err)
	}
	return parsed
}

func TestTokenRoundTrip(t *testing.T) {
	key := []byte("server-secret-key-for-tests")
	issued := mustTime(t, "2026-01-01T00:00:00Z")
	want := Claims{
		Subject: "sub_opaque_123",
		Tier:    TierHosted,
		Expiry:  issued.Add(24 * time.Hour).Unix(),
	}

	tok, err := Issue(want, key)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// Token must be URL-safe (base64url, no padding, no +/ characters).
	if strings.ContainsAny(tok, "+/=") {
		t.Errorf("token is not URL-safe: %q", tok)
	}

	got, err := Verify(tok, key, fixedClock{issued})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != want {
		t.Errorf("round-trip mismatch: got %+v want %+v", got, want)
	}
}

func TestVerifyRejections(t *testing.T) {
	key := []byte("server-secret-key-for-tests")
	wrongKey := []byte("a-different-secret-key------")
	issued := mustTime(t, "2026-01-01T00:00:00Z")
	claims := Claims{
		Subject: "sub_opaque_123",
		Tier:    TierHosted,
		Expiry:  issued.Add(time.Hour).Unix(),
	}

	good, err := Issue(claims, key)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	tests := []struct {
		name  string
		token string
		key   []byte
		now   time.Time
	}{
		{
			name:  "tampered payload",
			token: tamper(good),
			key:   key,
			now:   issued,
		},
		{
			name:  "wrong key",
			token: good,
			key:   wrongKey,
			now:   issued,
		},
		{
			name:  "expired",
			token: good,
			key:   key,
			now:   issued.Add(2 * time.Hour),
		},
		{
			name:  "malformed structure",
			token: "not-a-valid-token",
			key:   key,
			now:   issued,
		},
		{
			name:  "empty",
			token: "",
			key:   key,
			now:   issued,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Verify(tc.token, tc.key, fixedClock{tc.now}); err == nil {
				t.Errorf("Verify accepted %s token, want rejection", tc.name)
			}
		})
	}
}

// tamper flips a byte in the payload section of a token so the signature no
// longer matches, simulating an attacker editing claims in transit.
func tamper(tok string) string {
	parts := strings.SplitN(tok, ".", 2)
	if len(parts) != 2 {
		return tok + "x"
	}
	payload := []byte(parts[0])
	// Flip a character that stays within the base64url alphabet so the
	// failure is signature mismatch, not a decode error.
	if payload[0] == 'A' {
		payload[0] = 'B'
	} else {
		payload[0] = 'A'
	}
	return string(payload) + "." + parts[1]
}

func TestVerifyExpiryBoundary(t *testing.T) {
	key := []byte("server-secret-key-for-tests")
	issued := mustTime(t, "2026-01-01T00:00:00Z")
	exp := issued.Add(time.Hour)
	claims := Claims{Subject: "sub", Tier: TierHosted, Expiry: exp.Unix()}

	tok, err := Issue(claims, key)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// At exactly the expiry second the token is still valid.
	if _, err := Verify(tok, key, fixedClock{exp}); err != nil {
		t.Errorf("token rejected at expiry boundary: %v", err)
	}
	// One second past expiry it is rejected.
	if _, err := Verify(tok, key, fixedClock{exp.Add(time.Second)}); err == nil {
		t.Errorf("token accepted one second past expiry")
	}
}
