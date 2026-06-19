# Stage 0 Review Comments

Reviewer: independent technical/provenance review (separate from the implementation session)
Date: 2026-06-19
Evidence inspected at working tree; git-level checks run locally to verify doc claims.

---

## Issue #1 — Engineering Stage 0 Review

**Decision: Approved for Stage 1 engineering start** (with non-blocking follow-ups below).

### Verified independently (not just trusting the docs)

- `.research-src/` is git-ignored and **no** research-source files are tracked (`git ls-files` clean; `git check-ignore` confirms).
- No upstream source leaked into product directories: no tracked files under `apps/ cmd/ internal/ pkg/ bridge/ api/ db/ deploy/ observability/ tests/` reference the upstream projects.
- `research/upstream-manifest.yaml`: all 11 upstreams pinned to valid 40-hex commits.
- Go/Swift skeletons are genuine stubs — `internal/*` are `doc.go` package declarations, `cmd/*` print a bootstrap stage string. No transport, dedup or framing behaviour is implemented or implied.
- `scripts/licence-check.sh` is a real gate: fails on GPL text in product dirs, missing legal files, or invalid/pending manifest pins. Run in CI (Licence Scan).
- ADR-0001 through ADR-0004 present; ADR-0001 records continuity-first and explicitly bars aggregation/zero-loss marketing claims.
- macOS scaffold is SwiftPM-only by design; `apps/macos/README.md` records manual Xcode/entitlement steps deferred to owner review (ADR-aligned).

### Answers to the specific review questions

- Repo controls sufficient to start Stage 1: **yes**.
- Stage 1 correctly scoped to a dual-path UDP probe, not a VPN: **yes** (`docs/backlog/stage-1.md` is probe/evidence-only).
- SwiftPM-only scaffold acceptable until signing/entitlements known: **yes**.
- Skeletons do not imply unproven transport behaviour: **confirmed**.
- CI sufficient as clean-checkout validation: **yes, with the caveat below**.

### Non-blocking follow-ups (record in TASKS.md, do not block Stage 1)

1. **CI trigger hardening.** The first repository push produced `startup_failure` (run `27815677467`). Push-triggered path-filtered CI subsequently passed on `fcd6238` and again on `4a8afd4` after action-version hardening. Before Stage 1 merges, confirm the PR path-filtered triggers actually run, and add branch protection / required status checks on `main`.
2. **Doc commit drift.** `stage-0-review-request.md` originally cited baseline `1e3b461` and CI evidence `fcd6238`, while `PROJECT_STATE.md` records newer passing CI on `4a8afd4`. Point reviewers at the latest green commit so the evidence trail is unambiguous.
3. **`release.yml` exists but is not listed** among engineering records in the review request. It is a disabled placeholder (correct for Stage 0) — just add it to the inventory for completeness.
4. **SwiftLint installed via unpinned `brew install`** in macOS CI (already tracked). Pin it before relying on lint as a gate.
5. **No real Swift XCTest/UI tests yet** (already a known hardening task tied to Xcode project creation).

None of the above changes the gate decision. Recommend recording items 1–5 as Stage 0 hardening tasks, then proceeding to Stage 1.

---

## Issue #2 — Legal / Provenance Review

**Decision: Stage 0 licence/provenance records are sufficient for Stage 0. Approved**, subject to the standing import-time conditions below. (This is a provenance-records review, not legal advice; not a substitute for counsel before any import or distribution.)

### Confirmations requested by the review

- **No upstream implementation source imported.** `docs/legal/code-provenance.md` "Current Imports: none" — verified against git: no upstream source is tracked anywhere in the tree. `NOTICE` states the same accurately.
- **GPL projects stay inspiration-only.** Engarde (GPL-2.0) and OpenMPTCProuter (GPL-3.0) are marked `inspiration-only` in `upstream-licences.md` and carry explicit `prohibited_use` in the manifest (`copy-into-proprietary-core`, `line-by-line-translation`, `llm-rewrite-of-source-files`) plus `clean_room_notes_required: true`. Consistent and correct given the proprietary "All rights reserved" LICENSE.
- **Root licence classifications adequate for Stage 0.** All 11 entries are honestly scoped as "root licence file observed/reviewed" and flagged not-legal-advice. Since no code was imported, root-level classification is enough to clear the Stage 0 gate.

### Conditions to carry into Stage 1 (record, not blockers)

1. **Root-file-only inspection ≠ import clearance.** Only each repo's root LICENSE was read. Per-file headers, multi-licensed subtrees and bundled third-party vendor dirs are not yet audited. Before importing *any* file, do a per-file/subtree licence scan of the specific path. The manifest already encodes `legal_review_required: before-import` — good; keep it enforced.
2. **Clean-room discipline for GPL inspiration must be behavioural, not just documented.** Because the product is proprietary, GPL contamination is the central risk. Recommend an explicit rule: whoever reads Engarde/OpenMPTCProuter source does not author the corresponding dedup/transport core, and Stage 1 "inspiration" produces the required clean-room notes before code is written.
3. **WireGuard reuse needs notice handling at import time.** wireguard-go / wireguard-apple are MIT at root but ship specific copyright notices; if reused, `NOTICE` and `code-provenance.md` must capture them before the file lands in a product dir.
4. **MPL-2.0 (terraform-provider-hcloud) is file-level copyleft.** Fine to *use* as a provider; if any MPL-covered file is ever modified/vendored, per-file source disclosure obligations apply — flag at import time.

### Bottom line

No import has occurred, the inspiration-only boundary is documented and enforced by `licence-check.sh`, and the records are internally consistent. **Approve Stage 0 legal/provenance.** Carry conditions 1–4 forward as standing import-time requirements.
