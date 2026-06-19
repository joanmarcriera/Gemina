# Tasks

Last updated: 2026-06-19

## Current Objective: Stage 0 Repository Bootstrap and Source Due Diligence

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
* [ ] Full legal review is complete before any import.
* [x] CI has run on a clean checkout.
* [x] Atomic Stage 0 bootstrap commit has been created.
* [ ] Stage 0 exit criteria have been reviewed.

## Next Exact Action

Completion criteria:

* [x] Stage all outstanding Stage 0 files.
* [x] Run `git diff --cached --check`.
* [x] Create the atomic initial bootstrap commit.
* [x] Confirm ignored artefacts remain untracked: `.research-src/`, `.build/`, `.codex/` and `apps/macos/.build/`.
* [x] Configure or confirm the chosen Git remote.
* [x] Push the initial bootstrap commit to the chosen remote.
* [x] Run GitHub CI or equivalent clean-checkout validation.
* [x] Update `PROJECT_STATE.md` with CI or clean-checkout validation results.
* [x] Request Stage 0 review before Stage 1 work starts.

## Remaining Stage 0 Hardening

* [ ] Decide whether to create the Xcode project manually now or keep SwiftPM-only validation until signing details are available.
* [ ] Add real Swift XCTest/UI tests after the Xcode project and full Apple test toolchain are configured.
* [ ] Run OpenTofu validation in an environment with `tofu` installed.
* [ ] Run SwiftLint in an environment with `swiftlint` installed.
* [x] Investigate GitHub Actions startup failure from run `27815677467`.
* [x] Inspect GitHub Actions results after pushing the initial commit.
* [ ] Review non-blocking GitHub CI annotations for future hardening.
* [x] Request Stage 0 review before any Stage 1 transport work.
* [ ] Complete Stage 0 engineering review issue 1.
* [ ] Complete Stage 0 legal/provenance review issue 2.

## Stage 1 Candidate, Not Yet Started

First engineering issue after Stage 0 review:

* [ ] Prove that UDP socket A explicitly leaves through Wi-Fi, UDP socket B explicitly leaves through Android USB tethering, both reach the same Hetzner process, one logical packet is delivered once, and either path can disappear without ending the logical session.

Do not start this task until Stage 0 is complete and reviewed.
