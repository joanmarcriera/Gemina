---
name: macos-network-extension-ops
description: Install, approve, load, run and DEBUG the continuity-vpn packet-tunnel Network Extension on macOS — the NEPacketTunnelProvider lifecycle, the NETunnelProviderManager install flow, the System Settings approval prompt, reading the extension's logs, and the common "it won't start / won't connect" failures. Use for Phase 3 work (the real tunnel), for anything touching NEPacketTunnelProvider / NETunnelProviderManager / packetFlow / setTunnelNetworkSettings, or when the VPN toggle does nothing, the extension never loads, or no packets flow. Pairs with [[macos-app-xcode-build]] (build/sign) and [[userspace-rndis-dataplane]] (the uplink).
---

# Running & debugging the macOS Network Extension

The tunnel ships as a **packet-tunnel-provider *app extension*** (`ContinuityTunnel.appex`,
embedded in `Continuity.app/Contents/PlugIns/`), **not** a System Extension. That
choice matters for how it loads and how you debug it:

- An **app extension** is owned by its host app, loaded on demand by the system
  when a VPN configuration that names it is enabled. There is **no
  `systemextensionsctl install` step** — that command is for System Extensions
  (`.dext`/sysext), a different model. Don't go looking for it here.
- The app extension's provider class is an `NEPacketTunnelProvider` subclass; the
  entitlement is `com.apple.developer.networking.networkextension →
  packet-tunnel-provider` (already present and signed — see [[macos-app-xcode-build]]).

## The install → approve → load flow (what actually has to happen)

The extension cannot just "run". The **host app** must create a VPN configuration
in the system's preferences, the **user must approve it once**, and only then can
the tunnel be started. Sequence:

1. **App creates/saves a config** via `NETunnelProviderManager`:
   - `NETunnelProviderManager.loadAllFromPreferences` → reuse or make one.
   - Set `.protocolConfiguration` to an `NETunnelProviderProtocol` with
     `providerBundleIdentifier = "com.joanmarcriera.continuity.tunnel"` and a
     `serverAddress` string (shown in the UI; for us, the gateway address — keep
     it configurable per [[product-model-open-core-hosted-gateway]]).
   - `.isEnabled = true`, then `saveToPreferences`.
2. **First save triggers a system approval prompt** ("…would like to add VPN
   configurations"). Until the user clicks Allow, the config is inert. After
   approval it appears in **System Settings → VPN** (and Network).
3. **Start the tunnel:** `manager.connection.startVPNTunnel()` (optionally with
   `options:` passed through to the provider). The system spawns the `.appex`
   process and calls `startTunnel(options:completionHandler:)`.

A development-signed build loads fine locally — **notarization is only for
distribution**, not for running your own dev build. The user still has to approve
the VPN config the first time.

## NEPacketTunnelProvider lifecycle (the contract to implement)

- `startTunnel(options:completionHandler:)` — build the uplink(s), do the on-wire
  handshake, then call `setTunnelNetworkSettings(_:)` with an
  `NEPacketTunnelNetworkSettings` (tunnel addresses, routes, DNS). **You MUST call
  the completion handler** — call it only after settings are applied (or with an
  error). Forgetting it = the VPN spins forever on "Connecting".
- Pump packets: read with `packetFlow.readPackets(completionHandler:)` (re-arm it
  each callback — it's one-shot), write with `packetFlow.writePackets(_:withProtocols:)`.
  Outbound = read from `packetFlow` → send on the uplink; inbound = uplink →
  `writePackets`.
- `stopTunnel(with:completionHandler:)` — tear down sockets/USB, then call the
  handler.
- `sleep`/`wake` — optional; quiesce on sleep.
- Keep a strong reference to the relay; the provider is short-lived per call.

For our two-path design the provider builds the Wi-Fi `IP_BOUND_IF` sender + the
userspace RNDIS uplink ([[userspace-rndis-dataplane]]) and drives
`clientcore.BeginClientHandshake`/`Complete` over the wire via the cgo bridge.

## Debugging (where the time goes)

The extension runs as its **own process**, so app `print`/breakpoints won't show
its logs. Use the unified log:

```bash
# Live stream just the extension's subsystem (set a real os.Logger subsystem in Swift):
log stream --level debug --predicate 'subsystem == "com.joanmarcriera.continuity"'

# Or by process once it's running:
log stream --predicate 'process == "ContinuityTunnel"' --level debug

# Dump the last 10 min after a failed connect (no need to have streamed live):
log show --last 10m --predicate 'process == "ContinuityTunnel"' --info --debug

# Is the provider process even alive / what does the system think of the config?
pgrep -fl ContinuityTunnel
scutil --nc list                     # lists VPN services + state (Connected/Disconnected)
ifconfig | grep -A3 utun            # the utun interface appears when settings apply
```

In Xcode you can also **Debug → Attach to Process → ContinuityTunnel** after it
spawns, or set the scheme to wait-for-launch. Add an `os.Logger(subsystem:
"com.joanmarcriera.continuity", category: ...)` early in the provider so there's
something to grep.

## Common failures (symptom → cause)

| Symptom | Likely cause |
|---|---|
| Toggle does nothing / no approval prompt | App never called `saveToPreferences`, or `providerBundleIdentifier` doesn't match the `.appex` bundle id exactly. |
| Stuck on "Connecting…" forever | `startTunnel` never called its `completionHandler`, or `setTunnelNetworkSettings` completion errored/ignored. |
| Connects then 0 bytes flow | `readPackets` not re-armed each callback, or routes in `NEPacketTunnelNetworkSettings` don't cover the traffic. |
| Extension never spawns | Embedded `.appex` not signed with the NE entitlement / wrong team — re-check [[macos-app-xcode-build]] signing section. |
| "permission denied" opening USB in the extension | App-sandbox + `com.apple.security.device.usb` needed in the *extension's* entitlements (it's already there); confirm the libusb path works inside the sandbox. |
| Config vanished after rebuild | Reinstalling the app can orphan the saved VPN config; `loadAllFromPreferences` and remove stale ones, or delete it in System Settings → VPN. |

## Quick reset when state gets weird

```bash
scutil --nc list                     # find the service
# Remove the VPN config in System Settings → VPN (– button), then re-run the app
# so it re-creates and re-prompts. Approving again is normal after a clean reinstall.
```

## Status

Phase 2 (build + sign of app and `.appex`) is **done**. Phase 3 — implementing
the real `NEPacketTunnelProvider` body and the app-side `NETunnelProviderManager`
install UI — is the next work and is what this skill supports.
