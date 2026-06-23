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

**Stage-1 dual-path transport is PROVEN** (2026-06-23): the userspace spike
sends one identity over Wi-Fi (IP_BOUND_IF en0) and the phone's cellular RNDIS
uplink simultaneously; the gateway dedups to one delivery, the host capture saw
two independent public WANs, and path-loss phases survive either path dropping.
All five "Definition of Dual-Path Success" criteria met (see `PROJECT_STATE.md`).
The transport *capability* is done; the work now splits into **productisation**
and **go-to-market** (open-core + hosted gateway). Next, in priority order:

* [x] Add encryption/framing for real traffic (2026-06-23). Identity-bound
  AES-256-GCM in `pkg/clientcore` (ADR-0006), and `internal/gateway.DataPlane`
  decrypts + dedups server-side, proven end-to-end in-process. No invented crypto
  (stdlib AEAD); key agreement (handshake) left as future work.
* [~] Wire the proven transport into the shipping app via
  `NEPacketTunnelProvider`. Transport brain + encryption + dedup + key agreement
  (X25519/HKDF, ADR-0007) built and tested; the **cgo bridge**
  (`bridge/continuitycore`, c-archive) and a guarded `ContinuityTunnelProvider`
  skeleton exist. **Still needed (owner/Xcode-gated, `docs/dev/xcode-signing.md`):**
  the real provider linking the bridge with packetFlow I/O + path sockets,
  authenticating the handshake keys + their exchange, and batching.
* [x] Monitoring/observability (2026-06-23): stdlib Prometheus `/metrics` on the
  gateway (failover signal `continuity_packets_total{decision,path}` +
  `continuity_rejected_total{reason}`, redaction-enforced), plus Grafana/alerts/
  scrape assets and `observability/METRICS.md`. Design in
  `docs/superpowers/specs/2026-06-23-monitoring-design.md`. Client-side metrics
  defined for when the app lands.
* [x] Authenticate the handshake against an active MITM (2026-06-24): Ed25519
  gateway identity + client pinning, gateway signs its ephemeral key
  (`pkg/clientcore/handshake_auth.go`, ADR-0007). On-wire message + pinned-key
  distribution remain.
* [x] Wire entitlement into gateway admission (2026-06-24):
  `SessionStore`/`Admitter` gate sessions by token (open=free, hosted=token,
  fail-closed); the DataPlane only serves admitted sessions.
* [ ] Re-confirm the userspace USB claim succeeds inside an App-Sandbox context
  with `com.apple.security.device.usb` (the spike ran un-sandboxed). Gates the
  App Store route.
* [ ] Going public: rewrite git history to drop the real LAN address, finalise
  CONTRIBUTING licence wording, then run `scripts/prepare-public.sh` (see
  `docs/dev/repository-strategy.md`). Marketing groundwork done
  (`docs/marketing/seo.md`, `docs/marketing/video-script.md`).
* [~] Go-to-market: licence decided + applied (AGPL gateway / Apache client),
  open-core README + landing page done; hosted-tier entitlement/payments
  scaffolded (`internal/entitlement`). Remaining: real payment integration
  (Stripe/IAP + accounts) and making the repo public.

Proven foundations (done): RNDIS control handshake, data plane (DHCP round-trip),
real UDP egress to the gateway over cellular, and simultaneous dual-path. Drivers
+ unit tests in `research/usb-rndis-spike/`; see skill
`userspace-rndis-dataplane`.

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

### Product model: open-core + hosted gateway (owner direction 2026-06-23)

FOSS client + gateway; revenue from a **paid hosted gateway** (our Oracle box)
for users who don't want to self-host. See memory
`product-model-open-core-hosted-gateway`. Open work:

* [ ] Keep the gateway address **configurable** end-to-end (self-host vs hosted);
  never hard-code it. The dual-path client and the RNDIS egress already take it
  as config — hold this line through the app UI and the NE tunnel.
* [ ] Make the gateway trivially self-hostable: document the single-container
  deploy + the one port (UDP 51820); a `docker run` quickstart in the README.
* [x] Decide + apply licences (2026-06-23): AGPL-3.0 gateway + Apache-2.0
  client/core (`docs/legal/licensing.md`) — stops a hosted-tier reseller while
  keeping the App-Store-bound client permissive. Per-file SPDX headers to follow.
* [~] Hosted tier entitlement/payment path scaffolded (`internal/entitlement`):
  signed tokens + Open/Hosted gate so gating lives at the hosted gateway, not by
  crippling the OSS client. Remaining: real payment integration (Stripe/IAP) +
  accounts, and wiring the gate into the gateway admission path.
* [~] Public GitHub repo prep: open-core README + self-host quickstart, dual
  LICENSE/NOTICE, landing page (`website/`) done. Remaining: make the repo
  public and route the landing page through `continuityctl preflight`.

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
* [x] Build a user-facing preflight that maps `device_functions` + OS version to
  a verdict + next step (2026-06-23). `continuityctl preflight`
  (`internal/diagnostics/compatibility.go`) returns supported / needs-android /
  needs-wifi / needs-both / unsupported-macos with a plain next step; default
  one-line summary, `-json` for the app/website. Key call: an RNDIS function
  present = **supported** (the app's userspace driver handles any Android),
  native NCM = supported without the driver. Table-driven tests cover the matrix;
  verified live ("supported", app-driver-rndis, macOS 26.5).
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

* [x] Bind one UDP socket per path and prove per-interface egress, then prove
  **simultaneous** Wi-Fi + cellular dual-path (2026-06-23). Go mechanism:
  `internal/transport` `PathDialer` + `continuityctl probe` (IP_BOUND_IF). Full
  real proof via the userspace spike `rndis_dualpath.c`: the same identity left
  the Mac over Wi-Fi (IP_BOUND_IF en0) and over the phone's cellular RNDIS uplink
  at once; the gateway-host capture saw two distinct public WAN sources (home ISP
  + cellular carrier) and the gateway deduplicated to one delivery each.
* [x] Send duplicated probes to one gateway process; deduplicate server-side.
  `internal/gateway` + `cmd/gateway` deployed to `oracle` as an arm64 container
  under systemd; on-host end-to-end test shows first-copy then duplicate
  (`first_path` preserved). Wire format: `internal/protocol` `ProbePacket`.
* [x] Open the Oracle Cloud VCN security list for ingress UDP 51820 and confirm
  the Mac reaches the deployed gateway over the internet (timestamped capture,
  2026-06-21). Gotcha recorded: ingress **source** port range must be "All", not
  the listen port. See `docs/dev/gateway-deploy.md`.
* [x] Capture packet evidence showing each path independently reaches the gateway
  (2026-06-23): gateway-host tcpdump saw two distinct public WAN sources — home
  ISP (Wi-Fi) and cellular carrier (RNDIS) — for one session.
* [x] Path-loss test: either path can disappear without ending the logical
  session (2026-06-23). `rndis_dualpath.c` phases 2/3 sent over one path only
  (Wi-Fi-only, then cellular-only); the gateway received every identity via the
  surviving path.
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
