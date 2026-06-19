# Stage 0 Review Request

Date: 2026-06-19

Repository: `joanmarcriera/continuity-vpn`

Review evidence baseline commit: `1e3b461`

CI evidence commits:

* `fcd6238` — first passing push-triggered Stage 0 CI after `workflow_dispatch` was added.
* `4a8afd4` — latest passing push-triggered Stage 0 CI at review time after action-version hardening.

Review tracking:

* Engineering review: https://github.com/joanmarcriera/continuity-vpn/issues/1
* Legal/provenance review: https://github.com/joanmarcriera/continuity-vpn/issues/2

## Purpose

This document requests Stage 0 engineering review and legal/provenance review before any Stage 1 transport work begins.

Stage 0 created the repository controls, architecture records, provenance boundaries, source due-diligence records, skeleton code and CI evidence required before implementation. This document is not reviewer approval and is not legal advice.

## Scope Reviewed By This Request

In scope:

* Repository bootstrap and structure.
* Product scope and operating rules.
* Architecture and ADR baseline.
* Upstream source manifest and root licence-file inspection records.
* Provenance controls for future source imports.
* Go, Swift, infrastructure and licence CI skeletons.
* Stage 1 backlog readiness, without starting Stage 1.

Out of scope:

* Stage 1 dual-path UDP probe implementation.
* VPN transport, packet framing, deduplication or gateway runtime.
* NetworkExtension packet handling.
* Payments, entitlement service or customer access-key flows.
* Any legal approval to import or distribute third-party source.

## Stage 0 Exit Criteria Evidence

| Criterion | Status | Evidence |
| --- | --- | --- |
| Clean checkout bootstraps successfully | Complete | `make clean-workspace-check` passed from a temporary copy. |
| All upstream projects are pinned | Complete | `research/upstream-manifest.yaml` contains exact commits for all listed upstreams. |
| Licence classifications are documented | Engineering record complete, legal review pending | `docs/legal/upstream-licences.md` and `docs/legal/dependency-inventory.md`. |
| No GPL source copied into product directories | Complete by repository inspection and provenance records | `.research-src/` is ignored; `docs/legal/code-provenance.md` records no imports. |
| CI runs on empty skeleton | Complete | GitHub CI passed on commits `fcd6238` and `4a8afd4`. |
| ADR-0001 records continuity-first decision | Complete | `docs/adr/0001-continuity-first.md`. |

## GitHub CI Evidence

GitHub CI passed for push-triggered commit `fcd6238`:

* Go CI: `27816489936`
* Infrastructure CI: `27816489953`
* Licence Scan: `27816489977`
* macOS CI: `27816489933`

GitHub CI passed again for push-triggered commit `4a8afd4` after CI action-version hardening:

* Go CI: `27820615456`
* Infrastructure CI: `27820615438`
* Licence Scan: `27820615455`
* macOS CI: `27820615442`

Notes:

* The first repository push produced GitHub Actions run `27815677467` with `startup_failure`, no jobs and no logs.
* The workflows were later made manually dispatchable; subsequent push commits `fcd6238` and `4a8afd4` triggered the path-filtered workflows successfully.
* The `4a8afd4` hardening removed the non-blocking GitHub Node.js runtime and missing `go.sum` cache annotations. A Homebrew tap-trust warning remains while installing SwiftLint from runner state.

## Licence And Provenance Review Request

Review requested:

* Confirm whether the recorded root licence classifications are sufficient for Stage 0.
* Confirm that no upstream implementation source has been imported into product directories.
* Confirm that Engarde and OpenMPTCProuter remain inspiration-only.
* Confirm that WireGuard Apple and wireguard-go may be considered for later reuse only after notices, import records and legal review.
* Confirm that future imports must update `docs/legal/code-provenance.md`, `docs/legal/upstream-licences.md`, `docs/legal/dependency-inventory.md` and `NOTICE` as applicable.

Legal/provenance records:

* `docs/legal/upstream-licences.md`
* `docs/legal/code-provenance.md`
* `docs/legal/dependency-inventory.md`
* `research/upstream-manifest.yaml`
* `NOTICE`

## Engineering Review Request

Review requested:

* Confirm that the Stage 0 repository controls are sufficient to start Stage 1.
* Confirm that Stage 1 remains correctly scoped to a dual-path UDP probe, not a VPN.
* Confirm that the SwiftPM-only macOS scaffold is acceptable until signing and entitlement details are known.
* Confirm that the Go and Swift skeletons do not imply transport behaviour that has not been proved.
* Confirm that the CI evidence is sufficient as clean-checkout validation for Stage 0.

Engineering records:

* `AGENTS.md`
* `PROJECT_STATE.md`
* `TASKS.md`
* `DECISIONS.md`
* `docs/product/project-specification.md`
* `docs/architecture/overview.md`
* `docs/adr/`
* `docs/backlog/stage-1.md`
* `.github/workflows/go-ci.yml`
* `.github/workflows/infra-ci.yml`
* `.github/workflows/licence-scan.yml`
* `.github/workflows/macos-ci.yml`
* `.github/workflows/release.yml` (disabled Stage 0 placeholder)

## Known Non-Blocking Hardening

These items do not need to block review unless reviewers decide otherwise:

* Create a full Xcode project when signing, entitlements and bundle identifiers are known.
* Add real Swift XCTest and UI tests after the Xcode project exists.
* Run local OpenTofu validation in an environment with `tofu` or `terraform` installed.
* Run local SwiftLint where `swiftlint` is installed.
* Review non-blocking GitHub CI runner annotations for future hardening.

## Stage 1 Gate

Do not begin Stage 1 until:

* Stage 0 engineering review issue 1 is complete.
* Legal/provenance review issue 2 is complete or has explicitly approved a narrower Stage 1 scope.
* Any required review follow-ups are recorded in `TASKS.md` or GitHub issues.

The first Stage 1 issue remains:

* Prove that UDP socket A explicitly leaves through Wi-Fi, UDP socket B explicitly leaves through Android USB tethering, both reach the same Hetzner process, one logical packet is delivered once, and either path can disappear without ending the logical session.
