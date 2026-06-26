// Package entitlement gates the optional paid hosted gateway tier.
//
// The gemina client and gateway are open source and self-hostable for
// free; revenue comes only from an optional paid hosted gateway. This package
// therefore gates the hosted tier only: a self-hosted gateway runs the Service
// in ModeOpen, where Admit always succeeds and no token is required, while the
// paid hosted gateway runs in ModeHosted and admits a client only on a valid,
// unexpired, correctly-signed entitlement token.
//
// It provides three pieces:
//
//   - A compact, URL-safe, HMAC-SHA256-signed token (Issue/Verify) carrying
//     opaque claims {subject, tier, expiry} and no personally-identifying data.
//   - A PaymentProvider abstraction over a real billing backend, with a
//     deterministic in-memory FakeProvider for tests and local development.
//   - A Service that mints a token from a verified payment webhook and exposes
//     Admit for the gateway to call.
//
// The package uses the Go standard library only and contains no real API keys,
// secrets, or network calls.
package entitlement
