# Monetisation

This document describes how continuity-vpn is monetised and how the entitlement
scaffold in `internal/entitlement` gates the paid tier without crippling the
free, open-source path. It uses British English throughout.

## The model: open core, paid hosted gateway

continuity-vpn follows an **open-core** model:

- **The client is open source and free.** Anyone may build and run it.
- **The gateway is open source and self-hostable for free.** Anyone may stand
  up their own gateway and point their own client at it at no cost.
- **Revenue comes from an optional paid hosted gateway.** We operate a managed
  gateway service so that users who do not want to run their own infrastructure
  can pay us to do it for them.

The commercial offering is therefore convenience and operations, not the
software itself. Nothing in the paid tier is allowed to degrade or restrict the
self-hosted path.

## What the entitlement gate does (and does not) gate

The entitlement check gates **only the hosted gateway**. It is expressed as a
`Mode` on the entitlement `Service`:

- **`ModeOpen` — self-hosted.** The gate is disabled. `Admit` always succeeds
  and **no token is required**. A self-hosted gateway runs in this mode, so the
  open-source path needs no account, no payment, and no entitlement token.
- **`ModeHosted` — paid hosted gateway.** The gate is enforced. `Admit` admits a
  client only on a valid, unexpired, correctly-signed entitlement token for the
  hosted tier.

The default zero value of `Mode` is `ModeOpen`, so a gateway built without any
billing configuration safely behaves as a free self-hosted gateway. The paid
gateway must opt in to enforcement explicitly.

## The token shape

An entitlement token is **opaque, signed, and expiring**:

- **Opaque.** It carries only an opaque subject identifier, a tier, and an
  expiry. It contains **no personally-identifying data** — no email address, no
  name. The subject is an internal reference issued by the billing system.
- **Signed.** The token is a compact `payload.signature` string. The payload is
  the base64url encoding of the claims; the signature is an HMAC-SHA256 of the
  payload under a server-held secret key. Verification rejects a tampered token,
  a token signed with the wrong key, and an expired token. Signature comparison
  is constant-time.
- **URL-safe.** Encoding is base64url without padding, so the token contains no
  `+`, `/`, or `=` and travels safely in URLs, headers, and config files.
- **Expiring.** The claims carry an expiry as Unix seconds. A token is valid up
  to and including its expiry second and is rejected afterwards. Verification
  takes an injectable clock, so expiry behaviour is deterministically testable.

The token is built with the Go standard library only (`crypto/hmac`,
`crypto/sha256`, `encoding/base64`); it is intentionally **not** a JWT, to avoid
a dependency and the well-known JWT footguns (algorithm confusion, `alg=none`).

## The provider abstraction

Payment is modelled behind a `PaymentProvider` interface so the codebase never
binds to one billing vendor:

- `CreateCheckout(subject, tier)` starts a purchase and returns a checkout
  session (a hosted checkout URL the user opens to pay).
- `VerifyWebhook(payload, signature)` authenticates an inbound webhook against
  its signature and returns a normalised `PaymentEvent`. The service never
  trusts raw webhook bytes; the provider verifies the signature first.

A deterministic, in-memory `FakeProvider` implements this interface for tests
and local development. It performs **no network calls** and embeds **no real
secrets** — the webhook signing secret is supplied by the caller.

The flow that mints an entitlement is:

1. The billing webhook handler receives a provider webhook.
2. `Service.OnPaymentEvent(payload, signature)` calls the provider to verify the
   webhook and, on a paid event, issues an entitlement token for the subject.
3. The token is delivered to the client (out of band).
4. The hosted gateway calls `Service.Admit(token)` to admit the client.

## What remains for a real launch

The scaffold is deliberately self-contained and dependency-free. Before a real
commercial launch the following work remains:

- **A real payment integration.** Implement `PaymentProvider` against an actual
  backend — Stripe for web, and/or App Store / Play in-app purchases for mobile
  — with real publishable/secret keys held in secret storage, real checkout
  session creation, and real webhook signature verification.
- **Account and subscription storage.** Persist the mapping from billing
  customer to opaque subject, the current subscription state, and issued-token
  metadata, so entitlements can be re-issued, renewed, and revoked.
- **Key management.** Provision, rotate, and protect the token-signing key
  (e.g. in a KMS or secrets manager), and support overlapping keys during
  rotation.
- **Token delivery and renewal.** A flow for the client to fetch and refresh its
  entitlement token as subscriptions renew or expire.
- **Revocation.** A path to invalidate an entitlement before its natural expiry
  (e.g. on refund or chargeback), since short-lived signed tokens otherwise
  remain valid until they expire.

None of the above changes the principle: the gate applies to the hosted tier
only, and the self-hosted, open-source path stays free and ungated.
