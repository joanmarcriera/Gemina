# Project State

Last updated: 2026-06-19

## Current Objective

Stage 0 repository bootstrap and source due diligence.

This cycle completed a bounded Stage 0 review-request objective: create a review packet and open GitHub issues for engineering review and legal/provenance review without marking either review complete.

This cycle completed a bounded Stage 0 CI annotation-hardening objective: update CI action versions, record CI action/tool dependencies and verify GitHub CI still passes.

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

No VPN transport, packet framing, deduplication, NetworkExtension packet handling, gateway runtime, entitlement service, payment flow or real infrastructure resource exists.

The repository has a validating Stage 0 skeleton. The upstream manifest is fully pinned by shell-verified commits. Research sources are present only in `.research-src/`, which is ignored by Git.

The initial Stage 0 bootstrap is committed and pushed to GitHub. Stage 0 GitHub CI now passes on `origin/main`. No Stage 1 implementation exists.

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

* `.github/workflows/go-ci.yml`
* `.github/workflows/infra-ci.yml`
* `.github/workflows/licence-scan.yml`
* `.github/workflows/macos-ci.yml`
* `docs/legal/dependency-inventory.md`
* `docs/reviews/stage-0-review-request.md`
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

* `make clean-workspace-check`
* `scripts/docs-check.sh`
* `make licence-check`
* `ruby -e 'require "yaml"; ARGV.each { |path| YAML.load_file(path) }; puts "workflow yaml parsed"' .github/workflows/*.yml`
* `git diff --check`
* `git diff --cached --check`
* `git push origin main`
* `gh api repos/actions/checkout/releases/latest --jq '{tag_name,name,html_url,published_at}'`
* `gh api repos/actions/setup-go/releases/latest --jq '{tag_name,name,html_url,published_at}'`
* `gh api repos/opentofu/setup-opentofu/releases/latest --jq '{tag_name,name,html_url,published_at}'`
* `rg -n "actions/checkout@v4|actions/setup-go@v5|opentofu/setup-opentofu@v1|Node.js 24" .github/workflows`
  * Returned no matches.
* `rg -n "actions/checkout@v7.0.0|actions/setup-go@v6.4.0|opentofu/setup-opentofu@v2.0.1|cache: false" .github/workflows`
* `gh issue list --repo joanmarcriera/continuity-vpn --state open --limit 20`
* `gh label list --repo joanmarcriera/continuity-vpn --limit 100`
* `gh label create stage-0 --repo joanmarcriera/continuity-vpn --description "Stage 0 bootstrap, provenance and planning" --color 0e8a16`
* `gh label create stage-review --repo joanmarcriera/continuity-vpn --description "Stage gate review required" --color 5319e7`
* `gh label create legal-review --repo joanmarcriera/continuity-vpn --description "Legal or licence provenance review required" --color b60205`
* `gh issue create` for Stage 0 engineering review issue 1.
* `gh issue create` for Stage 0 legal/provenance review issue 2.
* `gh run watch 27820615456 --repo joanmarcriera/continuity-vpn --exit-status`
* `gh run watch 27820615438 --repo joanmarcriera/continuity-vpn --exit-status`
* `gh run watch 27820615455 --repo joanmarcriera/continuity-vpn --exit-status`
* `gh run watch 27820615442 --repo joanmarcriera/continuity-vpn --exit-status`
* `gh run view 27820615456 --repo joanmarcriera/continuity-vpn`
* `gh run view 27820615438 --repo joanmarcriera/continuity-vpn`
* `gh run view 27820615455 --repo joanmarcriera/continuity-vpn`
* `gh run view 27820615442 --repo joanmarcriera/continuity-vpn`
* GitHub Go CI passed on commit `4a8afd4`.
* GitHub Infrastructure CI passed on commit `4a8afd4`.
* GitHub Licence Scan passed on commit `4a8afd4`.
* GitHub macOS CI passed on commit `4a8afd4`.

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
* Stage 0 engineering review is requested in issue 1 but is not yet complete.
* Stage 0 legal/provenance review is requested in issue 2 but is not yet complete.
* The Swift scaffold is build-only; real XCTest/UI tests need an Xcode project and full Apple test toolchain.
* The first GitHub Actions push event produced `startup_failure` run `27815677467` with no jobs/logs. A later workflow-trigger hardening commit ran the named project CI workflows successfully, so this is no longer blocking clean-checkout validation evidence.
* Successful macOS CI still emits a non-blocking Homebrew tap-trust transition warning while installing SwiftLint from runner state.
* No packet captures, gateway tests or transport evidence exists because Stage 1 has not started.

## Known Blockers

* `tofu`, `terraform` and `swiftlint` are not installed locally.
* SwiftPM `sandbox-exec` is blocked inside the Codex sandbox, requiring `--disable-sandbox` for local Stage 0 Swift package commands.

## Next Recommended Action

Wait for Stage 0 engineering review issue 1 and legal/provenance review issue 2 to be completed, then record the outcome and any follow-up tasks before beginning Stage 1 transport work or source import.

Do not begin Stage 1 dual-path probe implementation until Stage 0 exit criteria are complete and reviewed.
