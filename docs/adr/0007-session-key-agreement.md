# ADR-0007: Session-Key Agreement (X25519 + HKDF)

Date: 2026-06-23

## Status

Accepted

## Context

ADR-0006 encrypts each packet with an AES-256-GCM key but assumed the key was
pre-shared. A pre-shared key does not scale (every client needs a distinct key
provisioned out of band) and gives no forward secrecy. The encryption format must
not change; only how the 32-byte key is established.

## Decision

Derive the per-session key with an **X25519 ECDH** exchange followed by
**HKDF-SHA256**, using only the Go standard library (`crypto/ecdh`, `crypto/hkdf`
— no invented cryptography), in `pkg/clientcore` (`GenerateKeyPair`,
`DeriveSessionKey`):

* Each endpoint generates an ephemeral X25519 key pair and sends its public key.
* Both compute `shared = ECDH(myPriv, peerPub)` (symmetric) and then
  `key = HKDF-SHA256(secret = shared, salt = sessionID, info = "continuity-vpn
  session key v1", len = 32)`. Salting by the session id binds the key to the
  session and separates keys across sessions between the same parties.
* The key is role-independent; nonce direction separation (ADR-0006) handles the
  two directions, so both ends derive one identical key.

Ephemeral key pairs give **forward secrecy**: a later key compromise does not
decrypt past sessions.

## Alternatives Considered

* **Keep pre-shared keys.** No forward secrecy; painful provisioning.
* **Adopt the full Noise/WireGuard handshake.** Stronger (mutual auth, identity
  hiding, rekeying) but heavier and a larger dependency/clean-room surface; the
  X25519+HKDF core is the same primitive and can be upgraded to Noise later
  without changing the packet format.
* **TLS to the gateway.** Heavier, stream-oriented, and awkward over the
  duplicated datagram paths.

## Consequences

* This ADR covers key *derivation*, not *authentication of the public keys*. As
  written it is secure against a passive eavesdropper but not an active
  man-in-the-middle: the transport that exchanges the public keys must
  authenticate the gateway's key — e.g. pin the gateway static key in the client,
  or carry/cosign it via the signed entitlement token (`internal/entitlement`).
  Tracked in `TASKS.md`.
* The exchange that carries the public keys (a small handshake datagram before
  data, or in the connection setup) is still to be wired into the gateway and the
  NE provider.
* Rekeying / session rotation before the AEAD counter could wrap remains a
  production concern (ADR-0006).

## Conditions for Revisiting

Upgrade to a full Noise pattern if mutual authentication, identity hiding or
formal rekeying are required, or if a reviewed reuse of the WireGuard handshake
is preferred.
