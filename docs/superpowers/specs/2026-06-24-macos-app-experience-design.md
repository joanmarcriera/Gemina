# macOS app experience — design

Date: 2026-06-24
Status: Approved in brainstorming (owner), ready for an implementation plan.

## Goal

A great day-one experience for the Continuity VPN macOS app. The product's soul
is *reliability you can feel*: the app must make "you're protected, and you'd
survive a link dropping" obvious at a glance, prove its value over time, respect
the user's data and privacy, and let a power user tune behaviour — without
jargon.

This spec covers the app UI and behaviour. The transport, gateway, encryption,
handshake, entitlement and metrics already exist (Go core + cgo bridge); this is
the Swift/AppKit surface plus the small local state it needs. It does not cover
the NEPacketTunnelProvider runtime wiring (separate, Xcode-gated) beyond the
interfaces it consumes.

## Non-goals

- No new transport/crypto. The policy and impact features are client-side
  decisions and local counters over the existing transport.
- No analytics SDK, no tracking, no stable user id in any shared data.

## App shape (decided: A)

A **status-led menu-bar app** with a **Settings window**. The menu bar item is
the always-present surface; a popover shows live status; the Settings window
holds depth (the lesson from Tailscale outgrowing menu-bar-only and WARP being
criticised for ambiguous status).

## Menu-bar status (the hero)

The icon is unmistakable at a glance, conveyed by **glyph + colour together**
(colour-blind-safe; VoiceOver-labelled; legible in light and dark):

| State | Meaning | Treatment |
|---|---|---|
| Protected | both paths up | solid two-bars, green |
| Protected, degraded | one link down, surviving on the other | amber + "still covered" — framed as **success**, the product doing its job |
| At risk | only one connection, no redundancy | hollow/grey — "not protected against a drop" |
| Paused / Off | user paused | outline |
| Connecting | establishing | subtle pulse, respects Reduce Motion |

The "degraded" state is deliberately positive: a dropped link is exactly when the
product earns its keep.

## Popover (status-led)

Top-to-bottom:
1. **Headline status** ("Protected" / "Protected — one link down" / "Not
   protected — add a second connection" / "Paused").
2. **Per-path rows**: name • state dot • kind (e.g. "iPhone • up • cellular").
3. **Session impact line**: "This session: survived 3 drops · 47s of outage
   absorbed." (See Impact.)
4. **Gateway line**: Hosted/Self-host • live latency (from the benchmark ping) •
   a switch-gateway affordance.
5. **Primary action**: Pause / Resume protection.
6. **Footer**: Settings… / Quit. When there is no second path, a **"Check
   compatibility"** button runs the preflight.

## Settings window (SwiftUI Settings scene, tabbed per macOS HIG)

- **General** — launch at login; show in menu bar; auto-protect when a second
  connection appears.
- **Connections** — choose the two paths (auto-detect + manual); the **path
  policy** and **preferred path** (below); **which traffic to protect** (all, or
  scoped — ties to the footprint included-routes and the metered-data concern); a
  live **data-cost note** for the chosen policy; run the preflight; link to
  `COMPATIBILITY.md`.
- **Gateway** — Hosted (sign in) vs Self-hosted (address); a **Test latency**
  button (the benchmark, Hosted vs the user's box); the *don't-host-at-home* hint.
- **Privacy** — the **"Share anonymous compatibility data"** toggle with an
  **inline preview of exactly what is sent** (the redacted `preflight -share`
  block) and a privacy-policy link.
- **Impact** — the value-over-time view (below).
- **Advanced** — **Debug logging** toggle + **Export debug bundle…** (redacted,
  for testers) + log level.
- **About** — version, open-core licence, repo/docs links, and **Remove
  configuration & uninstall** (the footprint contract:
  `removeFromPreferences()` + keychain cleanup).

## Path policy (prioritising one network)

The send side chooses how many paths each packet goes over; the gateway
deduplicates regardless, so all modes work over the existing transport.

Modes (Settings → Connections):
- **Duplicate** — every packet over both paths. Seamless, zero-loss failover;
  doubles data on protected traffic.
- **Failover** — one primary, the other on hot standby; switch on degrade. Lowest
  data; a brief gap on the switch.
- **Smart** — primary-only normally; automatically duplicate **all** traffic for a
  short window whenever the primary path shows instability (a loss or latency
  spike measured by the benchmark pings / provider stats), then return to
  primary-only once it settles. The balance. (Per-flow "duplicate only this app"
  is future, not v1, to keep the behaviour unambiguous.)

**Preferred path**: manual ("prefer Wi-Fi, spare cellular") or **Auto** — prefer
the unmetered / lower-latency link (latency from the benchmark), use the metered
one only as needed.

**Default: Auto / Adaptive** — Duplicate over unmetered links; on metered cellular
fall back to Smart. Maximum protection when data is free; conserve it when it
isn't. Per-connection overridable. The UI always shows the data-cost implication
so the trade is honest. Metered detection: treat a cellular tether as metered by
default; let the user mark any connection metered/unmetered.

## Impact (showing usefulness)

Honest because measured, not invented. Fed by the client path-state events
(`path_state`, `failovers_survived` from the metrics vocabulary), stored
**locally** in the app container (the user's own history; never leaves the device
unless the user separately opts into the anonymous catalogue, which does **not**
include impact history).

- **Definitions**: *outage absorbed* = summed time one path was down while the
  session continued on the other; *failovers survived* = transitions where the
  session did not drop; *% protected* = share of session time with ≥2 usable
  paths; *longest drop survived*; *data per path*.
- **Surfaces**: a one-line session summary in the popover; an Impact tab with
  day / week / all-time figures and a simple sparkline.

## Telemetry / consent model

- **Free / self-host → opt-in** (off by default; onboarding invites helping the
  catalogue).
- **Paid (future) → opt-out** (on by default, **disclosed clearly at purchase**,
  one toggle to stop). Even opt-out must be transparent and easy to disable
  (Apple guidelines + UK/EU GDPR; covered by the lawyer review already pending on
  `docs/legal/privacy-policy.md`).
- **What is shared**: only the redacted `CompatibilityReport.ShareReport()` tokens
  (verdict, macOS version, tether mode) + app version. **Never** IPs, serials,
  traffic content, impact history, or a stable user/device id. The user can
  preview the exact payload before anything is sent.

## First-run onboarding

Welcome (one line + the dual-path diagram) → **Compatibility check** (runs the
preflight; honest verdict + next step) → **Gateway** (Hosted early-access or
Self-host address) → **Data-sharing consent** (explicit, *explained before
asked*, tier-appropriate default) → "You're protected." Skippable where sensible;
never blocks on a choice that has a safe default.

## Architecture mapping

- The UI is SwiftUI/AppKit in the app target. It reads live state from the NE
  provider (path up/down, current policy, gateway latency via the benchmark ping)
  and calls the Go core via the **cgo bridge** (`CContinuityCore` / `CoreTransport`).
- **Reuse, do not rebuild**: the share payload is `ShareReport()` (built,
  redaction-tested); gateway latency is the `continuityctl benchmark` ping; the
  debug bundle is the redacted diagnostics; uninstall is the footprint contract.
- **New Swift**: the menu-bar controller + popover, the Settings tabs, the
  onboarding flow, a small local **ImpactStore**, and a **PathPolicy** value type
  the relay's send decision consults.
- **New Go (small, optional, later)**: an opt-in "submit compatibility report"
  endpoint; the relay honouring a `PathPolicy` (send over one vs both). Neither
  blocks the UI design.

## Testing

Unit-testable in Swift (once the Xcode project exists), no UI harness needed:
- **Status mapping**: path states → menu-bar status enum (all combinations).
- **Consent default**: tier → default share setting (free=opt-in, paid=opt-out).
- **Policy selection**: (mode, preferred, metered?) → per-packet path set, incl.
  the Auto/Adaptive rules.
- **Impact maths**: a sequence of path-state events → outage-absorbed / failovers
  / %-protected (pure function over an event log).
The Go-side payloads (ShareReport, benchmark stats, diagnostics) are already
tested.

## Day-one quality bar

Full keyboard navigation and VoiceOver labels; colour-blind-safe status (glyph +
colour); Reduce-Motion respected; legible menu-bar icon in light/dark; plain
language in the main UI ("Wi-Fi", "your phone's cellular" — never "RNDIS"/"NCM",
which stay in Advanced/diagnostics); honest, non-alarming framing of the degraded
state.

## Out of scope / future

- iOS app; Windows/Linux client.
- Server-side aggregation of opt-in reports (a later, separately-reviewed
  endpoint).
- Per-app traffic scoping UI beyond a basic include/exclude (later).
