# ADR-0003: One Opaque Key Per Active Device

Date: 2026-06-17

## Status

Accepted

## Context

The first commercial model is prepaid access without a conventional customer account. The product must avoid usernames, passwords and traffic quotas.

## Decision

One opaque random access key authorises one active device lease. A second device may explicitly take over the lease.

The project will not introduce usernames, passwords or conventional customer accounts for the first release.

## Alternatives Considered

* Create user accounts with email and password.
* Require a customer portal before technical access.
* Use traffic-based quotas.

## Rationale

Opaque keys simplify access while avoiding customer-account scope and stored-password risk. One active device preserves a simple commercial control without metered traffic.

## Consequences

Raw access keys must not be stored or logged. Future entitlement work needs careful hashing, lease takeover semantics and support diagnostics that do not reveal secrets.

## Conditions for Revisiting

Revisit only if payment recovery, enterprise administration or fraud controls require a materially different identity model.
