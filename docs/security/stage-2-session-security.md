# Stage 2 Session Security

## Scope

Component: the gateway session admission and data plane.

In scope:

* `internal/gateway` session admission (`Admitter`, `SessionStore`).
* `internal/gateway` data plane decryption and deduplication (`DataPlane`).
* Per-session key registration and lookup.

Out of scope (covered elsewhere or deferred):

* The on-wire handshake message and pinned-identity distribution.
* Payment and entitlement token issuance internals.
* macOS client and NetworkExtension behaviour.

## Single-use SessionID/key invariant

A SessionID/key pair is **single-use for the gateway's lifetime**.

A fresh handshake mints a fresh cryptographically-random SessionID (128 bits) and
derives a fresh session key by X25519 key agreement (ADR-0007). The gateway never
reuses a SessionID for a second key. A repeated SessionID is therefore either a
duplicate ClientHello or a deliberate attempt to rebind a live session to an
attacker-chosen key.

**Rationale.** Without this invariant, a peer could present a previously-admitted
SessionID alongside a fresh ephemeral key and have the gateway overwrite the
established key, binding the id to a new ("fresh") session. That would let a
party who can choose the SessionID rebind a session it does not own, and would
let a captured datagram for an old session decrypt under a newly-installed key.

**Enforcement.** `SessionStore.register` performs an atomic test-and-set under
its mutex: it refuses to overwrite an existing id and reports the id was taken.
`Admitter.Admit` then returns `ErrSessionReused` and registers nothing, so
admission is fail-closed — the data plane holds no key for the rebind attempt and
rejects its packets as an unknown session. The dual-path client performs **one**
handshake per session and then duplicates data across paths (see
`tests/end-to-end/rig_linux.go`), so refusing a repeated handshake never breaks a
legitimate flow. Covered by `TestAdmitterRejectsReusedSessionID`.

## Residual risk

The session table is in-memory and per-session. Sessions reset on gateway
restart by design: a restart loses all keys, a re-handshake yields a fresh
SessionID+key, and no old datagram decrypts into a new session. Surviving a
restart for long-lived sessions is out of scope; if it is ever required, persist
only the scalar replay high-water mark, not the key.

The set of admitted SessionIDs grows with the number of distinct sessions for the
process lifetime. In the hosted tier this is bounded by entitlement and
handshake rate limiting; a self-hosted gateway runs on the operator's own box.
Replay-window width sizing against worst-case path skew is tracked separately.
