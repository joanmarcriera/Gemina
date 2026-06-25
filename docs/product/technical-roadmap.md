# Technical Roadmap to Public Release

Last updated: 2026-06-25

This roadmap sequences the work from the current state to a deliberately-limited
public release. It complements the authoritative
[`project-specification.md`](project-specification.md) (long-range epics),
`DECISIONS.md` / `docs/adr/` (locked architectural choices) and `TASKS.md` (the single
ordered source of truth for outstanding work). It does not re-list live tasks — `TASKS.md`
and GitHub issues #3–#10 own those; this document owns sequencing, parallelism and risk.

> Note on baseline: the original specification's linear Stage 0–9 walk has been partly
> overtaken by reality. Stage 1 dual-path transport is **proven**, much of Stage 2 has
> **landed**, and the commercial model has evolved to **open-core + paid hosted gateway**
> (see below). This roadmap re-baselines the forward plan on what is actually left, and
> maps each phase to the existing issues rather than re-deriving the spec stages.

## Where the project actually is (2026-06-25)

**Proven / landed** (all unit/race tested unless noted):

- **Stage 1 dual-path transport is PROVEN on hardware (2026-06-23).** A userspace spike
  (`research/usb-rndis-spike/`) sent one packet identity over Wi-Fi (`IP_BOUND_IF en0`)
  **and** the phone's cellular RNDIS uplink simultaneously; the gateway-host capture saw
  two independent public WANs (home ISP + cellular carrier); the gateway deduplicated to
  one delivery; and either path could drop without ending the session. All five
  "Definition of Dual-Path Success" criteria are met.
- **Encrypted, authenticated data plane.** Identity-bound AES-256-GCM (ADR-0006); X25519
  ECDH + HKDF-SHA256 session keys with a pinned Ed25519 gateway identity defeating active
  MITM (ADR-0007); CVH1 ClientHello/ServerHello mutual-auth handshake with admission
  (open=free, hosted=token, fail-closed); per-session **RFC 6479** sliding-window
  anti-replay (`internal/dedup/replay.go`, fuzzed + rollover-tested).
- **Real gateway exit node.** `internal/exit/*` (tunnel-IP allocator, IPv4 parse, expiring
  path-set, router with reverse-path filter + return-path duplication, Linux TUN, NAT
  health-check), wired into `DataGateway`. The probe gateway is **deployed and reachable
  over the internet** on the Oracle VPS (distroless arm64 container under systemd).
- **Swift↔Go bridge.** `bridge/continuitycore/*` — a handle-based cgo C-archive ABI
  (ADR-0005, no Go pointers crossing the boundary), with the handshake exposed
  (`cc_handshake_begin`/`cc_handshake_complete`). Apple paid programme is **Active** (team
  `D427C2J4RG`); the Xcode app + Network Extension code-sign with the
  `packet-tunnel-provider` entitlement; Phase 1 ran on hardware, Phase 2 is
  headless-verified (`xcodebuild … BUILD SUCCEEDED`).
- **Commercial scaffold.** `internal/entitlement` signed tokens + Open/Hosted gate;
  `StripeProvider` (stdlib, checkout + HMAC-verified webhooks). Licences decided and
  applied: **AGPL-3.0 gateway + Apache-2.0 client/core** (`docs/legal/licensing.md`).
- **Compatibility + observability + GTM groundwork.** `continuityctl preflight`
  (supported / needs-android / needs-wifi / needs-both / unsupported-macos); stdlib
  Prometheus `/metrics`; SEO-hardened `website/`, `docs/marketing/`, privacy/terms drafts,
  and a read-only `scripts/prepare-public.sh` GO/NO-GO audit.

**Commercial model (owner direction):** open-core — a FOSS client + gateway, with revenue
from a **paid hosted gateway** (the Oracle box) for users who don't want to self-host. The
gateway address must stay **configurable end-to-end** (self-host vs hosted); never
hard-coded.

## The remaining critical gate

**The userspace RNDIS host driver under the macOS App Sandbox.** The dual-path *concept*
is proven, but on **macOS specifically** the independent second WAN (the phone over USB)
needs a userspace RNDIS host driver, because OxygenOS-class phones pin RNDIS tethering and
macOS never brings up a NIC for it. That driver is proven in the spike — but as
**un-sandboxed C using libusb**, which the App Sandbox denies (libusb reaches the device
via legacy IOKit; `com.apple.security.device.usb` gates Apple's **IOUSBHost**, not
libusb).

So the single highest-risk experiment that governs the release shape is issue **#8**: does
an IOUSBHost **match → claim → INITIALIZE** succeed **inside a signed, sandboxed Network
Extension**?

- ✅ succeeds → the Mac App Store route (sandboxed, NE app extension, no kext/DriverKit,
  per `docs/product/footprint.md`) is viable; proceed to the Swift/IOUSBHost port (#9).
- ❌ fails → the App Store route is dead; fall back to a **Developer-ID app + privileged
  helper**.

Everything in Phase C and the App Store packaging decision hinges on this. Run it early.
NCM-default phones (Pixel/AOSP 14+) tether natively with zero install and do **not** need
the driver — but RNDIS-pinned phones (OnePlus and many OEMs) do, so the driver is required
for broad device support.

## Forward plan (phases mapped to issues)

### Phase A — Finish Stage 2: real tunnel, Linux-proven end-to-end

- Deliver the assigned tunnel IP to the client **in-band** (extended ServerHello / config
  channel) — the one wire step before routing real traffic — **#3**.
- Make `DataGateway` runnable: a `cmd/gateway` data mode (probe | data) that loads/persists
  an Ed25519 identity + entitlement config, serves the real handshake + data path, and
  exposes `/metrics`. The library and tests exist; only cmd wiring + deploy remain
  (`TASKS.md`).
- On-hardware Stage-2 demo: `curl` and a continuous SSH session through the tunnel
  survive cutting one uplink; `tcpdump` confirms encrypted egress; `/metrics` advances —
  **#4**.
- Gateway egress deploy hardening: systemd TUN address + MTU 1280, `ip_forward`, kernel
  `MASQUERADE`, privilege-drop after TUN+socket open, systemd hardening; confirm Oracle
  VCN ingress UDP 51820 — **#5**.
- Durability: tune the RFC 6479 window width against worst-case path skew so a late-but-
  valid copy is never misread as stale — **#6**; make SessionID/key single-use-per-gateway-
  lifetime an explicit, tested invariant — **#7**.

### Phase B — macOS Network Extension Phase 3 (Wi-Fi single-path first)

The first shipping client, over one path, end-to-end — **#10 (milestone 1)**:

- Override `makeRelay()` to assemble the cgo `CoreTransport`, a Wi-Fi `PathSender`
  (`NWConnection` pinned via `requiredInterface` / `IP_BOUND_IF`), packetFlow read →
  `relay.sendOutbound` → path, and path receive → `relay.receiveInbound` (dedup) →
  packetFlow write.
- Drive the on-wire handshake from Swift via the bridge ABI; distribute the pinned gateway
  Ed25519 identity (the on-wire handshake **message** + pinned-key distribution are the
  remaining wire steps).
- `setTunnelNetworkSettings`: MTU 1280, gateway-scoped included routes only (never the
  system default), management subnet excluded (`docs/product/footprint.md`). (Today it
  only sets `tunnelRemoteAddress`, so zero packets flow.)
- Live path state (up/down, RTT/jitter/loss) → app group → menu-bar UI; Keychain for the
  access key; `os.Logger` for `log stream` debugging.
- Exit criteria (spec Stage 4): Internet flows; clean stop restores routes/DNS; no Go
  callback touches released Swift memory; no deadlock across repeated connect/disconnect.

### Phase C — RNDIS productionisation: the independent second WAN on macOS *(top risk)*

- **GATING spike (#8):** throwaway IOUSBHost match → claim → INITIALIZE inside a real
  signed + sandboxed NE; record PASS/FAIL and the resulting architecture decision as an
  ADR (App Store vs Developer-ID + helper).
- **Port (#9):** keep the pure-framing C brain (`research/usb-rndis-spike/rndis_lib.{c,h}`)
  unchanged behind a modulemap (avoids re-incurring clean-room provenance risk); rewrite
  only the sandbox-incompatible USB I/O — `rndis_usb.c` → `RNDISUSBTransport`
  (IOUSBHost claim, control transfers, bulk pipes) and `rndis_uplink.c` → `RNDISUplink`
  (DHCP + ARP bring-up, each frame built/parsed by the C `rl_*` functions). Conform to the
  existing `PathSender` seam; run the provenance/licence reviewer before merge.
- Layer the RNDIS uplink into the dual-path provider — **#10 (milestone 2)**.
- Finalise the supported matrix (NCM-native vs RNDIS-driver) and decide build-vs-bundle so
  a supported user needs no manual setup (`TASKS.md` compatibility section).

### Phase D — Entitlement & payments to production *(parallelisable with B/C)*

- Real Stripe keys + account/key storage + the webhook endpoint the gateway/site exposes;
  wire the Open/Hosted gate live at the hosted gateway.
- Key lifecycle: issue / extend / expiry / revoke / takeover; one-device lease (specification
  §8.7/§9). Keep the gateway address configurable through the app UI and the NE tunnel.

### Phase E — Private alpha → beta (signed, notarised)

- Signed development build → notarised app; SwiftUI menu-bar/Settings/onboarding;
  diagnostics export (no secrets); in-app "Remove configuration & uninstall"
  (`removeFromPreferences()` + Keychain cleanup) and the footprint verification checklist;
  bundle-identifier namespace.
- Sleep/wake, network-change and captive-portal handling; operational dashboard + support
  runbook; ten-plus connect/disconnect cycles clean.

### Phase F — Security & release hardening *(human sign-off — LLMs assist, do not approve)*

- Refresh the threat model per data plane / handshake / USB surface; SBOM; broaden fuzzing;
  add **golangci-lint** + **gosec**; complete gateway privilege reduction; rate limiting /
  abuse controls; independent security review.
- CI hardening (carry-over): branch protection + required checks on `main`, confirm
  PR-triggered CI, pin SwiftLint.
- Pre-public release audit: **rewrite git history to drop the real LAN address**; add
  per-file SPDX headers; finalise CONTRIBUTING licence wording; lawyer review of
  `docs/legal/privacy-policy.md` + `terms-of-service.md`; run `scripts/prepare-public.sh`
  GO/NO-GO.

### Phase G — Public release (deliberately limited)

- Make the open-core monorepo public (`docs/dev/repository-strategy.md`); hosted gateway
  live on Oracle with the single-container self-host `docker run` quickstart documented;
  route the landing page through `continuityctl preflight`; record the demo video; Show HN
  / Product Hunt drafts in `docs/marketing/`.
- Operate limited: one gateway, controlled keys, defined SLIs, status page, support
  channel, documented shutdown procedure.

## Cross-cutting (continuous, not phases)

- **Configurable gateway address** end-to-end (self-host vs hosted) — never hard-coded;
  hold the line through the app UI and the NE tunnel.
- **Redaction invariants** — no MAC/IPv4/serial/raw-key leakage in diagnostics, logs or
  metrics; enforced by the smoke gate.
- **Provenance** — clean-room discipline for any GPL-inspired code; the RNDIS framing brain
  stays as the independently-written C, reviewed before any reuse decision; capture notices
  on any upstream reuse (`TASKS.md` standing conditions).
- **Observability** — extend the §11.4 metric set as each subsystem lands; bounded-
  cardinality pseudonymous session IDs only.

## GitHub project structure

Milestones map to phases; the existing issues attach to them (do **not** open duplicates).
The GitHub tools available in-session cannot create milestones or labels, so create the
milestones once (UI or `gh`), then attach:

| Milestone | Existing issues | Adds |
|---|---|---|
| Stage 2 — real tunnel | #3, #4, #5, #6, #7 | `cmd/gateway` data mode |
| macOS NE Phase 3 | #10 | on-wire handshake message + pinned-identity distribution |
| RNDIS productionisation | #8 (gating), #9 | supported-matrix + build-vs-bundle |
| Entitlement & payments | — | real Stripe keys/accounts; key lifecycle |
| Alpha → beta | — | notarisation; uninstall; sleep/wake |
| Security & release hardening | — | git-history rewrite; lawyer review; CI hardening |
| Public release | — | repo public; hosted gateway live |

Standing cross-cutting labels (no milestone): `ci`, `security`, `provenance`,
`observability`. The issue template is the §15.3 Definition-of-Done checklist. `TASKS.md`
remains the single ordered source of truth; issues link to it.

## Risk / effort tiering

| Phase | Tier | Note |
|---|---|---|
| C — RNDIS under sandbox | **Highest — feasibility** | #8 decides App Store vs Developer-ID+helper and whether the second WAN works on macOS for RNDIS phones. |
| B — NE Phase 3 | High integration | packetFlow I/O, Swift↔Go memory ownership, handshake-over-the-wire, route scoping. Apple-platform specialist reviewer. |
| A — finish Stage 2 | Moderate | Mostly wiring + deploy; the core landed. |
| D — payments | Moderate, idempotency-critical | Webhook idempotency + reconciliation = financial correctness. |
| E — alpha/beta | Mostly mechanical | UX, packaging, supportability. |
| F — hardening | Security — human sign-off | Independent review + legal; cannot be auto-closed. The git-history rewrite is a one-way release gate. |
| G — launch | Operational | Discipline over engineering. |

## Verification

Per work cycle, the existing gate is authoritative: run
`.claude/skills/run-continuityctl/smoke.sh verify` (build, stage marker, `darwin-evidence`
JSON + redaction invariant, usage errors, then `go vet`, `go test -race ./...`, dedup
benchmark, `gofmt`, `docs-check.sh`, `licence-check.sh`). Also `make test` / `make lint`.

Phase-specific end-to-end verification:

- **A:** on-hardware `curl`/SSH through the tunnel surviving a deliberate path cut; capture
  confirms ciphertext; gateway `/metrics` advances; recorded evidence attached to #4.
- **B:** Internet via gateway through the NE over Wi-Fi single-path; routes/DNS restored on
  disconnect; no use-after-free / deadlock across repeated connect/disconnect.
- **C:** documented PASS/FAIL of the sandboxed USB claim (#8); RNDIS uplink passes packets
  from a sandboxed NE with the C framing tests still green and throughput measured (#9).
- **D:** duplicate webhooks never double-extend; failed payments never activate keys; one
  key cannot hold two device leases.
- **F/G:** `scripts/prepare-public.sh` GO; git history clean of the LAN address; lawyer
  sign-off on privacy/terms before the repo goes public.

Provenance and security gates are blocking, not advisory: no upstream import without a
per-subtree licence scan, `NOTICE`/provenance updates and clean-room notes; the release
audit and human security/legal review must pass before public launch.
