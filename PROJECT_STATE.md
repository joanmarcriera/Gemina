# Project State

Last updated: 2026-06-21

This is the durable cross-session handover. It records current state, risks,
blockers and the last validation run. The outstanding task list lives in
`TASKS.md`; architectural choices live in `DECISIONS.md`; the full change history
lives in Git. Keep this file lean — do not grow an append-only log here.

## Current Stage and Objective

Stage 1: dual-path UDP probe — **transport capability proven 2026-06-23**
(see "Definition of Dual-Path Success" below). Stage 0 exit criteria are met and
reviewed.

The probe proves packet identity, first-copy duplicate suppression, and
fixture/command-driven Darwin path classification via the redacted
`continuityctl darwin-evidence` diagnostic. Beyond that, the userspace spike now
proves the real thing: the same logical packet sent over Wi-Fi **and** the
phone's cellular link (independent WANs) both reach the deployed gateway, are
deduplicated to one delivery, and either path can drop without ending the
session. It does **not** yet prove encryption or any VPN behaviour, nor is the
dual-path transport wired through the shipping Swift app / NEPacketTunnelProvider.

Next action is recorded at the top of `TASKS.md`: wire the proven dual-path
transport into the product (NEPacketTunnelProvider) and add encryption; in
parallel, the open-core + hosted-gateway go-to-market work.

## Current Implementation State

Present and unit/race tested:

* `internal/protocol` — probe packet identity (`SessionID`, `PacketNumber`,
  `PacketID`).
* `internal/dedup` — bounded first-copy suppression backed by an O(1) ring
  buffer; benchmarked. This is a feasibility window, **not** production replay
  protection: it is in-memory, process-local, FIFO-evicted (an evicted ID seen
  again is re-accepted), and does not survive gateway restart.
* `internal/paths` — Wi-Fi / Android USB tethering candidate classification from
  typed observations, with no hard-coded BSD interface names.
* `internal/platform/darwin` — snapshot boundary, conservative live BSD
  collector (flags + IPv4 presence only), evidence-derived link kinds with a
  single shared evidence vocabulary, and provisional command-backed live evidence
  reduced from `networksetup`/`ioreg` to redacted tokens. Plus a **device-level**
  USB-function source (`USBFunctionDeviceSource`) that detects an Android RNDIS
  tether from `ioreg -r -c IOUSBHostInterface -l` by interface class
  (224/1/3 control), **not** a vendor string, so it sees the tether function
  before any host driver brings a NIC up. Verified against the live OnePlus
  (2026-06-23). Fixed a latent shared-parser bug along the way: the ioreg block
  splitter used `bufio.Scanner`, which silently truncated at its 64 KiB line
  limit — a full USB `ioreg -l` dump has ~90 KiB lines — so it now splits on raw
  newlines.
* `internal/diagnostics` + `cmd/continuityctl darwin-evidence` — redacted JSON
  Stage 1 evidence report labelled diagnostic-only-not-path-success. Now carries
  a `device_functions` channel (present-but-unusable tether functions, with
  `usable:false`/`host_driver_claimed:false`) plus a `tether-present-not-usable`
  issue, so the report honestly says "tether function present but not yet usable"
  instead of just "missing" — the signal a pre-purchase compatibility check
  consumes. A present-but-unusable function never becomes a usable `Candidate`.
* `internal/protocol` probe wire codec (`ProbePacket`, `PathTag`) — versioned
  fixed-size datagram carrying packet identity + coarse path tag, no host
  identifiers; round-trip and malformed-input tested.
* `internal/gateway` + `cmd/gateway` — UDP server that decodes probes, feeds the
  dedup window, and logs first-copy/duplicate/rejected per path as redacted JSON
  (source address never reaches the handler). Loopback-driven tests under race.
  **Deployed and running** on the `oracle` VPS as a distroless arm64 container
  under systemd (`continuity-gateway.service`); confirmed reachable end-to-end
  over the internet. See `docs/dev/gateway-deploy.md`.
* `internal/transport` `PathDialer` + `continuityctl probe` — binds a connected
  UDP socket to a chosen interface for egress via Darwin `IP_BOUND_IF`, so a
  socket leaves a specific path regardless of the default route. `probe` supports
  dual-path (`-interface2`/`-path2`): the same identity over two interfaces.
  Proven live: `en0`-bound reached the gateway, `lo0`-bound was correctly trapped
  ("network is unreachable"); and a dual-path run over `en0` (Wi-Fi) + `en4`
  (dock Ethernet) deduplicated correctly at the gateway. NB those two interfaces
  share the same upstream router, so this proves the *machinery*, not two
  independent WANs — the independent second WAN (phone 5G) still needs the
  userspace RNDIS host driver, which is not built (the tether has live RNDIS USB
  interfaces but no macOS NIC; confirmed 2026-06-21).
* Gateway logging/throughput: per-packet decisions log at Debug (guarded, so
  attribute strings are not built when disabled) with periodic Info summaries;
  the default Info hot path is ~30 ns/op and allocation-free (was ~447 ns/op
  logging every packet). The listener sets a 4 MiB read buffer for burst
  tolerance. Set `CONTINUITY_GATEWAY_LOG_LEVEL=debug` for per-packet detail.

Built this cycle (all unit/race tested):

* `pkg/clientcore` — the client/gateway transport core: frames tunnel packets
  with a per-session identity, encrypts the payload with identity-bound
  AES-256-GCM (ADR-0006; nonce from identity+direction, header authenticated, so
  encryption never disturbs the dual-path duplication), and deduplicates inbound
  so a logical packet duplicated over both paths is delivered once. Driven over
  the narrow Swift/Go boundary (ADR-0005); Swift seam sketched + builds.
* `internal/gateway` `DataPlane` — per-session responder state that decrypts +
  dedups CVD1 data packets, proven end-to-end in-process against a real client
  session. The encrypted real data path, server-side.
* `internal/entitlement` — hosted-tier scaffold: signed entitlement tokens, a
  payment-provider interface with a fake impl, and an Open/Hosted gate so the
  paid hosted gateway is gated while self-hosting stays free. No real keys yet.
* `pkg/clientcore` key agreement — X25519 ECDH + HKDF-SHA256 (ADR-0007), so the
  AEAD key is derived, not pre-shared (forward-secret), **with gateway
  authentication**: an Ed25519 gateway identity the client pins, signing the
  ephemeral key (defeats an active MITM; `handshake_auth.go`). The on-wire
  handshake message + pinned-key distribution remain.
* `internal/gateway` admission + handshake + real server — `SessionStore` +
  `Admitter` gate admission by entitlement (open=free, hosted=token, fail-closed).
  The CVH1 ClientHello/ServerHello exchange (`pkg/clientcore` +
  `Admitter.Handshake`) does mutual auth (client pins gateway Ed25519 identity;
  gateway admits the token) + X25519 key agreement; the ClientHello is timestamped
  and the gateway enforces a freshness window to bound replay. `DataGateway`
  ties it together: it demuxes ClientHello/CVD1 (`ClassifyDatagram`), terminates
  the handshake, decrypts+dedups the data plane, and exposes redacted data-path
  metrics (`continuity_handshakes_total`, `continuity_data_packets_total`,
  `continuity_active_sessions`); source addresses are used only to reply, never
  logged. End-to-end tested.
* `internal/entitlement` Stripe — `StripeProvider` (stdlib net/http, no SDK):
  checkout creation + real webhook signature verification (HMAC, constant-time,
  replay-tolerance). Drops into `entitlement.Service`. Real keys/accounts remain.
* `apps/macos` is now an **Xcode project** (XcodeGen, `project.yml`): a menu-bar
  app (`AppUI`, `MenuBarExtra`) + the `ContinuityTunnel` NetworkExtension packet
  tunnel + the logic frameworks, with the Go core linked as a cgo c-archive
  (bridging header + pre-build script). Phase 1 (menu-bar app) **ran** on the
  owner's Mac; Phase 2 (app + NE + Go core) is **headless-verified** (`xcodebuild
  … BUILD SUCCEEDED`). Blocked only on the **paid Apple Developer membership
  activating** (Network Extensions are unavailable to a free Personal Team) —
  waiting ~48h (resume ~2026-06-26). `project-dev.yml` builds a no-NE variant that
  runs on the Personal Team meanwhile. Build/sign recipe + gotchas in the
  `macos-app-xcode-build` skill. Next: Phase 3, the real `NEPacketTunnelProvider`
  (sockets, makeRelay, handshake over the wire, live status).
* `apps/macos` Swift↔Go link + app logic — `CContinuityCore` C target +
  `CoreTransport.swift` (cgo bridge), and `ContinuityVPNCore` (pure Swift):
  `PathPolicy` (Duplicate/Failover/Smart/Auto + preferred path; the relay honours
  it), `ProtectionStatus`, consent defaults (free opt-in / paid opt-out), and the
  `Impact` maths (outage absorbed, failovers survived). Verified headless by
  `ContinuityVPNCoreCheck` (no Xcode needed); `swift build` green. The SwiftUI
  views (menu bar, Settings, onboarding), the path sockets, and the
  handshake-over-the-wire remain Xcode-runtime work. App design spec:
  `docs/superpowers/specs/2026-06-24-macos-app-experience-design.md`.
* Monetisation study (`docs/product/monetisation-apple-study.md`): keep Stripe as
  the rail (a free macOS companion to a paid web service can skip App Store IAP);
  IAP, if ever used, is 15% (Small Business Program).
* `bridge/continuitycore` — the cgo C-shared bridge exposing the core to Swift
  over a handle-based ABI (ADR-0005); builds as a darwin/arm64 c-archive.
  `apps/macos` `ContinuityTunnelProvider` (guarded) wires packetFlow to the
  relay; builds in SwiftPM.
* `internal/metrics` + gateway `/metrics` — stdlib Prometheus registry; the
  gateway exposes `continuity_packets_total{decision,path}` (the failover signal)
  and `continuity_rejected_total{reason}`, redaction-enforced, on an opt-in
  metrics address. Grafana/alerts/scrape assets + `observability/METRICS.md`.
* Licence decided and applied: AGPL-3.0 gateway + Apache-2.0 client/core
  (`docs/legal/licensing.md`). Xcode/signing owner-action guide in
  `docs/dev/xcode-signing.md`.

Go-to-market groundwork: the static landing page (`website/`) is SEO-hardened
(OG/Twitter/JSON-LD, sitemap/robots, pricing) with draft privacy/terms pages;
`docs/marketing/` has the SEO strategy, video script, launch plan (with
ready-to-post Show HN / Product Hunt / social drafts) and a press kit;
`docs/legal/` has canonical privacy-policy and terms-of-service drafts (lawyer
review pending). A public-repository strategy (`docs/dev/repository-strategy.md`,
recommends a single open-core monorepo) with a read-only `scripts/prepare-public.sh`
GO/NO-GO audit.

Not implemented: the real `NEPacketTunnelProvider` (packetFlow I/O + path
sockets, linking the cgo bridge; needs full Xcode/signing), the on-wire handshake
*message* carrying the signed ephemeral key + pinned-identity distribution,
App-Sandbox re-confirmation of the USB claim, a real payment integration
(Stripe/IAP + accounts), client-side metrics (no app yet), production dedup
window sizing/rollover, real infrastructure resources, and direct
SystemConfiguration / Network framework / IORegistry API collection (only
command-backed collection exists). Before going public: rewrite git history to
drop a real LAN address (see the release audit).

Research sources are present only in the Git-ignored `.research-src/`. No upstream
implementation source has been imported into product directories.

Git remote: `origin` = `git@github.com:joanmarcriera/continuity-vpn.git`
(`https://github.com/joanmarcriera/continuity-vpn`).

## This Cycle's Changes

* Replaced `internal/dedup` slice-shift eviction (O(n) per insert at capacity)
  with an O(1) ring buffer; added a FIFO-order eviction test and a steady-state
  benchmark (~49 ns/op, 0 allocs/op at capacity 4096).
* Centralised the Darwin evidence key/value vocabulary into shared constants in
  `internal/platform/darwin/evidence.go`; producers and consumers now reference
  the same symbols, the duplicated Wi-Fi hardware-port helper is folded, and a
  new test pins producer↔consumer agreement.
* Consolidated task tracking: `TASKS.md` is now the single ordered source of
  truth; `docs/backlog/stage-1.md` points at it; this file was trimmed from an
  append-only log to a lean handover.

## Unresolved Defects or Risks

* The dedup window is a feasibility structure, not production replay protection
  (see above). Spec §8.2 requires a sequence-aware sliding window; the current
  structure will be replaced, not extended.
* Path classification depends on platform-provided link kinds. The Darwin
  boundary derives them from explicit evidence and command-backed sources only;
  direct SystemConfiguration / Network framework / IORegistry API collection is
  not implemented.
* The local diagnostic no longer reports the Android tether as simply *missing*
  when the phone is connected: as of 2026-06-23 a device-level USB-function
  source detects the RNDIS tether by interface class and the report shows it as
  `device_functions` + a `tether-present-not-usable` issue. The underlying
  limitation remains — macOS ships no RNDIS host driver, so the phone never
  becomes a usable `enX` NIC and there is still no usable `android-usb-tether`
  candidate; the report is now honest about *why*. The interface-level
  `blockHasAndroidUSBTetherEvidence` string matcher (vendor token `android`) is
  superseded by the class-keyed device source for detection but is left as-is for
  the host-driver-claimed case; see `TASKS.md`.
* In-app uplink acquisition is decided and de-risked (ADR 2026-06-21), and the
  RNDIS uplink is now **proven end-to-end to the deployed gateway over cellular**
  (2026-06-23). The userspace spike (`research/usb-rndis-spike/`) claims the
  phone's RNDIS interfaces, completes `REMOTE_NDIS_INITIALIZE`, sets the packet
  filter, holds a DHCP lease (DISCOVER→OFFER→REQUEST→ACK), ARP-resolves the
  phone's gateway, and sends real continuity probes (CVP1) in UDP/IP frames to
  the oracle gateway. Verified: the gateway logged first-copy/duplicate decisions
  tagged `android-usb-tether` (correct server-side dedup) and a host-side tcpdump
  saw the probes arrive from a cellular carrier public IP — so the phone is a
  real independent WAN reaching the gateway, all from an unprivileged process
  (no kext, no SIP, no root). This is the Route-B uplink and works on any Android
  (RNDIS is universal). Pure framing logic is unit-tested (`make test`).
  Remaining: feed RX/TX through an `NEPacketTunnelProvider`, App-Sandbox
  re-confirmation of the USB claim, then simultaneous Wi-Fi + cellular dual-path.
* Route decided 2026-06-23 (owner): pursue the RNDIS data plane, not NCM, for
  broad device support. NCM was investigated live and is two-sided — macOS claims
  an NCM tether natively (no kext; proven `en10`) but OxygenOS root-locks USB
  tethering to RNDIS, so NCM-with-cellular is device-dependent (Pixel/AOSP 14+
  work natively; OnePlus does not). See the `ncm-tether-lower-friction-than-rndis`
  memory and `android-usb-tether-function` skill.
* The gateway exists, is tested, and is deployed, but there is still no
  *client-side* per-path UDP egress and no packet capture proving each path
  independently reaches it — so dual-path success is NOT yet claimed.
* Remote reachability to the deployed gateway is confirmed end-to-end over the
  public internet (2026-06-21): probes from the Mac reach `oracle` and are
  deduplicated server-side, verified with a timestamped capture. The earlier VCN
  block was a source-port-range misconfiguration (set to the listen port instead
  of "All"); fixed. See `docs/dev/gateway-deploy.md`.
* Licence classifications are Stage 0 due-diligence records, not legal advice; no
  upstream source is approved for import.
* Pre-merge CI gating is unfinished: PR-triggered CI confirmation, branch
  protection / required checks on `main`, and SwiftLint pinning are outstanding
  (tracked in `TASKS.md`).
* The Swift scaffold is build-only; real XCTest/UI tests need an Xcode project
  and full Apple toolchain.

## Known Blockers

* `tofu`, `terraform` and `swiftlint` are not installed locally.
* SwiftPM `sandbox-exec` is blocked inside the Codex sandbox, requiring
  `--disable-sandbox` for local Swift package commands.

## Tests Run This Cycle

* `go vet ./...` — clean.
* `go test -race ./...` — passed (packages with logic: bootstrap, dedup,
  diagnostics, paths, platform/darwin, protocol).
* `go test -bench BenchmarkWindowObserveSteadyState -benchmem ./internal/dedup` —
  ~49 ns/op, 0 allocs/op.
* `gofmt -l internal/ cmd/` — no unformatted files.

## Definition of Dual-Path Success — ACHIEVED 2026-06-23

All five criteria are now met, with evidence (`research/usb-rndis-spike/`,
`rndis_dualpath.c`):

1. **Socket A leaves through Wi-Fi** — an OS UDP socket bound to `en0` via
   `IP_BOUND_IF`; the gateway-host capture saw its probes arrive from the home
   ISP public IP.
2. **Socket B leaves through Android USB tethering** — the userspace RNDIS
   uplink; its probes arrived from a cellular carrier public IP (no kext, no SIP,
   no root). The two paths showed up as two distinct public WAN sources at the
   gateway host.
3. **Both reach the same gateway process** — the deployed `oracle` gateway.
4. **One logical packet delivered once** — sending each identity over both paths,
   the gateway logged 11 first-copy + 5 duplicate decisions (correct dedup;
   first-copy split 8 wi-fi / 3 cellular, with 5 cellular copies deduplicated).
5. **Either path can disappear without ending the session** — a Wi-Fi-only phase
   and a cellular-only phase each delivered every identity via the surviving
   path.

Caveat: this is proven by the C spike (userspace RNDIS + bound socket), not yet
wired through the shipping Swift app / `NEPacketTunnelProvider`. The transport
*capability* is proven; product integration remains (see `TASKS.md`).
