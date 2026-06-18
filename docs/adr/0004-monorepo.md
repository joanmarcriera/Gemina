# ADR-0004: Monorepo

Date: 2026-06-17

## Status

Accepted

## Context

The project is starting from a fresh repository and will initially combine macOS client, Go transport, gateway, infrastructure, documentation and research due diligence.

## Decision

Use one monorepo initially.

Do not split repositories until there are independent release cycles, access-control requirements or operational reasons that outweigh the coordination cost.

## Alternatives Considered

* Separate client, gateway, infrastructure and documentation repositories immediately.
* Keep research material outside version control entirely.

## Rationale

Stage 0 depends on shared architecture, provenance and CI controls. A monorepo keeps the early boundary decisions visible and reviewable.

## Consequences

The repository must keep clear package boundaries and must not commit full upstream source repositories. Research clones belong in `.research-src/`.

## Conditions for Revisiting

Revisit if client, gateway or infrastructure release independently, or if access-control requirements require separation.
