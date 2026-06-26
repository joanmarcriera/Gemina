# Public repository strategy

Status: decision record and release runbook. Last updated 2026-06-23.

This document decides how Gemina VPN is published as open source and how the
owner takes it public. It is written for the moment the repository stops being
private. It assumes the dual licence already recorded in
[`docs/legal/licensing.md`](../legal/licensing.md) and the root
[`LICENSE`](../../LICENSE)/[`NOTICE`](../../NOTICE).

## Decision: one public monorepo (Option A)

**Publish a single public monorepo, keeping the dual licence in place via
per-directory licensing.** The AGPL-3.0-only gateway and the Apache-2.0
client/shared core live in one repository; the licence boundary is documented in
`docs/legal/licensing.md`, not enforced by a repository split.

This is the more honest option given the code as it stands, and it is the one
recommended here.

### Why A, on the evidence

The gateway is **not cleanly separable** today. The AGPL server directly imports
the Apache-2.0 shared core:

- `internal/gateway/dataplane.go` imports `gemina/pkg/clientcore`.
- `internal/gateway/server.go` imports `gemina/internal/protocol`,
  `gemina/internal/dedup` and `gemina/internal/metrics`.
- `cmd/gateway` imports `gemina/internal/protocol`.

This direction of dependency is *exactly what the licence intends*: Apache-2.0 is
one-way compatible into AGPL-3.0, so the AGPL gateway is allowed to include the
Apache core, and the combined gateway is then governed as a whole by the AGPL.
The client, by contrast, never imports a gateway package — verified, and the
property the App Store distribution story depends on.

Because the dependency only flows core → gateway, a single module expresses the
licence rule cleanly:

- Per-directory licence headers and `docs/legal/licensing.md` state which licence
  applies to which file.
- The one invariant that must hold — *the client never imports gateway code* — is
  a property of the import graph, checkable in CI, not something a repo split buys
  you.

A monorepo keeps one issue tracker, one CI matrix, one release tag stream, and no
cross-repo version coordination. For a pre-release project with a shared core
under active change, that is materially less overhead and less chance of the two
halves drifting.

### What Option B would have required (and why it is rejected for now)

Option B is two repositories — a client/Apache repo and a gateway/AGPL repo —
with the shared core extracted into a **third** standalone Go module that the
gateway depends on. To do that you would have to:

1. Extract a new module, e.g. `github.com/<org>/gemina-core`, containing the
   packages both sides share: `pkg/clientcore`, `pkg/protocoltypes`,
   `pkg/testkit`, `internal/protocol`, `internal/dedup`, `internal/metrics`, and
   any other shared `internal/*` the gateway reaches into. Shared `internal/`
   packages must become exported `pkg/` packages, because Go's `internal/` rule
   forbids importing them across module boundaries — a non-trivial API and
   import-path change across the whole tree.
2. Licence that shared module Apache-2.0, then have **both** the client repo and
   the AGPL gateway repo depend on it by tagged version.
3. Version it under semantic-import versioning and cut a tagged release of the
   core every time a shared type changes, then bump the dependency in two
   downstream repos and keep three changelogs in step.

That is a cleaner *paper* licence story — three physical repos matching three
licence zones — but it buys little the documented boundary does not already give,
while adding real release-coordination cost and a forced refactor of the shared
`internal/*` packages. It becomes worth doing only if a hard requirement appears:
for example, a third party who must vendor the core under Apache-2.0 without ever
touching AGPL files, or a legal preference for physical separation over a
documented one. **Until such a requirement is real, stay with A.**

If B is ever adopted, the package-to-repo map is:

| Destination repo / module            | Licence       | Packages |
| ------------------------------------ | ------------- | -------- |
| `gemina-core` (new shared module) | Apache-2.0    | `pkg/clientcore`, `pkg/protocoltypes`, `pkg/testkit`, and the shared logic currently in `internal/protocol`, `internal/dedup`, `internal/metrics` (re-exported as `pkg/…`) |
| `gemina-client`                  | Apache-2.0    | `apps/macos`, `bridge/`, `cmd/geminactl`, client-only `internal/*` (`paths`, `platform`, `diagnostics`, `transport`, `bootstrap`) |
| `gemina-gateway`                 | AGPL-3.0-only | `cmd/gateway`, `internal/gateway`, gateway-only `deploy/` assets |

The shared module would be versioned with annotated tags `vMAJOR.MINOR.PATCH`,
following semantic-import versioning, and both downstream repos pinned to a tag
(never a floating `main`).

## Taking it public — runbook

Placeholders: replace `<org>` with the GitHub owner (today `joanmarcriera`) and
`<repo>` with the repository name (today `gemina`). The existing remote is
`github.com/joanmarcriera/gemina`.

### 0. Pre-flight (do this first, every time)

```sh
# From the repository root:
scripts/prepare-public.sh
```

This is a read-only audit. It must print **GO** before you proceed. If it prints
**NO-GO**, fix what it lists and run it again. It never deletes anything.

Also run the project's own gates:

```sh
make test
make lint
make licence-check
scripts/docs-check.sh
```

### 1. Confirm the working tree is clean of non-product files

The audit script greps the **tracked** tree, but untracked junk can still be
swept in by a careless `git add -A`. Confirm nothing unwanted is staged:

```sh
git status --porcelain
git ls-files | grep -E '^(\.agents|\.codebuddy|\.continue|\.junie|\.kiro|\.codex)/' || echo "clean: no tool dirs tracked"
git ls-files | grep -E '(^|/)(gateway|geminactl)$' || echo "clean: no built binaries tracked"
```

`.gitignore` already excludes these, so a fresh clone is clean; the checks above
catch anything that slipped in before the ignore rules existed.

### 2. Make the existing repository public (recommended)

The history is already on `github.com/<org>/<repo>`. The simplest path is to flip
its visibility rather than create a new remote:

```sh
# Requires the GitHub CLI, authenticated as the repo owner.
gh repo view <org>/<repo>                          # confirm you are on the right repo
gh repo edit <org>/<repo> --visibility public --accept-visibility-change-consequences
```

Before flipping, audit the **history**, not just the current tree — a secret
removed in a later commit is still public in an earlier one:

```sh
# Search all of history for the junk dirs and obvious secrets:
git log --all --oneline -- '.agents/*' '.codebuddy/*' '.continue/*' '.junie/*' '.kiro/*' '.codex/*'
git rev-list --all --objects | grep -E '(^|/)(gateway|geminactl)$' || echo "no built binary ever committed"
```

If history is dirty, prefer publishing a **fresh repository from a squashed or
filtered tree** (next section) over trying to retro-clean a public history.

### 3. Alternative: publish a fresh public repository

Use this if the private history contains anything you do not want public.

```sh
# Create the public repo (empty) under the chosen org:
gh repo create <org>/<repo> --public \
  --description "Dual-uplink continuity VPN for macOS (open-core)" \
  --disable-wiki

# In a CLEAN checkout with the audit passing, set the new remote and push:
git remote add public git@github.com:<org>/<repo>.git
git push public main
```

To start history from scratch (drop all prior commits):

```sh
# In a throwaway clone, after scripts/prepare-public.sh prints GO:
git checkout --orphan public-main
git add -A
git commit -m "chore: initial public release"
git push public public-main:main
```

### 4. Branch protection and required checks

Protect `main` so the licence invariant and tests cannot be bypassed:

```sh
gh api -X PUT repos/<org>/<repo>/branches/main/protection \
  -H "Accept: application/vnd.github+json" \
  -f 'required_status_checks[strict]=true' \
  -f 'required_status_checks[checks][][context]=Go CI' \
  -f 'required_status_checks[checks][][context]=macOS CI' \
  -f 'required_status_checks[checks][][context]=Licence Scan' \
  -f 'enforce_admins=true' \
  -f 'required_pull_request_reviews[required_approving_review_count]=1' \
  -f 'restrictions=null'
```

The check contexts above match the existing workflow names under
`.github/workflows/` (`Go CI`, `macOS CI`, `Licence Scan`, `Infrastructure CI`,
`Release`). Adjust the list to the checks you want to be blocking.

CI notes:

- `Licence Scan` (`licence-scan.yml`) runs `scripts/licence-check.sh`, which fails
  if GPL text appears in product directories or if provenance files are missing.
  Keep it required.
- Add a CI step asserting the licence invariant directly from the import graph,
  so a future change cannot make the client import a gateway package:

  ```sh
  # Fails if any client/core package imports a gateway package.
  go list -deps ./cmd/geminactl ./pkg/... \
    | grep -E 'gemina/(cmd/gateway|internal/gateway)' \
    && { echo "client imports gateway code — licence boundary violated"; exit 1; } \
    || echo "ok: client is gateway-free"
  ```

- The redaction guard (`.claude/hooks/redaction-guard.sh`) is an edit-time hook,
  not CI. Mirror it in CI by failing on raw dotted-quad IPv4 or MAC addresses in
  tracked docs/source (excluding `*_test.go` and `testdata/`). `prepare-public.sh`
  performs the same check at release time.

### 5. Post-publish hygiene

- Turn on secret scanning and push protection in repository settings (or via
  `gh api -X PATCH repos/<org>/<repo> -f 'security_and_analysis[secret_scanning][status]=enabled'`).
- Set up the `Release` workflow's permissions and any `GHCR` token for the
  gateway image (`ghcr.io/<org>/gemina-gateway`), replacing the README's
  `ghcr.io/example/...` placeholder once a real image is published.
- Confirm `CODEOWNERS`, `SECURITY.md` and `CONTRIBUTING.md` read correctly for a
  public audience. Note `CONTRIBUTING.md` still says the licence is "being
  finalised" — update it to state the dual licence is final before going public.

## Pre-publish checklist

Run `scripts/prepare-public.sh` to mechanise the first block. Tick the rest by
hand.

Mechanised by the script:

- [ ] Run from the repository root; `.git` present.
- [ ] No tracked tool/scratch dirs: `.agents/`, `.codebuddy/`, `.continue/`,
      `.junie/`, `.kiro/`, `.codex/`.
- [ ] No tracked `skills-lock.json`.
- [ ] No tracked built binaries (`gateway`, `geminactl`).
- [ ] `LICENSE`, `NOTICE`, `LICENSES/AGPL-3.0.txt`, `LICENSES/Apache-2.0.txt`
      all present.
- [ ] No obvious secrets (private keys, AWS-style keys, bearer tokens) in the
      tracked tree.
- [ ] No raw dotted-quad IPv4 in tracked source/docs, excluding documentation
      placeholders, TEST-NET ranges and the verbatim licence texts.

By hand:

- [ ] `make test`, `make lint`, `make licence-check`, `scripts/docs-check.sh` pass
      on a clean checkout (`scripts/validate-clean-workspace.sh`).
- [ ] Client-imports-gateway invariant checked (CI step above).
- [ ] History audited if flipping an existing private repo to public (Step 2),
      or a fresh repo published instead (Step 3).
- [ ] `CONTRIBUTING.md` licence wording updated to final.
- [ ] README placeholders (`ghcr.io/example/...`, `gateway.example.com`) are
      intentional or replaced.
- [ ] `.research-src/` is absent or ignored (it is Git-ignored; never publish it).
- [ ] No large vendored binaries or datasets in the tree.

## What must never be published

- The third-party AI/agent scratch dirs: `.agents/`, `.codebuddy/`, `.continue/`,
  `.junie/`, `.kiro/`, `.codex/` — now in `.gitignore`.
- `skills-lock.json` — local tooling lockfile, not product.
- The built `gateway` binary and any `geminactl` binary — rebuild from source.
- `.research-src/` — pinned upstream clones for due diligence only; already
  ignored. Publishing it could redistribute upstream GPL implementation source.
- `.netcheck/` — raw local host network state.
- Anything under the secret patterns in `.gitignore` (`*.pem`, `*.key`,
  `*.mobileprovision`, `.env`, `*.tfstate`).
</content>
