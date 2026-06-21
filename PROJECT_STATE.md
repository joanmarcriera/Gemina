# Project State

Last updated: 2026-06-21

This is the durable cross-session handover. It records current state, risks,
blockers and the last validation run. The outstanding task list lives in
`TASKS.md`; architectural choices live in `DECISIONS.md`; the full change history
lives in Git. Keep this file lean — do not grow an append-only log here.

## Current Stage and Objective

Stage 1: dual-path UDP probe. Stage 0 exit criteria are met and reviewed.

The probe so far proves packet identity, first-copy duplicate suppression, and
fixture/command-driven Darwin path classification, exposed through a redacted
`continuityctl darwin-evidence` diagnostic. It does **not** yet prove per-path
UDP egress, gateway reachability, packet-capture evidence, path-loss survival,
encryption or any VPN behaviour.

Next action is recorded at the top of `TASKS.md`: run the redacted Darwin
evidence diagnostic with Android USB tethering connected.

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
  reduced from `networksetup`/`ioreg` to redacted tokens.
* `internal/diagnostics` + `cmd/continuityctl darwin-evidence` — redacted JSON
  Stage 1 evidence report labelled diagnostic-only-not-path-success.
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
  socket leaves a specific path regardless of the default route. Loopback
  delivery test on darwin; portable validation tests; non-darwin stub. Proven
  live: `en0`-bound reached the gateway, `lo0`-bound was correctly trapped
  ("network is unreachable"). The client per-path *egress mechanism* now exists;
  the simultaneous two-path proof still needs the phone + cabled rig.

Not implemented: VPN transport (encryption/framing for real traffic), production
dedup integration, NetworkExtension packet handling, the simultaneous two-path
client probe (single-path egress binding now exists), entitlement service,
payment flow, real infrastructure resources, and direct
SystemConfiguration / Network
framework / IORegistry API collection (only command-backed collection exists).

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
* The local diagnostic reports the Android tether as a missing candidate even
  when the phone is connected with tethering on. Root cause confirmed
  2026-06-21: macOS ships no RNDIS host driver, so the phone never becomes a
  `enX` NIC for the BSD-interface collector to see, and the IORegistry matcher
  also requires the literal token `android` which this OnePlus
  (`OnePlus`/`KALAMA`/`RNDIS …`) lacks. Fix tracked in `TASKS.md`.
* In-app uplink acquisition is decided and de-risked (ADR 2026-06-21). A
  userspace spike (`research/usb-rndis-spike/`) opened the phone's RNDIS
  interfaces and completed `REMOTE_NDIS_INITIALIZE` (`status=0`, `medium=802.3`)
  from an unprivileged process under SIP. The data plane and App-Sandbox
  re-confirmation remain to build.
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

## Definition of Dual-Path Success (do not claim early)

Do not claim dual-path success until later work proves that UDP socket A
explicitly leaves through Wi-Fi, UDP socket B explicitly leaves through Android
USB tethering, both reach the same gateway process, one logical packet is
delivered once, and either path can disappear without ending the logical session.
