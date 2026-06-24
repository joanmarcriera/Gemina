# Compatibility catalogue — "works with…"

A community-maintained list of which **Mac + second-connection** combinations
work, so a prospective user can check before they start. Contributions welcome.

## Our privacy stance

We do **not** auto-collect anything about your devices. There is no silent
telemetry. This catalogue grows only from reports people choose to submit. The
`continuityctl preflight -share` command prints a **redacted** technical summary
(verdict, macOS version, tether mode) — it contains no IP, MAC, serial or phone
identifier. *You* decide whether to add your phone model when you submit. See
[`docs/product/compatibility-catalogue.md`](docs/product/compatibility-catalogue.md).

## How to contribute a report

1. Run the check: `continuityctl preflight -share`.
2. Open a pull request (or an issue) adding a row to the table below. Paste the
   redacted summary into the PR and fill in your phone model and any notes.

## Second-connection compatibility

The second path can be **any independent connection** — your phone's cellular
(iPhone or Android), a second broadband line, or an LTE/USB-Wi-Fi dongle. The
table tracks the trickier phone-tether cases.

| Second connection | macOS | Tether mode | Status | Notes |
|---|---|---|---|---|
| iPhone (Personal Hotspot over USB) | 11+ | native (no driver) | ✅ expected to work — native macOS NIC | Needs a verified report |
| Pixel / AOSP 14+ (USB tether) | 11+ | native NCM | ✅ expected to work — native macOS NIC | Needs a verified report |
| OnePlus 12R (OxygenOS 16) | 26.x | app-driver RNDIS | ⚙️ tether function detected; needs the app's userspace driver | Verified 2026-06 (dev rig) |
| Other Android (USB tether, RNDIS) | 11+ | app-driver RNDIS | ⚙️ should work via the app's userspace driver | Needs reports |
| Wired second line / LTE dongle | 11+ | native | ✅ any second NIC works | Needs reports |

Legend: ✅ works (native, no app driver needed) · ⚙️ works via the app's bundled
userspace driver · ❌ not supported.

> Status reflects the *transport* mechanism. The shipping macOS app is still in
> development; see [`PROJECT_STATE.md`](PROJECT_STATE.md).
