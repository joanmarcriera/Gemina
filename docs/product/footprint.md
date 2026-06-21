# System footprint and uninstall contract

This is the promise the product makes about what it writes to the user's Mac and
how completely it removes itself. It is a hard requirement, not an aspiration:
the app must be clean from day one, install with minimal friction, and leave
nothing behind on uninstall. Distribution target is the Mac App Store (see
DECISIONS.md, 2026-06-21), which keeps the footprint bounded by construction.

## Why the footprint stays small by construction

* **Sandboxed app.** The App Sandbox forces all app state into a single
  container, so removal is bounded and predictable rather than scattered across
  the system.
* **No kernel extension, no DriverKit, no system extension.** The second uplink
  is driven in userspace (ADR 2026-06-21), and on the App Store the
  NetworkExtension provider ships as a bundled *app extension* inside the bundle.
  There is therefore no system extension to approve at install or deactivate at
  uninstall.
* **No persistent system mutations.** Routing changes happen only inside the
  packet-tunnel while it is running and disappear when it stops. The app does not
  edit `/etc`, does not install a launch daemon, and does not persistently change
  the global network service order.

## Footprint inventory

Everything the app may create, and exactly how it is removed:

| # | Item | Location | Removal |
| --- | --- | --- | --- |
| 1 | App bundle | `/Applications/<App>.app` | Move to Trash |
| 2 | VPN configuration | NetworkExtension preferences (System Settings ▸ VPN) | `NETunnelProviderManager.removeFromPreferences()` via the in-app uninstall action |
| 3 | Session keys, if any | Keychain (tagged by bundle id) | Deleted by tag on uninstall |
| 4 | Preferences, caches, state | `~/Library/Containers/<bundle-id>/` | Removed with the container |
| 5 | Bundled app extension | inside the app bundle | Removed with the bundle (no separate sysext) |

Item 2 is the one most apps orphan and the one users feel as "intrusive". The
in-app uninstall MUST remove the VPN configuration before the user deletes the
bundle, and deleting the bundle alone must never leave a live VPN entry in
System Settings.

## Uninstall flow (the contract)

1. In-app **Remove configuration & uninstall** action:
   * disconnect the tunnel if connected;
   * `removeFromPreferences()` on every configuration this app created;
   * delete keychain items by tag.
2. User moves the app to Trash; the sandbox container goes with the standard
   "delete app data" path.
3. Verification: after uninstall, none of the inventory items remain — no VPN
   entry, no container, no keychain items, no extension registered.

## Verification checklist (run before any release)

* System Settings ▸ VPN shows no leftover configuration.
* `ls ~/Library/Containers/ | grep <bundle-id>` returns nothing.
* `security find-generic-password` by the app's tag returns nothing.
* `systemextensionsctl list` shows no extension from this app (there should
  never be one on the App Store build).
* The global network service order matches the pre-install baseline.

## Open items before this contract is testable

Tracked in `TASKS.md`: confirm the userspace USB claim works under the App
Sandbox (`com.apple.security.device.usb`); decide the bundle identifier
namespace; and implement the in-app uninstall action.
