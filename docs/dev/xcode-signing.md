# Xcode project, entitlements and signing (owner action)

This is the manual, Apple-account-bound work that cannot be done headless or by a
coding agent: producing a signed macOS app + Packet Tunnel Extension that loads
the proven transport. The Go transport core (`pkg/clientcore`), its encryption
(ADR-0006) and the data path are built and tested; this turns them into a
runnable, signed app. Do these in order.

## 0. Prerequisites

- A Mac with the full **Xcode** (not just Command Line Tools).
- **Apple Developer Program** membership (paid) — required for the Network
  Extension and USB entitlements and for notarisation/App Store.
- The Go toolchain (already used by this repo) for building the core as a
  C-linkable archive.

## 1. Create the app + extension in Xcode

The repo's `apps/macos` is a SwiftPM package (build-only); a signed app with a
Network Extension needs a real Xcode project.

1. New Xcode project → **macOS App** (SwiftUI). Set a bundle id, e.g.
   `com.<yourorg>.continuity`.
2. Add a target → **Network Extension** → **Packet Tunnel Provider**. Bundle id
   `com.<yourorg>.continuity.tunnel` (must be a child of the app id). This is the
   bundled **App Extension** route from `docs/product/footprint.md` (no system
   extension, App Store compatible).
3. Add the existing Swift sources: `apps/macos/Shared/*`, and the bridge sketch
   `apps/macos/PacketTunnelExtension/DualPathTransport.swift` into the extension
   target. The real `NEPacketTunnelProvider` subclass fills the seam: read
   `packetFlow`, call the core to frame, send over each path, dedup inbound,
   write back.

## 2. Capabilities and entitlements

On the **app** target → Signing & Capabilities, add:

- **App Sandbox** (required for App Store).
- **Network Extensions** → tick **Packet Tunnel**. This adds
  `com.apple.developer.networking.networkextension = [packet-tunnel-provider]`.
- **USB** device access: add `com.apple.security.device.usb` (Boolean `YES`).
  This is what lets the extension claim the Android RNDIS function from userspace
  — re-confirm the libusb claim still succeeds **inside the sandbox** (this is the
  open gating task in `TASKS.md`; the spike ran un-sandboxed).
- **App Groups** (e.g. `group.com.<yourorg>.continuity`) so the app and the
  extension can share configuration/keys.

Mirror the Network Extension + App Group entitlements on the **extension** target.

## 3. Link the Go core into the extension

Build the transport core as a C-linkable static archive and link it from Swift
over the narrow C boundary (ADR-0005):

```sh
# arm64 (Apple Silicon); add an amd64 slice + lipo if you support Intel.
GOOS=darwin GOARCH=arm64 \
  go build -buildmode=c-archive -o build/libcontinuitycore.a ./<cgo-bridge-pkg>
```

The `<cgo-bridge-pkg>` is a small `//export`ed cgo wrapper around
`pkg/clientcore` exposing `cc_session_new / cc_outbound / cc_inbound /
cc_session_free` (see ADR-0005 for the signatures and the memory-ownership rule:
Swift owns the buffers; Go retains nothing across calls). Add the generated
`.a` + `.h` to the extension target and import via a bridging header. **This cgo
bridge is the next code task** and can be built/tested headlessly once written —
only the signing below needs your account.

## 4. Signing

- Set both targets' **Team** to your Developer Team; let Xcode manage signing, or
  create explicit provisioning profiles that include the **Network Extensions**
  capability (Apple provisions `packet-tunnel-provider` automatically once the
  capability is on the App ID).
- For **App Store**: archive → validate → upload. The AGPL gateway is **never**
  bundled in the app; the client is Apache-2.0 precisely so it can ship here
  (`docs/legal/licensing.md`).
- For **direct distribution**: Developer ID sign + **notarise**
  (`xcrun notarytool submit`) + staple.

## 5. Run and verify (on a real Mac)

- Network Extensions do **not** run in the Simulator. Run on the Mac.
- On first run, approve the VPN configuration (and the USB prompt when the phone
  is attached).
- Point the client at a gateway by address (self-hosted or hosted) — it is
  configuration, never hard-coded.
- Validate the data path: a tunnelled flow survives Wi-Fi or the phone dropping,
  mirroring the proven `rndis_dualpath` result but for real traffic.

## What remains in code (not blocked on your account)

- The cgo C-shared bridge around `pkg/clientcore` (§3).
- The concrete `NEPacketTunnelProvider` subclass (packetFlow loop + the two path
  senders: a Wi-Fi `IP_BOUND_IF` socket and the userspace RNDIS uplink).
- The session-key handshake (ADR-0006 leaves key agreement as future work).
