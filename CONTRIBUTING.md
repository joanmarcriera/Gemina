# Contributing

This repository is in Stage 1 dual-path UDP probe work. It is heading towards an
open-core release: the client and the gateway are open source and self-hostable,
with an optional paid hosted gateway as the commercial model. Contributions to the
client and gateway are welcome under that model.

Note that the project licence is still being finalised for the open-source
release — see [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE). Until it is settled,
contributions are accepted on the understanding that they may be released under
the eventual open-source licence the owner selects.

Before making changes:

1. Read `AGENTS.md`, `PROJECT_STATE.md`, `TASKS.md`, `DECISIONS.md` and `docs/product/project-specification.md`.
2. Pick one bounded objective from the current stage in `TASKS.md`.
3. Avoid production VPN transport, NetworkExtension packet handling, payment and entitlement implementation unless the relevant later-stage gate has explicitly opened.
4. Record material architectural choices in `DECISIONS.md` and, where required, under `docs/adr/`.
5. Run focused tests and update the project-state files before handing over.

Documentation uses British English. Prefer `licence` in prose except for fixed filenames, SPDX identifiers and upstream names that use `license`.
