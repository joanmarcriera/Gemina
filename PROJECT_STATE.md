# Project State

Last updated: 2026-06-19

## Current Objective

Stage 0 repository bootstrap and source due diligence.

This cycle completed a bounded GitHub remote objective: create the private GitHub repository, configure `origin`, push `main`, and inspect the first GitHub automation results.

This cycle completed a bounded Stage 0 CI objective: make the registered project CI workflows manually triggerable, push that workflow hardening, and verify GitHub clean-checkout CI succeeds.

## Completed Work

* Read `PROJECT_STATE.md`, `TASKS.md`, `DECISIONS.md` and `AGENTS.md`.
* Inspected Git status, recent commit and configured remotes.
* Confirmed `gh` is authenticated as `joanmarcriera`.
* Created the private GitHub repository `joanmarcriera/continuity-vpn`.
* Configured `origin` as `git@github.com:joanmarcriera/continuity-vpn.git`.
* Pushed `main` to `origin/main`.
* Confirmed GitHub workflows are present and active:
  * Go CI
  * Infrastructure CI
  * Licence Scan
  * macOS CI
  * Release
* Confirmed GitHub Dependency Graph completed successfully for the initial push.
* Inspected a GitHub Actions `startup_failure` run for the initial push; it had no jobs and no logs, so the initial push did not produce clean-checkout validation evidence.
* Delegated a bounded workflow-trigger review to `ollama_fast`; it agreed that adding `workflow_dispatch` to the four project CI workflows was a minimal Stage 0 fix and did not identify blocking workflow issues.
* Added `workflow_dispatch` to:
  * `.github/workflows/go-ci.yml`
  * `.github/workflows/infra-ci.yml`
  * `.github/workflows/licence-scan.yml`
  * `.github/workflows/macos-ci.yml`
* Verified the workflow YAML parses locally.
* Ran `make clean-workspace-check` after the workflow trigger change.
* Pushed commit `fcd6238` (`Allow manual Stage 0 CI runs`) to `origin/main`.
* Confirmed all four named project CI workflows ran and passed on GitHub for commit `fcd6238`:
  * Go CI run `27816489936`
  * Infrastructure CI run `27816489953`
  * Licence Scan run `27816489977`
  * macOS CI run `27816489933`

Prior completed Stage 0 work remains in place:

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
* `PROJECT_STATE.md`
* `TASKS.md`
* `DECISIONS.md`

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
* `ruby -e 'require "yaml"; ARGV.each { |path| YAML.load_file(path) }; puts "workflow yaml parsed"' .github/workflows/*.yml`
* `git diff --check`
* `git diff --cached --check`
* `git push origin main`
* `gh run watch 27816489936 --repo joanmarcriera/continuity-vpn --exit-status`
* `gh run watch 27816489953 --repo joanmarcriera/continuity-vpn --exit-status`
* `gh run watch 27816489977 --repo joanmarcriera/continuity-vpn --exit-status`
* `gh run watch 27816489933 --repo joanmarcriera/continuity-vpn --exit-status`
* `gh run view 27816489936 --repo joanmarcriera/continuity-vpn --json name,status,conclusion,event,headSha,createdAt,updatedAt,url,jobs`
* `gh run view 27816489953 --repo joanmarcriera/continuity-vpn --json name,status,conclusion,event,headSha,createdAt,updatedAt,url,jobs`
* `gh run view 27816489977 --repo joanmarcriera/continuity-vpn --json name,status,conclusion,event,headSha,createdAt,updatedAt,url,jobs`
* `gh run view 27816489933 --repo joanmarcriera/continuity-vpn --json name,status,conclusion,event,headSha,createdAt,updatedAt,url,jobs`
* GitHub Go CI passed on commit `fcd6238`.
* GitHub Infrastructure CI passed on commit `fcd6238`.
* GitHub Licence Scan passed on commit `fcd6238`.
* GitHub macOS CI passed on commit `fcd6238`.

Previously passed and still reflected in the repository state:

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
* The Swift scaffold is build-only; real XCTest/UI tests need an Xcode project and full Apple test toolchain.
* The first GitHub Actions push event produced `startup_failure` run `27815677467` with no jobs/logs. A later workflow-trigger hardening commit ran the named project CI workflows successfully, so this is no longer blocking clean-checkout validation evidence.
* Successful GitHub CI emitted non-blocking runner annotations:
  * `actions/checkout@v4` and `actions/setup-go@v5` are currently forced from Node.js 20 to Node.js 24 by GitHub.
  * Go CI reported a cache-restore warning because no `go.sum` exists yet.
  * macOS CI reported Homebrew tap-trust transition warnings while installing SwiftLint.
* No packet captures, gateway tests or transport evidence exists because Stage 1 has not started.

## Known Blockers

* `tofu`, `terraform` and `swiftlint` are not installed locally.
* SwiftPM `sandbox-exec` is blocked inside the Codex sandbox, requiring `--disable-sandbox` for local Stage 0 Swift package commands.

## Next Recommended Action

Request Stage 0 review and legal review of the recorded upstream licence/provenance material before any Stage 1 transport work or source import begins.

Do not begin Stage 1 dual-path probe implementation until Stage 0 exit criteria are complete and reviewed.
