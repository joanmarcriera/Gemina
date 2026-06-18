# ADR-0002: Swift Client and Go Core

Date: 2026-06-17

## Status

Accepted

## Context

The product is macOS-only for the first release and must integrate with NetworkExtension. The shared transport core and Linux gateway need network-programming ergonomics, testability and a path to reuse WireGuard Go foundations.

## Decision

Use Swift and SwiftUI for the macOS app, NetworkExtension integration and user-facing client code. Use Go for the shared transport core, gateway, control API and operational tools.

Define a narrow C-compatible boundary for Swift/Go integration.

## Alternatives Considered

* Implement all client code in Swift.
* Implement the macOS UI and extension in Go.
* Use Rust for the transport core.

## Rationale

Swift is the native choice for macOS UI and NetworkExtension lifecycle work. Go is a pragmatic fit for the gateway, transport experiments and wireguard-go reuse after review.

Keeping the boundary narrow reduces memory-ownership risk between Swift and Go.

## Consequences

Stage 0 keeps both toolchains present but does not implement the bridge. Future bridge work must verify memory ownership, callback lifetime and logging behaviour.

## Conditions for Revisiting

Revisit if the Swift/Go boundary becomes unsafe, untestable or blocks NetworkExtension requirements.
