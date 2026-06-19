# Contributing

This repository is in Stage 1 dual-path UDP probe work.

Before making changes:

1. Read `AGENTS.md`, `PROJECT_STATE.md`, `TASKS.md`, `DECISIONS.md` and `docs/product/project-specification.md`.
2. Pick one bounded objective from the current stage in `TASKS.md`.
3. Avoid production VPN transport, NetworkExtension packet handling, payment and entitlement implementation unless the relevant later-stage gate has explicitly opened.
4. Record material architectural choices in `DECISIONS.md` and, where required, under `docs/adr/`.
5. Run focused tests and update the project-state files before handing over.

Documentation uses British English. Prefer `licence` in prose except for fixed filenames, SPDX identifiers and upstream names that use `license`.
