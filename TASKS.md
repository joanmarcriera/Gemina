# Tasks

Last updated: 2026-06-21

This file is the single ordered source of truth for outstanding work. Durable
narrative history lives in `PROJECT_STATE.md` and Git; architectural choices live
in `DECISIONS.md`; the long-range epics live in
`docs/product/project-specification.md`. Do not re-list tasks in other files —
link here instead.

Current stage: **Stage 1 — dual-path UDP probe.** Stage 0 exit criteria are met
and reviewed.

## Next exact action

Run the redacted Darwin evidence diagnostic with Android USB tethering connected:

* [ ] Connect Android USB tethering on the target Mac.
* [ ] Run `go run ./cmd/continuityctl darwin-evidence`.
* [ ] Confirm the JSON reports one usable Wi-Fi candidate and one usable Android
  USB tethering candidate, with no source IP addresses, MAC addresses, serial
  numbers, raw access keys or raw IORegistry product strings.
* [ ] Record a redacted summary in `PROJECT_STATE.md` (do not commit raw local
  hardware output).
* [ ] If classification stays incomplete, refine evidence acquisition before
  socket binding.

## Stage 1 — transport proof (the actual gate)

Overall proof, not complete until packet captures, gateway logs and path-loss
evidence exist:

* [ ] Bind one UDP socket per path and prove per-interface egress (socket A via
  Wi-Fi, socket B via Android USB tethering).
* [ ] Send duplicated probes to one gateway process; deduplicate server-side.
* [ ] Capture packet evidence showing each path independently reaches the gateway.
* [ ] Path-loss test: either path can disappear without ending the logical session.
* [ ] Add a dedup **fuzz test** (`internal/dedup`); the benchmark now exists.
* [ ] Update the Stage 1 threat model and any ADRs if transport assumptions change.

## Code health (fold into the transport work, not separate effort)

* [x] Replace `dedup.Window` O(n) eviction with an O(1) ring buffer; add
  FIFO-order test and steady-state benchmark.
* [x] Centralise Darwin evidence key/value vocabulary into shared constants so
  producers and consumers cannot drift; fold the duplicated Wi-Fi helper.
* [ ] Note for the real sequence space: `protocol.PacketNumber == 0` is currently
  invalid, which interacts with the "safe on rollover" requirement.

## Stage 0 hardening — carry-over, complete before the first Stage 1 *merge*

* [ ] Add branch protection / required status checks on `main`.
* [ ] Confirm PR-triggered path-filtered CI runs (push-triggered path-filtered CI
  has passed on `fcd6238` and `4a8afd4`).
* [ ] Pin SwiftLint install in macOS CI instead of unpinned `brew install
  swiftlint`.
* [ ] Decide whether to generate the Xcode project now or stay SwiftPM-only until
  signing details are known; add real XCTest/UI tests once decided.
* [ ] Run OpenTofu validation and SwiftLint in an environment that has `tofu` and
  `swiftlint` installed.

## Legal / provenance standing conditions (block any upstream import)

From the Stage 0 reviewer comments (`docs/reviews/stage-0-review-comments.md`),
carried into Stage 1:

* [ ] Run a per-file/subtree licence scan of the specific path before importing
  ANY upstream file (root-file inspection is not import clearance).
* [ ] Author clean-room notes **before** writing any Engarde/OpenMPTCProuter
  (GPL) inspired dedup/transport code; the reader of GPL source must not author
  the corresponding core.
* [ ] Capture wireguard-go / wireguard-apple copyright notices in `NOTICE` and
  `docs/legal/code-provenance.md` at WireGuard reuse time.
* [ ] Honour MPL-2.0 per-file source-disclosure obligations if any
  terraform-provider-hcloud file is ever modified or vendored.
* [ ] Complete full legal review before any source import or distribution
  decision that depends on third-party source reuse.

## Completed

### Stage 0 exit (reviewed)

Repository structure, product spec, architecture overview, ADR framework
(ADR-0001..0004), legal/provenance templates, security threat-model template,
Makefile targets, clean-workspace check, Go workspace + skeletons, baseline
macOS scaffold, four CI workflows, pinned upstream manifest with shell-verified
commits, root licence inspection, atomic bootstrap commit, and both Stage 0
review gates (engineering issue 1, legal/provenance issue 2) — all complete. See
`PROJECT_STATE.md` and Git history for detail.

### Stage 1 so far

* [x] `internal/protocol` packet identity primitives + tests.
* [x] `internal/dedup` first-copy duplicate suppression (now ring-buffer backed)
  + unit/race tests + benchmark.
* [x] `internal/paths` fixture-driven Wi-Fi / Android USB tethering candidate
  classification without hard-coded interface names + tests.
* [x] `internal/platform/darwin` snapshot boundary, conservative live BSD
  collector, evidence-derived link kinds (shared constants), and command-backed
  live evidence from `networksetup` / `ioreg` reduced to redacted tokens + tests.
* [x] `continuityctl darwin-evidence` redacted JSON diagnostic + tests; run once
  locally (found Wi-Fi, correctly reported missing Android USB tethering).
* [x] Root stage markers moved from Stage 0 bootstrap to Stage 1 probe.
