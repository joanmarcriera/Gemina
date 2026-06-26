# ADR-0001: Gemina First

Date: 2026-06-17

## Status

Accepted

## Context

The product must keep calls and remote-working sessions alive over unreliable train Wi-Fi and Android USB tethering. The first feasibility risk is whether macOS can send protected traffic over two simultaneous upstream paths and whether a gateway can accept the first valid copy while discarding duplicates.

Bandwidth aggregation would require scheduler, fairness, measurement and user-expectation work before the core reliability claim is proven.

## Decision

The first commercial release is a continuity product, not an aggregate-bandwidth product.

It duplicates protected traffic over Wi-Fi and Android USB tethering, sends both copies to one gateway, accepts the first valid packet and discards duplicates.

## Alternatives Considered

* Start with aggregate bonding.
* Start with Multipath QUIC or MASQUE.
* Start with a general-purpose multi-region VPN.

## Rationale

Gemina directly matches the product proposition: stable calls, SSH, RDP and remote-working sessions during access-network loss.

Aggregation adds major transport complexity and could create false marketing claims before path control and duplicate delivery are proven.

## Consequences

Stage 1 must prove explicit per-path UDP transmission, gateway deduplication and session survival during path loss before UI polish, payment flows or broader infrastructure work become priorities.

Marketing and documentation must not claim doubled speed, zero packet loss, unbreakable connectivity or anonymity.

## Conditions for Revisiting

Revisit only after Stage 1 and real train testing produce measurable continuity evidence, or if the commercial proposition is explicitly changed.
