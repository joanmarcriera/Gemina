# Phase 3 — Working Wi-Fi Tunnel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the macOS app a working single-path (Wi-Fi) VPN — toggle on → approve config → handshake to the gateway → real IP packets flow through the `utun` and back.

**Architecture:** The host app installs a `NETunnelProviderManager` config and starts it. A concrete `NEPacketTunnelProvider` subclass performs the on-wire handshake **through the existing cgo bridge** (`cc_handshake_begin`/`cc_handshake_complete`, ADR-0007), learns its leased tunnel IP from the gateway's ServerHello (delivered in-band), applies real `NEPacketTunnelNetworkSettings`, and pumps packets over one Wi-Fi-bound UDP socket using the proven `DualPathRelay`. Cellular/RNDIS bonding is explicitly **out of scope** here — it is a separate follow-up plan once this loop is proven.

**Tech Stack:** Go 1.26 (gateway + cgo bridge), Swift 6 / NetworkExtension (provider + app), `pkg/clientcore` (transport core), `internal/gateway` + `internal/exit` (responder + IP leasing).

## Run status (2026-06-27) — WS-A..E DONE; only WS-F (on-hardware) remains

Branch `feat/phase3-wifi-tunnel` (not yet merged). Everything except the
on-hardware verification is done, each task TDD/build-verified with a per-task
commit. Full Go gate (build/vet/`-race`/gofmt) green; Swift `swift build` + three
headless checks (`GeminaVPNCoreCheck`, `WiFiPathSenderCheck`, `CoreTransportCheck`)
green; **headless AND signed `xcodebuild` both `** BUILD SUCCEEDED **`** (app + NE
extension + linked Go c-archive), signed with the paid team `D427C2J4RG` and the
`packet-tunnel-provider` entitlement verified on the `.appex`.

- **WS-D2** — `bfb19c3` `GeminaTunnelBootstrap` (principal class): reads gateway
  config from `providerConfiguration`, handshakes over a Wi-Fi-pinned UDP socket
  via `CoreTransport.connect`, installs real `NEPacketTunnelNetworkSettings` from
  the leased IP (default route, MTU 1380, optional DNS), runs `DualPathRelay`.
  Added an overridable `makeTunnelSettings` seam + fixed review M1 (strong self).
- **WS-E** — `9ea9703` `TunnelController` (install/start/stop via
  `NETunnelProviderManager`, publishes `NEVPNStatus`) + menu-bar Protect toggle +
  gateway-settings form; headline derived from live status.

- **WS-A** — A1 `d89d6fa` (ServerHello +4-byte AssignedIPv4, frame 118→122),
  A2 `2b4f29a` (gateway leases via an injected `Admitter.SetLeaser` hook, returns
  the IP, `EnableExit` wires `r.Lease`), A3 `aa062e2` (`cc_handshake_complete`
  gains a 4-byte out-param; both C headers updated).
- **WS-D1** — `946b5a6` provider state race-free (`OSAllocatedUnfairLock`).
- **WS-C** — `5d7a99d` `WiFiPathSender` (NWConnection + `requiredInterface`),
  headless loopback check. Adds `CGeminaCoreStubs` so checks link without the Go
  archive.
- **WS-B** — `6a0aa9b` `CoreTransport.connect(...)` over the bridge +
  `CoreTransportCheck` glue test (stubs became a deterministic fake, IP 10.99.0.5).
- **Review fixes** — `c280ea6`: added `cc_handshake_cancel` and a `defer` in
  `connect` so an aborted handshake cannot leak the ephemeral key (review H1);
  named the bad-handle test's wire size (H2).

Corrections applied vs the original task text: real funcs are
`EncodeServerHello`/`DecodeServerHello` (not Marshal/Parse); the IP is delivered
on the client via `Session.AssignedIPv4()` (keeps `Complete`'s signature stable);
the assigned IP is a fixed-offset field at `[118:122]`, **not** signed (documented
DoS-equivalence; data plane stays AEAD-authenticated — consider binding it into
the signed transcript as a follow-up).

Deferred review findings to fold into WS-D2/path-monitor work (not bugs today):
M1 `[weak self]` in `startTunnel` skips `readOutboundLoop` if self is nil (use a
strong capture in the bootstrap); M2 `currentPathStates`/`primaryUnstable` have no
writer yet — the future NE path-monitor write MUST go through `stateLock`; L1
`C.GoBytes` takes a signed length at the C boundary (pre-existing pattern across
all `cc_*`).

## Global Constraints

- **Swift 6, strict concurrency = complete** (set in `project.yml` and `Package.swift` tools-version 6.0). New provider code MUST be data-race-free under this mode — no `nonisolated(unsafe)` without a lock.
- **swiftlint `--strict` must pass** — line length ≤120 warn / ≤160 error; identifier min length 2 (`.swiftlint.yml`).
- **Go is the single source of truth for crypto/framing.** Never reimplement the handshake, AEAD, or dedup in Swift/CryptoKit — call the bridge. (ADR-0005, ADR-0007.)
- **Gateway address, token, and pinned Ed25519 identity are configuration**, carried in the VPN profile's `providerConfiguration`/`serverAddress`. Never hard-code a gateway. ([[product-model-open-core-hosted-gateway]])
- **No raw host IPs in tracked source/docs** — use `gateway.example.com` or TEST-NET ranges. (`scripts/prepare-public.sh` gate.)
- **Provider bundle id is exactly** `com.joanmarcriera.gemina.tunnel`; app group `group.com.joanmarcriera.gemina`. (The pending rename will change these — see the review notes; if the rename lands first, substitute the new prefix consistently.)
- **Every `startTunnel` path MUST call its completion handler** exactly once (success or error), or the VPN hangs on "Connecting…".
- Verification gate for Go work: `go build ./... && go vet ./... && go test ./...` all exit 0. For Swift: `swift build --package-path apps/macos` + `swiftlint lint --strict`.

---

## Current state (what already exists — do not rebuild)

- **cgo ABI** (`bridge/include/geminacore.h`, `bridge/geminacore/bridge.go`):
  - `int cc_handshake_begin(uint8_t *gatewayPub, char *token, uint8_t *out, int outCap, uint64_t *hsHandle)` → writes ClientHello to `out`, returns its length, sets `*hsHandle`.
  - `uint64_t cc_handshake_complete(uint64_t hsHandle, uint8_t *serverHello, int serverHelloLen, int dedupCapacity)` → returns a session handle (0 = failure).
  - `int cc_outbound(handle, payload, payloadLen, out, outCap)`, `int cc_inbound(handle, wire, wireLen, char *path, out, outCap, int *deliver)`, `void cc_session_free(handle)`.
- **Swift seam** (`apps/macos/PacketTunnelExtension/`): `TransportCore` + `PathSender` protocols, `DualPathRelay` (frames once, sends over selected paths, dedups inbound), and `GeminaTunnelProvider` — the skeleton whose `makeRelay()` throws `.notConfigured` and whose `setTunnelNetworkSettings` carries **no routes**. `CoreTransport` wraps the session ABI but only via the pre-shared-key `cc_session_new` path.
- **Gateway responder** (`internal/gateway/admission.go` `Admitter.Handshake`, UDP `:51820`, `GEMINA_GATEWAY_MODE=data`), Stage-2 exit (`internal/exit/*`, `GEMINA_GATEWAY_EXIT=on`) with an IP-leasing allocator, RFC-6479 replay. Green headless test: `internal/gateway/handshake_test.go`.
- Wire framing: `CVH1` handshake, `CVD1` data (30-byte header) in `pkg/clientcore/data.go`.

## File structure (created / modified by this plan)

- `pkg/clientcore/handshake_message.go` — ServerHello gains an assigned-tunnel-IP field (WS-A).
- `internal/gateway/admission.go` — responder leases an IP and writes it into ServerHello (WS-A).
- `bridge/geminacore/bridge.go` + `bridge/include/geminacore.h` — `cc_handshake_complete` gains an out-param for the assigned tunnel IPv4 (WS-A).
- `apps/macos/PacketTunnelExtension/CoreTransport.swift` — add a handshake-based factory returning the session + assigned IP (WS-B).
- `apps/macos/PacketTunnelExtension/WiFiPathSender.swift` — **new**: Wi-Fi-bound UDP socket sender + receive loop (WS-C).
- `apps/macos/PacketTunnelExtension/GeminaTunnelBootstrap.swift` — **new**: concrete provider, `makeRelay()`, real network settings, race-free state (WS-D).
- `apps/macos/PacketTunnelExtension/GeminaTunnelProvider.swift` — make state actor-isolated (WS-D).
- `apps/macos/Resources/GeminaTunnel-Info.plist` (via `project.yml`) — principal class → the bootstrap subclass (WS-D).
- `apps/macos/AppUI/TunnelController.swift` — **new**: `NETunnelProviderManager` install/start/stop (WS-E).
- `apps/macos/AppUI/GeminaApp.swift` — wire the on/off toggle (WS-E).

---

## WS-A — Deliver the leased tunnel IP in-band (Go + bridge)

The provider cannot set `NEPacketTunnelNetworkSettings.ipv4Settings` without the client's tunnel IP. The gateway's exit allocator already leases one; surface it in the ServerHello and out through the bridge. (GitHub issue #3.)

### Task A1: ServerHello carries the assigned tunnel IPv4

**Files:**
- Modify: `pkg/clientcore/handshake_message.go`
- Test: `pkg/clientcore/handshake_message_test.go`

**Interfaces:**
- Produces: `ServerHello` struct gains `AssignedIPv4 [4]byte`; `MarshalServerHello`/`ParseServerHello` round-trip it after the existing fields (append-only, bump `CVH1` minor or reserve the 4 bytes at a fixed offset — keep backward-compatible parse: zero = "unassigned").

- [ ] **Step 1: Write the failing test** — round-trip a ServerHello with `AssignedIPv4{10,99,0,2}` (TEST-range note: this is a value in code/test, allowed in `*_test.go`), assert `ParseServerHello(MarshalServerHello(h)).AssignedIPv4 == h.AssignedIPv4`.
- [ ] **Step 2: Run** `go test ./pkg/clientcore/ -run ServerHello -v` → FAIL (field missing).
- [ ] **Step 3: Implement** — add the 4 bytes to the struct and to marshal/parse; document zero = unassigned.
- [ ] **Step 4: Run** the test → PASS; then `go test ./pkg/clientcore/...` → all PASS.
- [ ] **Step 5: Commit** `feat(clientcore): carry the assigned tunnel IPv4 in ServerHello`.

### Task A2: Gateway leases an IP and writes it into ServerHello

**Files:**
- Modify: `internal/gateway/admission.go` (`Admitter.Handshake`)
- Test: `internal/gateway/handshake_test.go`

**Interfaces:**
- Consumes: the `internal/exit` allocator's lease call (`Allocator.LeaseOf`/equivalent) — reuse the existing leasing the exit router does on admit; do not introduce a second pool.
- Produces: `Handshake` sets `ServerHello.AssignedIPv4` from the lease; on `GEMINA_GATEWAY_EXIT=off` it stays zero.

- [ ] **Step 1: Write the failing test** — drive a full handshake against the in-process responder with exit enabled; assert the parsed ServerHello has a non-zero `AssignedIPv4` within the configured pool.
- [ ] **Step 2: Run** `go test ./internal/gateway/ -run Handshake -v` → FAIL.
- [ ] **Step 3: Implement** the lease + assignment in `Admitter.Handshake`.
- [ ] **Step 4: Run** → PASS; `go test ./internal/...` → all PASS.
- [ ] **Step 5: Commit** `feat(gateway): assign a tunnel IP during the handshake`.

### Task A3: Surface the assigned IP through `cc_handshake_complete`

**Files:**
- Modify: `bridge/geminacore/bridge.go`, `bridge/include/geminacore.h`
- Test: `bridge/geminacore/handshake_test.go`

**Interfaces:**
- Produces: ABI becomes `uint64_t cc_handshake_complete(uint64_t hsHandle, uint8_t *serverHello, int serverHelloLen, int dedupCapacity, uint8_t assignedIPv4[4])` — last arg is a 4-byte out buffer the bridge fills from the parsed ServerHello. Update the header doc comment.

- [ ] **Step 1: Write the failing test** — in `handshake_test.go`, complete a handshake whose ServerHello has a known IP; assert the 4 out-bytes match.
- [ ] **Step 2: Run** `go test ./bridge/geminacore/ -run Handshake -v` → FAIL.
- [ ] **Step 3: Implement** the extra out-param in the Go export + the C header signature.
- [ ] **Step 4: Run** → PASS; rebuild the archive: `go build -buildmode=c-archive -o apps/macos/build/libgeminacore.a ./bridge/geminacore` (exit 0).
- [ ] **Step 5: Commit** `feat(bridge): return the assigned tunnel IP from cc_handshake_complete`.

---

## WS-B — Swift handshake factory on `CoreTransport`

`CoreTransport` currently only wraps the pre-shared-key path. Add a factory that does the handshake using the bridge, with the network I/O injected (so it is testable and the provider owns the socket).

### Task B1: `CoreTransport.connect(...)` factory

**Files:**
- Modify: `apps/macos/PacketTunnelExtension/CoreTransport.swift`
- Test (headless): `apps/macos/CoreCheck/main.swift` — add a check that drives `connect` with in-memory `send`/`recv` closures backed by a Go-side fake ServerHello loaded from a fixture; assert it yields a non-nil transport and the expected assigned IP. (CoreCheck runs under the plain toolchain via `swift run GeminaVPNCoreCheck`.)

**Interfaces:**
- Produces:
  ```swift
  struct HandshakeResult { let core: CoreTransport; let assignedIPv4: (UInt8, UInt8, UInt8, UInt8) }
  static func connect(
      gatewayPublicKey: Data,           // 32-byte Ed25519 identity
      token: String,                    // entitlement token
      dedupCapacity: Int32,
      sendClientHello: (Data) throws -> Void,   // provider writes to the Wi-Fi socket
      receiveServerHello: () throws -> Data     // provider reads one datagram
  ) throws -> HandshakeResult
  ```
  Internals: call `cc_handshake_begin` into a stack buffer → `sendClientHello(clientHello)` → `let serverHello = try receiveServerHello()` → `cc_handshake_complete(...)` with the 4-byte out buffer → on non-zero handle build `CoreTransport(adopting: handle)`; throw `CoreTransportError.coreRejected` on a zero handle.

- [ ] **Step 1: Write the failing CoreCheck case** for `connect`.
- [ ] **Step 2: Run** `swift run --package-path apps/macos GeminaVPNCoreCheck` → FAIL (no `connect`).
- [ ] **Step 3: Implement** `connect` + a private `init(adopting handle: UInt64)`.
- [ ] **Step 4: Run** CoreCheck → PASS; `swift build --package-path apps/macos` exit 0.
- [ ] **Step 5: Commit** `feat(macos): handshake-based CoreTransport.connect over the bridge`.

---

## WS-C — Wi-Fi-bound UDP path sender + receive loop

### Task C1: `WiFiPathSender` (bind to the Wi-Fi interface, send/receive datagrams)

**Files:**
- Create: `apps/macos/PacketTunnelExtension/WiFiPathSender.swift`
- Test: `apps/macos/UnitTests/WiFiPathSenderTests` (loopback) — bind two sockets on the loopback interface (`localhost`), assert a datagram sent via `send` is received by the receive loop and surfaced to the `onDatagram` callback.

**Interfaces:**
- Produces:
  ```swift
  final class WiFiPathSender: PathSender {
      let name = "wifi"
      init(gatewayHost: String, gatewayPort: UInt16, boundInterface: String?) throws  // IP_BOUND_IF when non-nil
      func send(_ datagram: Data) throws                 // PathSender
      func receiveLoop(onDatagram: @escaping (Data) -> Void)   // re-arms; runs on its own queue
      func close()
  }
  ```
  Use a `Network.framework` `NWConnection` with `requiredInterface` set to the Wi-Fi interface (preferred over raw `IP_BOUND_IF`/BSD sockets under the app sandbox), `.udp`. The handshake's `sendClientHello`/`receiveServerHello` closures (WS-B) are backed by this connection before the receive loop is handed to inbound packet handling.

- [ ] **Step 1: Write the failing loopback test.**
- [ ] **Step 2: Run** it → FAIL (type missing).
- [ ] **Step 3: Implement** `WiFiPathSender` with `NWConnection`.
- [ ] **Step 4: Run** → PASS; `swift build` exit 0; `swiftlint lint --strict` clean.
- [ ] **Step 5: Commit** `feat(macos): Wi-Fi-bound UDP path sender with receive loop`.

---

## WS-D — Concrete provider: `makeRelay()`, real settings, race-free state

### Task D1: Make provider state data-race-free (fixes review H-1)

**Files:**
- Modify: `apps/macos/PacketTunnelExtension/GeminaTunnelProvider.swift`

**Interfaces:**
- Produces: `relay`, `currentPathStates`, `primaryUnstable` guarded by an `os_unfair_lock` (or a dedicated serial `DispatchQueue`); `handleInbound`, `readOutboundLoop`, `startTunnel`, `stopTunnel` all go through the guarded accessors. No behavioural change yet — this is the concurrency-correctness refactor that `swift build` currently warns about (`[#SendableClosureCaptures]`).

- [ ] **Step 1:** Add a `private let stateLock = OSAllocatedUnfairLock(initialState: State())` holding the three fields; replace direct access with `withLock`.
- [ ] **Step 2: Run** `swift build --package-path apps/macos` → the `SendableClosureCaptures` warnings on `GeminaTunnelProvider` are gone, exit 0.
- [ ] **Step 3: Commit** `refactor(macos): make packet-tunnel provider state race-free`.

### Task D2: `GeminaTunnelBootstrap` — concrete `makeRelay()` + network settings

**Files:**
- Create: `apps/macos/PacketTunnelExtension/GeminaTunnelBootstrap.swift`
- Modify: `apps/macos/project.yml` (principal class), then `cd apps/macos && xcodegen generate`

**Interfaces:**
- Consumes: `CoreTransport.connect` (WS-B), `WiFiPathSender` (WS-C), `DualPathRelay` (existing).
- Produces: `final class GeminaTunnelBootstrap: GeminaTunnelProvider` overriding `makeRelay()`:
  1. Read `gatewayHost`/`port`/`gatewayPublicKey`/`token` from `(protocolConfiguration as? NETunnelProviderProtocol)?.providerConfiguration`.
  2. `let wifi = try WiFiPathSender(gatewayHost:…, gatewayPort:…, boundInterface: primaryWiFiBSDName())`.
  3. `let hs = try CoreTransport.connect(gatewayPublicKey:…, token:…, dedupCapacity: 1024, sendClientHello: { try wifi.send($0) }, receiveServerHello: { try wifi.receiveOneDatagram(timeout: 5) })`.
  4. Stash `hs.assignedIPv4` so `startTunnel` can build settings; start `wifi.receiveLoop { [weak self] in self?.handleInbound($0, path: "wifi") }`.
  5. `return DualPathRelay(core: hs.core, paths: [wifi], policy: PathPolicy(mode: .auto))`.
- Also override the settings build: replace the route-less `NEPacketTunnelNetworkSettings` with
  ```swift
  let settings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: gatewayHost)
  let ipv4 = NEIPv4Settings(addresses: [assignedIPv4String], subnetMasks: [hostMask])  // hostMask = the /32 host mask
  ipv4.includedRoutes = [NEIPv4Route.default()]      // full-tunnel for the demo; scope later per footprint.md
  settings.ipv4Settings = ipv4
  settings.mtu = 1380                                  // headroom for the 30-byte CVD1 header + UDP/IP
  settings.dnsSettings = NEDNSSettings(servers: [resolverAddress])   // configurable resolver from providerConfiguration
  ```
- `project.yml`: set `NSExtensionPrincipalClass` to `$(PRODUCT_MODULE_NAME).GeminaTunnelBootstrap` and regenerate the Xcode project.

- [ ] **Step 1:** Implement `GeminaTunnelBootstrap`.
- [ ] **Step 2:** Update `project.yml` principal class; run `xcodegen generate`.
- [ ] **Step 3: Build the app target in Xcode** (`xcodebuild -project apps/macos/Gemina.xcodeproj -scheme Gemina -configuration Debug build`) → BUILD SUCCEEDED. (Note: this is the first task that needs full Xcode, not just `swift build`.)
- [ ] **Step 4: Commit** `feat(macos): concrete packet-tunnel bootstrap with real network settings`.

---

## WS-E — App install UI + on/off toggle

### Task E1: `TunnelController` — install/start/stop via `NETunnelProviderManager`

**Files:**
- Create: `apps/macos/AppUI/TunnelController.swift`

**Interfaces:**
- Produces: `@MainActor final class TunnelController: ObservableObject` with `@Published var status: NEVPNStatus`, and:
  ```swift
  func installIfNeeded(gatewayHost: String, gatewayPort: UInt16,
                       gatewayPublicKeyBase64: String, token: String) async throws
  func start() throws        // manager.connection.startVPNTunnel()
  func stop()                // manager.connection.stopVPNTunnel()
  ```
  `installIfNeeded` does `loadAllFromPreferences` → reuse or create one manager → set `NETunnelProviderProtocol` with `providerBundleIdentifier = "com.joanmarcriera.gemina.tunnel"`, `serverAddress = gatewayHost`, and `providerConfiguration = [host, port, gatewayPublicKey, token]` → `isEnabled = true` → `saveToPreferences` (this triggers the one-time system approval prompt). Observe `NEVPNStatusDidChange` to update `status`.

- [ ] **Step 1:** Implement `TunnelController`.
- [ ] **Step 2: Build** in Xcode → BUILD SUCCEEDED.
- [ ] **Step 3: Commit** `feat(macos): NETunnelProviderManager install/start/stop controller`.

### Task E2: Wire the toggle into the menu bar (fixes review H-2)

**Files:**
- Modify: `apps/macos/AppUI/GeminaApp.swift`

**Interfaces:**
- Consumes: `TunnelController`.
- Produces: `StatusView` owns `@StateObject var tunnel = TunnelController()`; add a `Toggle("Protect", isOn:)` whose binding maps `tunnel.status == .connected` → `start()`/`stop()`. Derive the displayed `ProtectionStatus` from `tunnel.status` (`.connecting` while connecting, `.off` when disconnected) instead of the DEBUG preview array. Keep the `#if DEBUG` preview rows only when `tunnel.status == .invalid` (not yet installed).

- [ ] **Step 1:** Add the toggle + status mapping.
- [ ] **Step 2: Build** in Xcode → BUILD SUCCEEDED; `swiftlint lint --strict` clean.
- [ ] **Step 3: Commit** `feat(macos): on/off VPN toggle wired to the tunnel controller`.

---

## WS-F — On-hardware verification (manual; cannot run in CI)

No unit test can prove the real tunnel; verify on a Mac with a running gateway. Use the [[macos-network-extension-ops]] runbook.

- [ ] **Step 1: Stand up a dev gateway** with exit on (reuse `scripts/deploy-dev-gateway.sh` / the Oracle box). Note its address, the printed base64 Ed25519 identity, and a valid token. Do **not** paste secrets into the repo.
- [ ] **Step 2: Run the app**, enter gateway host/key/token, toggle Protect → approve the system VPN prompt once.
- [ ] **Step 3: Confirm the interface and state:**
  ```bash
  scutil --nc list                       # service shows Connected
  ifconfig | grep -A3 utun               # utun has the assigned 10.99.x address
  log stream --predicate 'subsystem == "com.joanmarcriera.gemina"' --level debug
  ```
- [ ] **Step 4: Prove packets flow** — `curl https://gateway.example.com/` (or a known echo) through the tunnel; watch byte counters rise (`netstat -ib | grep utun`). Confirm an IPv6 request is delivered (regression guard for the `ipFamily` fix already landed).
- [ ] **Step 5: Toggle off** → `stopTunnel` tears down the socket; `scutil --nc list` shows Disconnected; no orphan `GeminaTunnel` process (`pgrep -fl GeminaTunnel`).
- [ ] **Step 6: Record the result** in `PROJECT_STATE.md` and update the `phase3-ne-tunnel-plan` memory; open follow-up issues for full-tunnel route scoping (`footprint.md`) and configurable DNS.

---

## Out of scope (separate future plans)

- **Cellular/RNDIS second path + true bonding** ([[userspace-rndis-dataplane]], `research/usb-rndis-spike/`) — add a second `PathSender` and exercise `PathPolicy` duplicate/failover once the Wi-Fi loop is proven.
- **Developer ID + notarization release pipeline** — the NE provisioning-profile caveat, manual signing, notarytool/stapler (see `macos-app-distribution` skill). Needed before public distribution, not before this demo.
- **Product rename** — if not already done, applies to bundle ids / app group / principal class used above.

## Self-review notes

- Spec coverage: WS-A→F cover the memory note's WS3 (provider bootstrap), WS4 (install UI), WS5 (verify) plus the issue-#3 in-band tunnel IP precursor and the H-1/H-2 review items. WS1 (bridge handshake export) is already landed and intentionally omitted.
- Type consistency: `CoreTransport.connect` (B1) is the only producer of a `CoreTransport`/assigned IP consumed by `GeminaTunnelBootstrap` (D2); `WiFiPathSender` (C1) is the `PathSender` consumed by D2 and the I/O backing B1's closures; `TunnelController` (E1) is consumed by `GeminaApp` (E2). No dangling references.
- Testability honesty: A1–A3, B1, C1, D1 have headless tests; D2, E1, E2 are build-gated; F is manual on hardware (flagged).
