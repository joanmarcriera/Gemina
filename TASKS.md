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

Build the userspace RNDIS data plane on top of the proven control handshake
(viability spike: `research/usb-rndis-spike/`, ADR 2026-06-21). The in-app
uplink-acquisition approach is decided; the control handshake is proven under
SIP. Next, in order:

* [ ] Re-confirm the userspace USB claim succeeds inside an App-Sandbox context
  with `com.apple.security.device.usb` (the spike ran un-sandboxed).
* [ ] Bring the RNDIS link up: `SET OID_GEN_CURRENT_PACKET_FILTER`, then DHCP
  over the bulk pipe to obtain the phone's address (redact it).
* [ ] Frame RNDIS data messages on the bulk IN/OUT endpoints; prove a round-trip
  packet over the tether with a capture.
* [ ] Present the link to the stack via `NEPacketTunnelProvider` so routing can
  bind a UDP socket to it.

### Detector gap found 2026-06-21 (fix alongside the above)

`darwin-evidence` cannot see a real Android tether for two confirmed reasons —
update evidence acquisition to match the RNDIS reality, not a NIC that never
appears:

* [ ] The collector reads BSD interfaces, but an unclaimed RNDIS function is not
  a `enX` NIC. Add a USB-function evidence source (RNDIS control class `0xE0` +
  data class `0x0A` present) so the Android uplink is detectable before the host
  driver brings a link up. Note (confirmed 2026-06-21 by reading
  `live_evidence.go`): `IORegistryCommandSource` queries
  `ioreg -r -c IOEthernetInterface`, which the unclaimed RNDIS function never
  publishes — the new source must query the USB layer (e.g.
  `ioreg -r -c IOUSBHostInterface -l`) and key on `bInterfaceClass`, not a
  vendor string.
  * Design decision needed (do WITH the owner — it reshapes the report
    contract): present-on-USB is **not** a usable candidate (no IP, no link yet),
    so the report needs an honest "tether function present but not yet usable"
    status / device-level evidence channel rather than a `Candidate`. Do not
    fabricate a usable `android-usb-tether` candidate from USB presence alone —
    that would violate the never-fake-path-success rule.
* [ ] `blockHasAndroidUSBTetherEvidence` (`internal/platform/darwin/live_evidence.go`)
  requires the literal token `android`; the OnePlus identifies as
  `OnePlus`/`KALAMA`/`RNDIS …` and is missed. Match on the RNDIS function
  signature, not the vendor string. Keep tokens coarse and redacted.

### Packaging & clean footprint (App Store target — ADR 2026-06-21)

Decided: ship via the Mac App Store, sandboxed, NE packet-tunnel as a bundled
app extension, no kernel/DriverKit/system extension. Contract in
`docs/product/footprint.md`. Open work:

* [ ] Confirm the userspace USB claim works under the App Sandbox with
  `com.apple.security.device.usb` (this gates the whole App Store choice; shares
  the spike re-confirmation task above).
* [ ] Decide the bundle identifier namespace for the app + extension.
* [ ] Implement the in-app "Remove configuration & uninstall" action
  (`removeFromPreferences()` + keychain cleanup) and the release-time footprint
  verification checklist.
* [ ] Scope the packet tunnel to included routes for the gateway only; never set
  it as the system default; exclude the management subnet.

### Test environment (cabled management channel — ADR 2026-06-21)

Protocol + scripts in `docs/dev/test-environment.md`. Standing practice:

* [ ] Before each test cycle: `scripts/snapshot-network.sh`, then
  `scripts/restore-network.sh` to pin the cabled service first. (Observed
  2026-06-21: the dock LAN was first in the service order, not the stable
  Thunderbolt Ethernet — pin the intended cable.)
* [ ] When the tunnel lands, add its disable step to `scripts/restore-network.sh`.

## Stage 1 — transport proof (the actual gate)

Overall proof, not complete until packet captures, gateway logs and path-loss
evidence exist:

* [~] Bind one UDP socket per path and prove per-interface egress. Mechanism
  done: `internal/transport` `PathDialer` binds a UDP socket to a named
  interface via Darwin `IP_BOUND_IF`/`IPV6_BOUND_IF`; `continuityctl probe`
  drives it. Proven live (2026-06-21): a socket bound to `en0` reached the
  deployed gateway, while the same bound to `lo0` returned "network is
  unreachable" — confirming egress is bound at the socket layer, not by source
  address. **Still needed for the full proof:** simultaneously bind socket A to
  Wi-Fi and socket B to the Android tether (needs the phone + cabled-management
  rig) and show both reach the gateway.
* [x] Send duplicated probes to one gateway process; deduplicate server-side.
  `internal/gateway` + `cmd/gateway` deployed to `oracle` as an arm64 container
  under systemd; on-host end-to-end test shows first-copy then duplicate
  (`first_path` preserved). Wire format: `internal/protocol` `ProbePacket`.
* [x] Open the Oracle Cloud VCN security list for ingress UDP 51820 and confirm
  the Mac reaches the deployed gateway over the internet (timestamped capture,
  2026-06-21). Gotcha recorded: ingress **source** port range must be "All", not
  the listen port. See `docs/dev/gateway-deploy.md`.
* [ ] Capture packet evidence showing each path independently reaches the gateway.
* [ ] Path-loss test: either path can disappear without ending the logical session.
* [x] Add a dedup **fuzz test** (`internal/dedup`): model-based
  `FuzzWindowObserveModel` checks Observe against an independent FIFO model and
  the ring invariant (`count == len(seen)`, bounded length) across arbitrary
  operation sequences. 16.7M execs / 30s with no failures; seed corpus runs
  under `-race` in the gate.
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
