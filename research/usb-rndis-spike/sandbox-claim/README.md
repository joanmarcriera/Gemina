# Sandboxed USB claim test (WS task #2564, 2026-09-25)

Proves that an App-Sandbox process with `com.apple.security.device.usb` can claim and
drive the phone's RNDIS function (claim, INITIALIZE, packet filter, bulk read) exactly like
the unsandboxed spike; without the entitlement the plugin open fails with 0xe00002be.
Evidence table and hashes: https://familia.riera.co.uk/tasks/2564 (comment 1615).

- `sbclaim.c` — libusb + IOKit claim test; `mk.sh` builds and signs three .app wrappers
  (unsandboxed, sandbox+usb, sandbox-no-usb) with the two entitlement files; `run.sh` runs them.
- Phone side (Android 16, CPH2609): `svc usb setFunctions rndis` (NOT `rndis,adb` — rejected on
  Android 16; adb is re-added automatically as `rndis,none,adb`). Restore with `setFunctions ""`.
- Not covered: the GeminaTunnel extension itself (needs `device.usb` in its own entitlements),
  TestFlight/App Store signing, and actual tethered traffic (USB tethering needs the phone unlocked).
