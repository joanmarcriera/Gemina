# Tasks

Last updated: 2026-06-19

## Current Objective: Stage 1 Dual-Path UDP Probe

Completion criteria for Stage 0 exit:

* [x] Repository structure exists.
* [x] Product specification and agent operating rules are stored in the repository.
* [x] Architecture overview exists.
* [x] ADR template exists.
* [x] ADR-0001 records the continuity-first decision.
* [x] ADR-0002 records Swift client and Go transport/gateway.
* [x] ADR-0003 records one opaque key per active device.
* [x] ADR-0004 records the monorepo decision.
* [x] Legal and provenance templates exist.
* [x] Security threat-model template exists.
* [x] Makefile exists with bootstrap, test, test-go, test-macos, lint, licence-check, fetch-research and docs-check targets.
* [x] Clean-workspace validation target exists.
* [x] Go workspace and skeleton Go module exist.
* [x] Baseline macOS source directories and manual Xcode creation instructions exist.
* [x] Baseline Go CI exists.
* [x] Baseline macOS Swift build/lint CI exists.
* [x] Baseline OpenTofu validation CI exists.
* [x] Baseline licence scanning CI exists.
* [x] Stage 1 backlog exists without starting Stage 1 implementation.
* [x] Focused validation commands were run and recorded in `PROJECT_STATE.md`.
* [x] All upstream projects are pinned with shell-verified commits.
* [x] `make fetch-research` has been run successfully with network access.
* [x] Root upstream licence files have been inspected and recorded.
* [x] `make clean-workspace-check` has passed from a temporary copy.
* [x] Stage 0 legal/provenance records review is complete for the no-import Stage 0 scope.
* [x] CI has run on a clean checkout.
* [x] Atomic Stage 0 bootstrap commit has been created.
* [x] Stage 0 exit criteria have been reviewed.

## Next Exact Action

Add a Darwin observation adapter boundary for path observations.

Completion criteria:

* [ ] Define an adapter boundary that returns `internal/paths.Observation` values.
* [ ] Add fixture-driven tests for Darwin observation data without live socket binding.
* [ ] Keep BSD interface names as data only; do not classify by names such as `en0`.
* [ ] Document which macOS evidence source will distinguish Wi-Fi from Android USB tethering.
* [ ] Update `PROJECT_STATE.md`, `TASKS.md` and `DECISIONS.md` with the Stage 1 implementation result.

## Completed Stage 1 Work

* [x] Define the initial probe package boundary below the network layer.
* [x] Add `internal/protocol` packet identity primitives.
* [x] Add `internal/dedup` first-copy duplicate suppression for one logical packet delivered once from duplicate path copies.
* [x] Add unit tests for identity validation, first-copy acceptance, duplicate rejection, invalid observations, bounded eviction and concurrent duplicate observations.
* [x] Run race detector for the dedup/protocol slice.
* [x] Define packet-capture and loss/recovery evidence required before claiming path success.
* [x] Update the Stage 1 threat model for packet identity and duplicate suppression.
* [x] Add `internal/paths` types for observed interfaces and path-candidate classification.
* [x] Identify Wi-Fi and Android USB tethering candidates from injected fixture data without hard-coded interface names.
* [x] Add unit tests for ambiguous, missing and multiple-candidate path observations.
* [x] Keep socket binding and live macOS API calls out of the path-candidate classification slice.

## Remaining Stage 0 Hardening

* [ ] Decide whether to create the Xcode project manually now or keep SwiftPM-only validation until signing details are available.
* [ ] Add real Swift XCTest/UI tests after the Xcode project and full Apple test toolchain are configured.
* [ ] Run OpenTofu validation in an environment with `tofu` installed.
* [ ] Run SwiftLint in an environment with `swiftlint` installed.
* [x] Investigate GitHub Actions startup failure from run `27815677467`.
* [x] Inspect GitHub Actions results after pushing the initial commit.
* [x] Review non-blocking GitHub CI annotations for future hardening.
* [x] Update Stage 0 CI action versions and dependency inventory.
* [ ] Decide whether to pin SwiftLint installation in macOS CI instead of relying on `brew install swiftlint`.
* [x] Request Stage 0 review before any Stage 1 transport work.
* [x] Complete Stage 0 engineering review issue 1.
* [x] Complete Stage 0 legal/provenance review issue 2.

## Review Follow-ups (from Stage 0 reviewer comments, 2026-06-19)

Source: `docs/reviews/stage-0-review-comments.md`. Non-blocking; recorded as a condition of approving Stage 1 start.

* [ ] Confirm PR path-filtered CI triggers run before Stage 1 merges. Push-triggered path-filtered CI has passed on `fcd6238` and `4a8afd4` after the initial `startup_failure` run `27815677467`.
* [ ] Add branch protection / required status checks on `main` before Stage 1 merges.
* [x] Reconcile commit-reference drift: `docs/reviews/stage-0-review-request.md` now cites both the original review baseline and the latest passing push-triggered CI on `4a8afd4`.
* [x] Add `release.yml` (disabled Stage 0 placeholder) to the engineering records/inventory for completeness.
* [ ] Pin SwiftLint install in macOS CI instead of unpinned `brew install swiftlint` (duplicate of hardening item above; close together).

## Legal/Provenance Standing Conditions (carry into Stage 1, from reviewer comments)

* [ ] Before importing ANY upstream file, run a per-file/subtree licence scan of the specific path (root-file inspection is not import clearance).
* [ ] Complete full legal review before any source import or distribution decision that depends on third-party source reuse.
* [ ] Enforce behavioural clean-room rule: whoever reads Engarde/OpenMPTCProuter (GPL) source must not author the corresponding dedup/transport core; produce clean-room notes before code is written.
* [ ] At WireGuard reuse time, capture wireguard-go/wireguard-apple copyright notices in `NOTICE` and `code-provenance.md`.
* [ ] If any MPL-2.0 file (terraform-provider-hcloud) is ever modified/vendored, honour per-file source-disclosure obligations.

## Stage 1 Overall Objective

Overall engineering proof:

* [ ] Prove that UDP socket A explicitly leaves through Wi-Fi, UDP socket B explicitly leaves through Android USB tethering, both reach the same Hetzner process, one logical packet is delivered once, and either path can disappear without ending the logical session.

Do not mark this complete until packet captures, gateway logs and path-loss evidence exist.
