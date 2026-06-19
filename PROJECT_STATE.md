# Project State

Last updated: 2026-06-19

## Current Objective

Stage 1 probe core: packet identity and first-copy duplicate suppression.

This cycle completed a bounded Stage 1 core objective: add unit-tested probe packet identity and in-memory first-copy duplicate suppression below the network layer, without implementing interface binding, gateway networking, encryption or source import.

## Completed Work

* Read `PROJECT_STATE.md`, `TASKS.md`, `DECISIONS.md` and `AGENTS.md`.
* Inspected Git status, recent commit and configured remotes.
* Delegated a bounded Stage 0 review-packet surface check to `ollama_deep`; it did not return before the local work completed and was closed without usable output.
* Added `docs/reviews/stage-0-review-request.md`.
* Pushed commit `6d4c432` (`Add Stage 0 review request`) to `origin/main`.
* Created GitHub labels:
  * `stage-0`
  * `stage-review`
  * `legal-review`
* Opened GitHub review issues:
  * Stage 0 engineering review: https://github.com/joanmarcriera/continuity-vpn/issues/1
  * Stage 0 legal/provenance review: https://github.com/joanmarcriera/continuity-vpn/issues/2
* Delegated a bounded GitHub Actions hardening review to `ollama_fast`; it agreed the patch was ready, though one suggested test command was intentionally ignored because it would have created a fake commit.
* Updated CI actions:
  * `actions/checkout` from `v4` to `v7.0.0`.
  * `actions/setup-go` from `v5` to `v6.4.0`.
  * `opentofu/setup-opentofu` from `v1` to `v2.0.1`.
* Disabled `actions/setup-go` caching while the module has no `go.sum`.
* Recorded GitHub Actions and CI binary tools in `docs/legal/dependency-inventory.md`.
* Confirmed the Node.js action runtime annotations and Go cache warning disappeared.
* Confirmed the remaining macOS CI annotation is the Homebrew tap-trust transition warning from runner state while installing SwiftLint.
* Pushed commit `4a8afd4` (`Harden Stage 0 CI action versions`) to `origin/main`.
* Confirmed all four named project CI workflows ran and passed on GitHub for commit `4a8afd4`:
  * Go CI run `27820615456`
  * Infrastructure CI run `27820615438`
  * Licence Scan run `27820615455`
  * macOS CI run `27820615442`
* Recorded independent Stage 0 reviewer comments in `docs/reviews/stage-0-review-comments.md`.
* Corrected the review packet so CI evidence points at push-triggered green runs on `fcd6238` and `4a8afd4`.
* Recorded non-blocking engineering follow-ups and legal/provenance standing conditions in `TASKS.md`.
* Stage 0 engineering review issue 1 approved Stage 1 engineering start with follow-ups.
* Stage 0 legal/provenance review issue 2 approved the Stage 0 records with standing import-time conditions. This is not legal advice and does not approve any source import.
* Posted the Stage 0 engineering review outcome to issue 1 and closed it: https://github.com/joanmarcriera/continuity-vpn/issues/1
* Posted the Stage 0 legal/provenance review outcome to issue 2 and closed it: https://github.com/joanmarcriera/continuity-vpn/issues/2
* Added `internal/protocol` packet identity primitives:
  * `SessionID`
  * `PacketNumber`
  * `PacketID`
* Added `internal/dedup` first-copy duplicate suppression:
  * invalid packet ID or empty path label -> `invalid`
  * first valid observation -> `first-copy`
  * later observation with the same `PacketID` -> `duplicate`
  * bounded in-memory eviction of oldest packet IDs
* Added unit tests for packet identity validation, first-copy acceptance, duplicate rejection, invalid observations, bounded eviction and concurrent duplicate calls.
* Ran the race detector for the new dedup/protocol packages.
* Documented the Stage 1 probe package boundary and evidence requirements in:
  * `docs/architecture/stage-1-probe.md`
  * `docs/testing/stage-1-probe-evidence.md`
  * `docs/security/stage-1-probe-threat-model.md`
* Delegated a bounded test-matrix request to `ollama_fast`; it returned unusable tool-call JSON, so source inspection and local tests remained authoritative.

Prior completed Stage 0 work remains in place:

* Created the private GitHub repository `joanmarcriera/continuity-vpn`.
* Configured `origin` as `git@github.com:joanmarcriera/continuity-vpn.git`.
* Pushed `main` to `origin/main`.
* Added `workflow_dispatch` to the four project CI workflows.
* Confirmed all four named project CI workflows ran and passed on GitHub for commit `fcd6238`:
  * Go CI run `27816489936`
  * Infrastructure CI run `27816489953`
  * Licence Scan run `27816489977`
  * macOS CI run `27816489933`
* Created the atomic initial Stage 0 bootstrap commit.
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
* Kept Stage 1 transport work deferred.

* Monorepo skeleton, root project controls, GitHub templates and CI stubs.
* ADR framework and ADR-0001 through ADR-0004.
* Architecture, security, legal, provenance and dependency-inventory documents.
* Go workspace and compile-only skeleton packages.
* SwiftPM macOS scaffold and manual Xcode creation instructions.
* Fully pinned upstream research manifest and exact-commit research fetch script.
* Root upstream licence-file inspection records.
* Workspace-local Go and Swift build caches for sandbox-compatible validation.

## Current Implementation State

No VPN transport, packet framing, production duplicate-suppression integration, NetworkExtension packet handling, gateway runtime, entitlement service, payment flow or real infrastructure resource exists.

The repository has a validating Stage 0 skeleton. The upstream manifest is fully pinned by shell-verified commits. Research sources are present only in `.research-src/`, which is ignored by Git.

The initial Stage 0 bootstrap is committed and pushed to GitHub. Stage 0 GitHub CI now passes on `origin/main`. The Stage 0 engineering and legal/provenance review gates are complete for starting Stage 1 probe work, subject to the recorded follow-ups and standing import-time conditions.

Stage 1 now has a unit-tested Go core for packet identity and first-copy duplicate suppression. It does not yet prove macOS interface discovery, per-interface UDP egress, gateway reachability, packet capture evidence, path loss survival, encryption or VPN behaviour.

Git remote:

* `origin`: `git@github.com:joanmarcriera/continuity-vpn.git`
* GitHub URL: `https://github.com/joanmarcriera/continuity-vpn`

Root upstream licence files have been inspected and recorded as Stage 0 engineering evidence, not legal advice or import approval.

The initial Xcode project is still intentionally not generated; `apps/macos/README.md` records the manual creation steps and the signing/entitlement decisions that must be reviewed first.

## Files Changed

Initial bootstrap content:

* Root project controls and validation files.
* `.github/` templates and workflows.
* `api/`, `apps/`, `bridge/`, `cmd/`, `db/`, `deploy/`, `internal/`, `observability/`, `pkg/`, `research/`, `scripts/` and `tests/`.
* `docs/adr/`, `docs/architecture/`, `docs/backlog/`, `docs/legal/`, `docs/operations/`, `docs/product/`, `docs/security/` and `docs/testing/`.
* Durable state files: `PROJECT_STATE.md`, `TASKS.md` and `DECISIONS.md`.

This cycle changed:

* `DECISIONS.md`
* `docs/architecture/overview.md`
* `docs/architecture/stage-1-probe.md`
* `docs/security/stage-1-probe-threat-model.md`
* `docs/testing/README.md`
* `docs/testing/stage-1-probe-evidence.md`
* `internal/dedup/doc.go`
* `internal/dedup/window.go`
* `internal/dedup/window_test.go`
* `internal/protocol/doc.go`
* `internal/protocol/identity.go`
* `internal/protocol/identity_test.go`
* `PROJECT_STATE.md`
* `TASKS.md`

Ignored local artefacts:

* `.build/`
* `apps/macos/.build/`
* `.research-src/`
* `.codex/`
* `.codex-cycle-started`

## Tests Run and Results

Passed in this cycle:

* `go test ./internal/protocol ./internal/dedup`
* `go test ./...`
* `go test -race ./internal/dedup ./internal/protocol`
* `make test`
  * Go tests passed for all packages.
  * SwiftPM build passed for the macOS scaffold.
  * Documentation structure check passed.
* `make lint`
  * Go formatting check passed.
  * Documentation structure check passed.
  * SwiftLint was not installed locally, so the local SwiftLint run was skipped.
* `scripts/docs-check.sh`
* `make licence-check`
* `git diff --check`
* `make clean-workspace-check`
  * Passed from a temporary copy; included docs checks, licence/provenance checks, Go tests and SwiftPM build.
* `git push origin main`
  * Pushed implementation commit `9081915` (`Add Stage 1 probe dedup core`).
* `gh run watch 27823267136 --repo joanmarcriera/continuity-vpn --exit-status`
  * GitHub Go CI passed on commit `9081915`.
* Fuzz testing considered.
  * Not added or run in this slice because no packet parser, serialiser or network input boundary exists yet; revisit when framing or gateway receive code exists.
* Integration testing considered.
  * Not applicable to this slice because no sockets, gateway receive loop or macOS path binding exists yet.

Previously passed and still reflected in the repository state:

* `make clean-workspace-check`
* `ruby -e 'require "yaml"; ARGV.each { |path| YAML.load_file(path) }; puts "workflow yaml parsed"' .github/workflows/*.yml`
* `gh run watch 27816489936 --repo joanmarcriera/continuity-vpn --exit-status`
* `gh run watch 27816489953 --repo joanmarcriera/continuity-vpn --exit-status`
* `gh run watch 27816489977 --repo joanmarcriera/continuity-vpn --exit-status`
* `gh run watch 27816489933 --repo joanmarcriera/continuity-vpn --exit-status`
* GitHub Go CI passed on commit `fcd6238`.
* GitHub Infrastructure CI passed on commit `fcd6238`.
* GitHub Licence Scan passed on commit `fcd6238`.
* GitHub macOS CI passed on commit `fcd6238`.
* `gh auth status`
* `gh repo create continuity-vpn --private --source . --remote origin --push`
* `gh run list --repo joanmarcriera/continuity-vpn --limit 20`
* `gh workflow list --repo joanmarcriera/continuity-vpn --all`
* `gh api repos/joanmarcriera/continuity-vpn/actions/runs/27815677467`
* `gh api repos/joanmarcriera/continuity-vpn/actions/runs/27815677467/jobs`
* `gh api repos/joanmarcriera/continuity-vpn/actions/workflows`
* `gh api repos/joanmarcriera/continuity-vpn/commits/ceb783cd777f3b6825622889d7243b19ab389c09/check-runs`
* `git remote -v`
* `git status --short --branch --ignored`
* `make clean-workspace-check`
* `make test`
* `make lint`
  * Go formatting and docs checks passed.
  * SwiftLint was not installed locally, so the local SwiftLint run was skipped.
* `make licence-check`
* `git diff --check`
* `git diff --cached --check`
* `git status --short --ignored`
* `make bootstrap`
* `make infra-check`
  * OpenTofu/Terraform was not installed locally, so local infra validation was skipped.
* `make fetch-research`
* `env GIT_CONFIG_GLOBAL=/dev/null git ls-remote ...` for all pinned upstream branch heads.

Not run:

* Local OpenTofu validation, because neither `tofu` nor `terraform` is installed locally.
* Local SwiftLint, because `swiftlint` is not installed locally.

## Unresolved Defects or Risks

* Licence classifications are documented as Stage 0 due-diligence records, not legal advice.
* No upstream source has been approved for import into product directories yet.
* Full legal review is still required before any source import or distribution decision that depends on third-party source reuse.
* PR-triggered CI still needs to be confirmed before Stage 1 merges; push-triggered path-filtered CI has passed on `fcd6238` and `4a8afd4`.
* Branch protection and required status checks on `main` still need to be configured before Stage 1 merges.
* SwiftLint installation is still unpinned in macOS CI and should be pinned before lint is treated as a release-quality gate.
* The Swift scaffold is build-only; real XCTest/UI tests need an Xcode project and full Apple test toolchain.
* The first GitHub Actions push event produced `startup_failure` run `27815677467` with no jobs/logs. A later workflow-trigger hardening commit ran the named project CI workflows successfully, so this is no longer blocking clean-checkout validation evidence.
* Successful macOS CI still emits a non-blocking Homebrew tap-trust transition warning while installing SwiftLint from runner state.
* The Stage 1 dedup window is in-memory and process-local; it is not production replay protection and does not survive gateway restart.
* No packet captures, gateway tests or transport evidence exists yet.

## Known Blockers

* `tofu`, `terraform` and `swiftlint` are not installed locally.
* SwiftPM `sandbox-exec` is blocked inside the Codex sandbox, requiring `--disable-sandbox` for local Stage 0 Swift package commands.

## Next Recommended Action

Begin the next Stage 1 slice: model macOS path candidates in `internal/paths` without hard-coded interface names, using fixture-driven tests before opening sockets.

Do not claim dual-path success until later work proves that UDP socket A explicitly leaves through Wi-Fi, UDP socket B explicitly leaves through Android USB tethering, both reach the same gateway process, one logical packet is delivered once, and either path can disappear without ending the logical session.
