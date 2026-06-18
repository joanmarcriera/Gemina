# Project State

Last updated: 2026-06-18

## Current Objective

Stage 0 repository bootstrap and source due diligence.

This cycle completed a bounded Stage 0 commit objective: verify the bootstrap handover, stage all outstanding Stage 0 files, rerun local and clean-workspace validation, and create the atomic initial bootstrap commit.

The previous `.git` write blocker did not reproduce during this cycle.

## Completed Work

* Read `PROJECT_STATE.md`, `TASKS.md`, `DECISIONS.md` and `AGENTS.md`.
* Inspected Git status, staged state and ignored artefacts.
* Delegated a bounded handover-file review to `ollama_fast`; the response made unsupported generic claims, so local source inspection and tests remained authoritative.
* Staged the outstanding Stage 0 files that were previously blocked by `.git` write errors.
* Verified `git diff --cached --check` after staging.
* Added `scripts/validate-clean-workspace.sh`.
* Added `make clean-workspace-check`.
* The clean-workspace script copies the repository to a temporary directory while excluding:
  * `.git/`
  * `.build/`
  * `.codex/`
  * `.codex-cycle-started`
  * `.env`
  * `.research-src/`
  * `.terraform/`
  * `*.pem`
  * `*.key`
  * `*.tfstate`
  * `*.tfstate.*`
  * `DerivedData/`
  * `apps/macos/.build/`
  * `bin/`
  * `coverage.out`
* Hardened `scripts/validate-clean-workspace.sh` so it refuses to create the temporary validation copy inside the source tree.
* Ran `make clean-workspace-check`; it passed from a temporary copy under `/private/tmp`.
* Re-ran local validation before commit.
* Created the atomic initial Stage 0 bootstrap commit.
* Kept Stage 1 transport work deferred.

Prior completed Stage 0 work remains in place:

* Monorepo skeleton, root project controls, GitHub templates and CI stubs.
* ADR framework and ADR-0001 through ADR-0004.
* Architecture, security, legal, provenance and dependency-inventory documents.
* Go workspace and compile-only skeleton packages.
* SwiftPM macOS scaffold and manual Xcode creation instructions.
* Fully pinned upstream research manifest and exact-commit research fetch script.
* Root upstream licence-file inspection records.
* Workspace-local Go and Swift build caches for sandbox-compatible validation.

## Current Implementation State

No VPN transport, packet framing, deduplication, NetworkExtension packet handling, gateway runtime, entitlement service, payment flow or real infrastructure resource exists.

The repository has a validating Stage 0 skeleton. The upstream manifest is fully pinned by shell-verified commits. Research sources are present only in `.research-src/`, which is ignored by Git.

The initial Stage 0 bootstrap is committed. No Stage 1 implementation exists.

No Git remote is configured in this checkout yet.

Root upstream licence files have been inspected and recorded as Stage 0 engineering evidence, not legal advice or import approval.

The initial Xcode project is still intentionally not generated; `apps/macos/README.md` records the manual creation steps and the signing/entitlement decisions that must be reviewed first.

## Files Changed

Initial bootstrap content:

* Root project controls and validation files.
* `.github/` templates and workflows.
* `api/`, `apps/`, `bridge/`, `cmd/`, `db/`, `deploy/`, `internal/`, `observability/`, `pkg/`, `research/`, `scripts/` and `tests/`.
* `docs/adr/`, `docs/architecture/`, `docs/backlog/`, `docs/legal/`, `docs/operations/`, `docs/product/`, `docs/security/` and `docs/testing/`.
* Durable state files: `PROJECT_STATE.md`, `TASKS.md` and `DECISIONS.md`.

Ignored local artefacts:

* `.build/`
* `apps/macos/.build/`
* `.research-src/`
* `.codex/`
* `.codex-cycle-started`

## Tests Run and Results

Passed in this cycle:

* `make clean-workspace-check`
* `make test`
* `make lint`
  * Go formatting and docs checks passed.
  * SwiftLint was not installed locally, so the local SwiftLint run was skipped.
* `make licence-check`
* `git diff --check`
* `git diff --cached --check`
* `git status --short --ignored`

Previously passed and still reflected in the repository state:

* `make bootstrap`
* `make infra-check`
  * OpenTofu/Terraform was not installed locally, so local infra validation was skipped.
* `make fetch-research`
* `env GIT_CONFIG_GLOBAL=/dev/null git ls-remote ...` for all pinned upstream branch heads.

Not run:

* Real OpenTofu validation, because neither `tofu` nor `terraform` is installed locally.
* SwiftLint, because `swiftlint` is not installed locally.
* GitHub CI, because this repository has not yet been pushed.

## Unresolved Defects or Risks

* Licence classifications are documented as Stage 0 due-diligence records, not legal advice.
* No upstream source has been approved for import into product directories yet.
* The Swift scaffold is build-only; real XCTest/UI tests need an Xcode project and full Apple test toolchain.
* CI workflows are stubs and have not run on GitHub.
* No packet captures, gateway tests or transport evidence exists because Stage 1 has not started.

## Known Blockers

* `tofu`, `terraform` and `swiftlint` are not installed locally.
* SwiftPM `sandbox-exec` is blocked inside the Codex sandbox, requiring `--disable-sandbox` for local Stage 0 Swift package commands.

## Next Recommended Action

Push the initial Stage 0 bootstrap commit to the chosen remote, run GitHub CI or equivalent clean-checkout validation, record those results here, then request Stage 0 review.

If no remote has been chosen, configure the remote first and then push `main`.

Do not begin Stage 1 dual-path probe implementation until Stage 0 exit criteria are complete and reviewed.
