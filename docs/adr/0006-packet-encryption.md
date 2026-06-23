# ADR-0006: Identity-Bound Packet Encryption

Date: 2026-06-23

## Status

Accepted

## Context

The dual-path transport duplicates each packet over two links to a gateway, which
deduplicates by identity. The traffic crosses public Wi-Fi and a mobile carrier,
so payloads must be confidential and tamper-evident. But the product's objective
is a *consistent network experience*: encryption must not perturb the duplication
or add handshakes/round-trips that hurt latency or failover, and it must not
break dedup (which operates on the cleartext identity before decryption).

The constraint from the owner: not "high-level encryption that tampers with the
packages", but something aligned with the project — confidentiality and integrity
that ride on the existing identity rather than adding a second, conflicting
sequencing/anti-replay scheme.

## Decision

Encrypt each data packet's payload with **AES-256-GCM** (Go standard library; no
invented cryptography), with the nonce derived deterministically from the packet
identity and a direction bit, and the cleartext frame header as the additional
authenticated data (AAD).

* **Wire format (`CVD1`, `pkg/clientcore`):** cleartext header — magic, version,
  flags (bit 0 = direction), 16-byte session, 8-byte packet number — followed by
  the AES-256-GCM ciphertext+tag of the payload. The header is cleartext so the
  gateway can deduplicate by identity without holding a key, and is authenticated
  (it is the AAD) so identity/direction cannot be forged.
* **Nonce = direction ‖ packet number.** The packet number is unique and
  monotonic per session per direction; the direction byte keeps the two endpoints
  (which share one session key) from colliding on numbers 1, 2, 3, … So no
  `(key, nonce)` pair ever repeats — the GCM safety requirement.
* **Identity-bound, duplication-safe.** Both duplicate copies of one logical
  packet share an identity, hence the same nonce and *identical ciphertext*. The
  dual-path duplication and dedup-by-identity are therefore unchanged by
  encryption; either copy decrypts identically.
* **Authenticate before dedup.** `Inbound` verifies the AEAD tag before the
  packet touches the dedup window, so a forged packet cannot suppress a real one
  by replaying its identity. Anti-replay across the wire is provided by the
  existing dedup window; the AEAD provides confidentiality and integrity.
* **Key agreement is out of scope here.** The 32-byte session key is supplied to
  the session (a pre-shared key for now, shared client↔gateway). A proper
  handshake (e.g. a Noise/WireGuard-style exchange) is future work and does not
  change this packet format.

## Alternatives Considered

* **Wrap the whole tunnel in WireGuard.** WireGuard adds its own counters,
  anti-replay window and handshake. Running our duplication underneath it would
  fight its replay protection (it would treat the second copy as a replay) or
  require running it above duplication (encrypting twice). The identity-bound
  AEAD reuses our existing identity instead of layering a second scheme.
* **Random per-packet nonces.** Larger frames and a birthday-bound risk over long
  sessions; the deterministic counter nonce is smaller and exact.
* **Encrypt the identity too.** Then the gateway must decrypt before it can dedup,
  forcing it to hold every session key on the hot path and decrypt duplicates.
  Keeping the (non-identifying) identity cleartext lets dedup run first.

## Consequences

* The gateway must hold each session's key to decrypt and forward; key
  distribution is the job of the (future) handshake/entitlement path.
* GCM safety depends on never reusing `(key, number, direction)`; the monotonic
  per-session counter and the `PacketNumber == 0` invalid rule must hold, and
  sessions must rekey before the counter could wrap (a production concern at high
  rates).
* The identity (random session + counter) is visible on the wire. It carries no
  host identifier, but it is a correlation handle for an on-path observer within
  a session's lifetime; session rotation bounds this.

## Conditions for Revisiting

Revisit when the key-agreement handshake lands (it may fold the direction/keys
differently), if traffic-analysis resistance becomes a requirement (padding,
identity rotation), or if a reviewed reuse of WireGuard foundations is preferred
over the bespoke AEAD framing.
