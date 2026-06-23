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

The userspace RNDIS **data plane is proven** (2026-06-23,
`research/usb-rndis-spike/rndis_dataplane.c`): claim → INITIALIZE → packet-filter
→ DHCP DISCOVER/OFFER round-trip in `REMOTE_NDIS_PACKET_MSG` frames, verified
live on the OnePlus 12R, no kext/SIP/root. Route B (RNDIS, works on any Android)
is confirmed by the owner over NCM. Next, in order:

* [x] Bring the RNDIS link up (`SET OID_GEN_CURRENT_PACKET_FILTER`) and prove a
  round-trip packet over the tether (DHCP DISCOVER out / OFFER in). Done.
* [ ] Hold the lease (DHCP REQUEST/ACK) and prove **real UDP egress** to the
  deployed oracle gateway through the RNDIS path — not just a local DHCP
  round-trip — confirming cellular reachability end-to-end (redact addresses).
* [ ] Present the link to the stack via `NEPacketTunnelProvider` so routing can
  bind a UDP socket to it (RX frames in, TX frames out).
* [ ] Re-confirm the userspace USB claim succeeds inside an App-Sandbox context
  with `com.apple.security.device.usb` (the spike ran un-sandboxed).

See skill `userspace-rndis-dataplane` for the build/run/safety loop and RNDIS
protocol reference.

### Detector gap found 2026-06-21 (fix alongside the above)

`darwin-evidence` cannot see a real Android tether for two confirmed reasons —
update evidence acquisition to match the RNDIS reality, not a NIC that never
appears:

* [x] The collector reads BSD interfaces, but an unclaimed RNDIS function is not
  a `enX` NIC. Added a USB-function evidence source
  (`USBFunctionDeviceSource`, `internal/platform/darwin/usb_functions.go`) that
  queries `ioreg -r -c IOUSBHostInterface -l` and keys on the RNDIS control
  signature (`bInterfaceClass=224`/subclass 1/protocol 3), **not** a vendor
  string. Verified live against the OnePlus on 2026-06-23. Also fixed a latent
  bug in the shared `splitIORegistryBlocks` parser: it used `bufio.Scanner`,
  which silently truncated at the 64 KiB line limit (full USB `ioreg -l` has
  ~90 KiB lines); it now splits on raw newlines.
  * [x] Design decision (resolved WITH owner, 2026-06-23): present-on-USB is a
    device-level signal, not a usable candidate. The report gained a
    `device_functions` channel (`usable:false`/`host_driver_claimed:false`) plus
    a `tether-present-not-usable` issue; a present-but-unusable function is never
    promoted to a `Candidate`. Owner framing: this is the data a **pre-purchase
    compatibility check** consumes ("is my Mac+Android combo supported?").
* [~] `blockHasAndroidUSBTetherEvidence` (`internal/platform/darwin/live_evidence.go`)
  requires the literal token `android`; the OnePlus identifies as
  `OnePlus`/`KALAMA`/`RNDIS …` and is missed. **Superseded for detection** by the
  class-keyed device source above. Left in place only for the
  host-driver-claimed (real `enX`) case; revisit when/if a host driver lands.

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

### Pre-purchase compatibility check (owner direction 2026-06-23)

Goal: minimal-friction onboarding — let a prospective macOS + Android user
confirm their combination is supported *before* they buy, and bundle whatever a
supported config needs into the installer. The `device_functions` channel in
`darwin-evidence` (added 2026-06-23) is the evidence foundation.

* NCM investigated live 2026-06-23 (OnePlus 12R, OxygenOS/Android 16) — result is
  two-sided (see memory `ncm-tether-lower-friction-than-rndis`, skill
  `android-usb-tether-function`):
  * [x] macOS claims an NCM tether natively — `svc usb setFunctions ncm` →
    `en10`, `ioreg` `CDC Network Control Model (NCM) … matched`, **no kext/SIP**.
  * [x] Confirmed the blocker: OxygenOS pins `mUsbTetheringFunction: RNDIS` in a
    **root-locked** overlay; the NCM function comes up as `usb0` with no NAT
    (macOS gets APIPA). No `device_config`/`cmd overlay`/settings lever exists.
* [ ] Define the supported matrix on this evidence: **NCM-default phones
  (Pixel/AOSP 14+)** → zero-install, works natively; **RNDIS-pinned phones
  (OnePlus & many OEMs)** → need the userspace RNDIS data plane or root. Confirm
  which Android builds default to NCM tethering.
* [ ] Build a user-facing preflight that maps `device_functions` + OS version to
  supported / not-yet-usable / unsupported, with a clear next step for each.
* [ ] Decide build-vs-bundle per supported config (driver/data-plane shipped in
  the installer) so a supported user needs no manual setup.

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
