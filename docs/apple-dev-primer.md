# Apple development, in plain English (for this project)

A map of the Apple-specific concepts behind the continuity-vpn macOS app, written
for someone who knows software but not the Apple ecosystem. You don't need to
memorise this — it's here so the words Claude and Xcode throw at you make sense.

## The one-paragraph mental model

Apple only lets code run on a Mac if it can prove **who built it** and **what it's
allowed to do**. "Who" = a **signing certificate** tied to your **team**. "What" =
**entitlements** (capabilities) granted by a **provisioning profile**. Building the
app staples all of that on via **code signing**. For *your own machine* during
development that's enough. To let *other people* download and run it, Apple adds
one more gate: **notarization**. That's the whole game.

## The vocabulary

- **Apple Developer Program** — the paid ($99/yr) membership. You're enrolled and
  **Active** (as of 2026-06-25). The free tier ("Personal Team") can't use the
  powerful capabilities — including the Network Extension this app needs — which is
  why activation mattered.
- **Team** — your identity as a publisher. Yours is **`D427C2J4RG`** (type:
  Individual). Everything you sign is "by" this team. (You also have an old free
  Personal Team and a leftover cert under a different ID, `476YVP24U6` — Xcode
  juggles them; don't worry about it.)
- **Signing certificate** — a cryptographic ID proving code came from your team.
  Two kinds matter here:
  - *Apple Development* — for running on your own machines while building. (What we
    used today.)
  - *Developer ID Application* — for shipping a downloadable app to other people.
    (We'll create this at release time.)
- **Bundle identifier** — the app's unique name in reverse-DNS, e.g.
  `com.joanmarcriera.continuity`. The extension has its own:
  `…continuity.tunnel`.
- **Entitlement** — a single permission the app declares it needs (e.g. "run a VPN
  tunnel", "talk to USB", "share data with my extension"). Listed in a
  `.entitlements` file. Our key one is the **Network Extension** entitlement.
- **Capability** — the Apple-portal side of an entitlement: you enable a capability
  for your app's identifier, Apple agrees, and that flows into a profile.
- **Provisioning profile** — Apple's signed permission slip that says "team
  `D427C2J4RG` may run app `com.joanmarcriera.continuity` with these
  entitlements on these machines". Xcode fetches these automatically; you saw two
  get downloaded today (one for the app, one for the extension).
- **Code signing** — the build step that wraps the cert + entitlements + a hash of
  the code around the app so macOS can verify it. "Signing the packages" = this.
- **Hardened runtime** — an extra-locked-down mode required before Apple will
  notarize. Enabled at signing time. (Relevant only at release.)
- **Notarization** — for distribution: you upload the finished app, Apple scans it
  for malware and issues a "ticket". **Stapling** attaches that ticket to the app
  so it works offline. Without it, downloaders see *"app can't be opened / damaged
  / unidentified developer"*.
- **Gatekeeper** — the macOS bouncer that checks all of the above when someone
  opens a downloaded app.

## What's specific to *this* app: the Network Extension

This app routes network traffic, so it ships a **Network Extension** — specifically
a **packet-tunnel provider**. macOS treats VPN-like code as privileged: it runs in
its **own sandboxed process** (`ContinuityTunnel.appex`) that the system launches
on demand, and the **user must approve the VPN configuration once** in System
Settings before it can run. This is the capability a free account can't have, and
it's why "is the membership active?" was the blocking question.

## What's done vs. what's next

- ✅ **Done:** membership active; app + Network Extension **built and code-signed**
  with your team; verified the NE permission is really baked in.
- ⏭ **Phase 3 (next):** make the tunnel actually move packets (Wi-Fi + the phone's
  USB uplink). Technical, not account/signing work.
- 🚢 **Release (later):** create the *Developer ID* cert, notarize and staple, then
  hand out a downloadable build. There's one known wrinkle for Network-Extension
  apps at this step that we'll check against Apple's docs when we get there.

## Where the detail lives (so you don't have to)

Claude has skills that encode the actual commands and gotchas, so future sessions
don't relearn them:
- **`macos-app-xcode-build`** — building & signing (incl. today's proven recipe).
- **`macos-network-extension-ops`** — loading/approving/debugging the tunnel (Phase 3).
- **`macos-app-distribution`** — Developer ID signing + notarization (release).

If you ever want to verify the membership yourself: developer.apple.com/account →
it should show your name with **"Developer Team"** (not "Personal Team") and a
green *Certificates, Identifiers & Profiles*.
