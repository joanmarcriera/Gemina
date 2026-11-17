# Contributing

Gemina is source-available under the PolyForm Noncommercial License 1.0.0. It is free
for noncommercial use with attribution; commercial use needs a paid licence (see
[`LICENSE`](LICENSE), [`COMMERCIAL.md`](COMMERCIAL.md), [`NOTICE`](NOTICE) and
[`docs/legal/licensing.md`](docs/legal/licensing.md)).

This repository is pre-release; the current stage is recorded in `TASKS.md`.
By sending a contribution you agree that it is licensed under the same terms and
that Riera Technologies Ltd may also license it commercially.

The most useful contribution today is a report from real hardware. See the
README section "Report results from real hardware".

Before making changes:

1. Read `AGENTS.md`, `PROJECT_STATE.md`, `TASKS.md`, `DECISIONS.md` and `docs/product/project-specification.md`.
2. Pick one bounded objective from the current stage in `TASKS.md`.
3. Avoid production VPN transport, NetworkExtension packet handling, payment and entitlement implementation unless the relevant later-stage gate has explicitly opened.
4. Record material architectural choices in `DECISIONS.md` and, where required, under `docs/adr/`.
5. Run focused tests and update the project-state files before handing over.

Documentation uses British English. Prefer `licence` in prose except for fixed filenames, SPDX identifiers and upstream names that use `license`.
