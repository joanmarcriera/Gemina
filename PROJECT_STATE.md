# Project State

Last updated: 2026-06-19

## Current Objective

Stage 1 conservative Darwin interface-state collector.

This cycle completed a bounded Stage 1 path objective: add conservative Darwin
interface-state collection behind the existing `internal/platform/darwin`
snapshot boundary, without role inference from names, socket binding, gateway
networking, encryption or source import.

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
* Added `internal/paths` observation and classification primitives:
  * `LinkKind`
  * `Role`
  * `Observation`
  * `Candidate`
  * `Issue`
  * `Classification`
* Added fixture-driven path classification for usable Wi-Fi and Android USB tethering observations.
* Added unit tests for:
  * successful Wi-Fi and Android USB tethering candidate selection;
  * non-standard fake interface identifiers to guard against BSD-name assumptions;
  * unusable observations;
  * missing candidates;
  * ambiguous Wi-Fi candidates;
  * ambiguous Android USB tethering candidates;
  * unknown link kinds.
* Updated Stage 1 architecture, test evidence and threat-model docs to cover path-candidate classification.
* Delegated a bounded path-model review to `ollama_deep`; it did not return before local work and validation completed, so it was closed without usable output.
* Added `internal/platform/darwin` fixture-boundary primitives:
  * `EvidenceSource`
  * `Evidence`
  * `InterfaceSnapshot`
  * `ObservationsFromSnapshots`
  * `EvidenceBySource`
* Added fixture-driven Darwin tests proving:
  * snapshot fields map to `paths.Observation`;
  * snapshots can feed `paths.Classify`;
  * conventional BSD names such as `en0` and `en7` still require explicit `LinkKind`;
  * BSD names and display names alone do not assign link kind;
  * missing BSD names remain unusable;
  * evidence metadata can be filtered by source.
* Updated Stage 1 architecture, test evidence and threat-model docs to cover the Darwin observation boundary and planned macOS evidence sources.
* Touched `session-timestamp.log` as requested and read `SESSION-CONTEXT.md`.
* Delegated a bounded Darwin adapter boundary review to `ollama_fast`; it returned a useful test/doc checklist and the relevant items were incorporated.
* Read the tracked first-party Markdown corpus, the project-local Go/Bash skill
  Markdown files and the current project state files before editing.
* Added `internal/platform/darwin` conservative live collection primitives:
  * `InterfaceSource`
  * `InterfaceRecord`
  * `NetInterfaceSource`
  * `LiveInterfaceSnapshots`
  * `CollectInterfaceSnapshots`
  * `EvidenceSourceBSDNetworkState`
* Added collector tests proving:
  * BSD flags and IPv4 presence map to `InterfaceSnapshot`;
  * explicit link kind evidence is preserved when injected;
  * BSD names and display names still do not classify Wi-Fi or Android USB tethering;
  * IPv6-only interfaces remain unusable for the IPv4-only Stage 1 probe;
  * nil sources and source errors are handled explicitly.
* Updated root stage documentation from Stage 0 bootstrap to Stage 1 probe while
  preserving the later-stage gates.
* Updated the Go and Swift stage markers from `stage-0-bootstrap` to
  `stage-1-probe`.
* Updated architecture, testing and threat-model docs for the conservative
  Darwin collector.
* Delegated a bounded collector review to `ollama_fast`; it returned generic Go
  testing guidance rather than patch findings, so it was not used as evidence.
* Cleaned up remaining first-party Stage 0 placeholder wording outside
  historical review/spec/state records so future sessions see Stage 1 as the
  active stage.
* Created and pushed commit `a0aa2b4` (`Add Darwin interface state collector`).
* Confirmed GitHub CI passed on commit `a0aa2b4`:
  * Go CI run `27836988541`
  * Infrastructure CI run `27836988540`
  * macOS CI run `27836988568`
* Confirmed the only macOS CI annotation remains the non-blocking Homebrew
  tap-trust transition warning while installing SwiftLint.

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

The repository has a completed validating Stage 0 baseline plus bounded Stage 1
probe packages. The upstream manifest is fully pinned by shell-verified commits.
Research sources are present only in `.research-src/`, which is ignored by Git.

The initial Stage 0 bootstrap is committed and pushed to GitHub. Stage 0 GitHub CI now passes on `origin/main`. The Stage 0 engineering and legal/provenance review gates are complete for starting Stage 1 probe work, subject to the recorded follow-ups and standing import-time conditions.

Stage 1 now has a unit-tested Go core for packet identity, first-copy duplicate
suppression, fixture-driven path-candidate classification, a Darwin snapshot
boundary and a conservative live collector for BSD interface flags and IPv4
presence. It does not yet prove Wi-Fi or Android USB tethering classification
from live macOS evidence, per-interface UDP egress, gateway reachability, packet
capture evidence, path loss survival, encryption or VPN behaviour.

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
* `PROJECT_STATE.md`
* `TASKS.md`
* `AGENTS.md`
* `CONTRIBUTING.md`
* `NOTICE`
* `README.md`
* `SECURITY.md`
* `CODEOWNERS`
* `api/openapi.yaml`
* `api/protocol/framing.md`
* `api/protocol/messages.md`
* `apps/macos/README.md`
* `apps/macos/Shared/ProductStage.swift`
* `apps/macos/UnitTests/README.md`
* `bridge/README.md`
* `bridge/darwin/README.md`
* `deploy/ansible/README.md`
* `deploy/tofu/environments/dev/main.tf`
* `deploy/tofu/environments/production/main.tf`
* `deploy/tofu/modules/README.md`
* `docs/architecture/overview.md`
* `docs/architecture/stage-1-probe.md`
* `docs/security/stage-1-probe-threat-model.md`
* `docs/testing/stage-1-probe-evidence.md`
* `internal/bootstrap/stage.go`
* `internal/bootstrap/stage_test.go`
* `internal/entitlement/doc.go`
* `internal/framing/doc.go`
* `internal/gateway/doc.go`
* `internal/logging/doc.go`
* `internal/metrics/doc.go`
* `internal/platform/darwin/collector.go`
* `internal/platform/darwin/collector_test.go`
* `internal/platform/darwin/observations.go`
* `internal/platform/darwin/observations_test.go`
* `internal/platform/linux/doc.go`
* `internal/replay/doc.go`
* `internal/sessions/doc.go`
* `internal/transport/doc.go`
* `pkg/clientcore/doc.go`
* `pkg/protocoltypes/doc.go`
* `pkg/testkit/doc.go`
* `research/experiments/README.md`
* `scripts/build-apple-bridge.sh`
* `scripts/collect-diagnostics.sh`
* `scripts/deploy-dev-gateway.sh`
* `scripts/run-network-tests.sh`

Ignored local artefacts:

* `.build/`
* `apps/macos/.build/`
* `.research-src/`
* `.codex/`
* `.codex-cycle-started`

## Tests Run and Results

Passed in this cycle:

* `go test ./internal/platform/darwin ./internal/paths`
* `go test ./internal/platform/darwin ./internal/paths ./internal/bootstrap`
* `go test -race ./internal/platform/darwin ./internal/paths`
* `scripts/docs-check.sh`
* `go test ./...`
* `go test -race ./...`
* `make test`
  * Go tests passed for all packages.
  * SwiftPM build passed for the macOS scaffold.
  * Documentation structure check passed.
* `make lint`
  * Go formatting check passed.
  * Documentation structure check passed.
  * SwiftLint was not installed locally, so the local SwiftLint run was skipped.
* `make licence-check`
* `make infra-check`
  * OpenTofu/Terraform was not installed locally, so local infra validation was
    skipped.
* `git diff --check`
* `make clean-workspace-check`
  * Passed from a temporary copy; included docs checks, licence/provenance
    checks, Go tests, SwiftPM build and local lint checks. SwiftLint was not
    installed locally, so local SwiftLint was skipped.
* `git push origin main`
  * Pushed implementation commit `a0aa2b4` (`Add Darwin interface state collector`).
* `gh run watch 27836988541 --repo joanmarcriera/continuity-vpn --exit-status`
  * GitHub Go CI passed on commit `a0aa2b4`.
* `gh run watch 27836988540 --repo joanmarcriera/continuity-vpn --exit-status`
  * GitHub Infrastructure CI passed on commit `a0aa2b4`.
* `gh run watch 27836988568 --repo joanmarcriera/continuity-vpn --exit-status`
  * GitHub macOS CI passed on commit `a0aa2b4` with the known non-blocking
    Homebrew tap-trust annotation.
* Fuzz testing considered.
  * Not added or run in this slice because the collector maps typed interface
    records and CIDR strings from the standard library rather than parsing a
    custom packet or token format. Revisit when live command output, protocol
    framing or entitlement parsing exists.
* Integration testing considered.
  * Not applicable to this slice because no socket binding, gateway receive loop
    or macOS path egress exists yet.

Previously passed in the prior cycle:

* `go test ./internal/platform/darwin ./internal/paths`
* `go test ./...`
* `go test -race ./internal/platform/darwin ./internal/paths`
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
  * Pushed implementation commit `63bd192` (`Add Darwin path observation boundary`).
* `gh run watch 27836030241 --repo joanmarcriera/continuity-vpn --exit-status`
  * GitHub Go CI passed on commit `63bd192`.
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
* The Stage 1 path classifier depends on platform-provided link kinds; the
  conservative Darwin collector now records BSD interface state but does not yet
  populate Wi-Fi or Android USB tethering link kinds from SystemConfiguration,
  Network framework or IORegistry evidence.
* No packet captures, gateway tests or transport evidence exists yet.

## Known Blockers

* `tofu`, `terraform` and `swiftlint` are not installed locally.
* SwiftPM `sandbox-exec` is blocked inside the Codex sandbox, requiring `--disable-sandbox` for local Stage 0 Swift package commands.

## Next Recommended Action

Begin the next Stage 1 slice: add explicit Wi-Fi and Android USB tethering
evidence sources behind the Darwin snapshot boundary. Prefer redacted fixtures
or fixture-driven adapters for SystemConfiguration/Network framework Wi-Fi
evidence and IORegistry USB association evidence before attempting socket
binding.

Do not claim dual-path success until later work proves that UDP socket A explicitly leaves through Wi-Fi, UDP socket B explicitly leaves through Android USB tethering, both reach the same gateway process, one logical packet is delivered once, and either path can disappear without ending the logical session.
