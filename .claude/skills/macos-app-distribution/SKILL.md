---
name: macos-app-distribution
description: Ship the continuity-vpn macOS app OUTSIDE the Mac App Store — Developer ID signing, the hardened runtime, notarization with notarytool, and stapling, so Gatekeeper lets users open it. Use when packaging a release, producing a downloadable .app/.dmg/.pkg, setting up notarization in CI, or debugging "app is damaged / cannot be opened / unidentified developer". NOT for day-to-day dev builds (that's [[macos-app-xcode-build]]) — this is the release path. Note the Network-Extension caveat below.
---

# Distributing the macOS app (Developer ID + notarization)

This is the **release** path — distributing a downloadable app for self-hosters
(fits the open-core / hosted-gateway model, [[product-model-open-core-hosted-gateway]]).
It is distinct from the dev build in [[macos-app-xcode-build]]: dev builds run
locally signed with an *Apple Development* cert; **distribution needs a *Developer
ID Application* cert + the hardened runtime + notarization.** The Mac App Store is
the alternative path (different cert, no notarytool — Apple notarizes server-side)
and is documented separately if we ever go that way.

## The four requirements Gatekeeper enforces

1. **Developer ID Application** signing identity (create in Xcode → Settings →
   Accounts → Manage Certificates → +, or on developer.apple.com). Team
   `D427C2J4RG`. Confirm with `security find-identity -v -p codesigning` — you
   want a line reading `Developer ID Application: …`.
2. **Hardened runtime** on every Mach-O (`codesign -o runtime`). Required for
   notarization.
3. **Notarization** — upload to Apple's notary service, which scans for malware
   and signs a "ticket".
4. **Stapling** — attach the ticket to the app so it verifies offline.

## Sign inside-out, then archive

Xcode's "Archive → Distribute → Developer ID" does all of this through the GUI and
is the easiest path. For a scripted/CI build, sign **innermost first** (the Go
`.dylib`/archive consumers, then frameworks, then the `.appex`, then the `.app`),
each with hardened runtime and the right entitlements:

```bash
# Each nested code item, deepest first. --options runtime = hardened runtime.
codesign --force --timestamp --options runtime \
  --sign "Developer ID Application: … (D427C2J4RG)" \
  --entitlements <that-target>.entitlements <path-to-nested-binary>
# … repeat for frameworks and ContinuityTunnel.appex …
# Finally the app bundle (signs the wrapper; nested items already signed):
codesign --force --timestamp --options runtime \
  --sign "Developer ID Application: … (D427C2J4RG)" \
  --entitlements Resources/Continuity.entitlements Continuity.app
codesign --verify --deep --strict --verbose=2 Continuity.app   # must pass
```

Hardened runtime + the app sandbox can conflict with how the cgo core and the
userspace-RNDIS libusb access behave — test the **notarized, stapled** build, not
just the dev build, before claiming a release works.

## Notarize + staple (notarytool — altool is dead since 2023-11)

```bash
# One-time: store credentials in the keychain (app-specific password OR an
# App Store Connect API key — key is better for CI). NEVER hardcode/commit these.
xcrun notarytool store-credentials "continuity-notary" \
  --apple-id "joanmarcriera@gmail.com" --team-id D427C2J4RG   # prompts for an app-specific pw

# Notarization needs a container: zip the .app, or build a .dmg / .pkg.
ditto -c -k --keepParent Continuity.app Continuity.zip

xcrun notarytool submit Continuity.zip --keychain-profile "continuity-notary" --wait
#   --wait blocks until Accepted/Invalid. On Invalid:
xcrun notarytool log <submission-id> --keychain-profile "continuity-notary"  # why it failed

# Staple the ticket onto the .app (or the .dmg/.pkg you actually ship):
xcrun stapler staple Continuity.app
xcrun stapler validate Continuity.app
spctl -a -vvv --type exec Continuity.app   # Gatekeeper's verdict: "accepted, source=Notarized Developer ID"
```

You staple the **artifact you distribute**: if you ship a `.dmg`, staple the dmg
(after stapling the app inside it). A stapled app passes Gatekeeper offline.

## ⚠️ Network-Extension caveat — verify before the first release

Apps containing a `packet-tunnel-provider` Network Extension have **extra
distribution requirements** beyond a plain app: the NE capability for Developer
ID (non-App-Store) distribution generally needs a **Developer ID provisioning
profile that includes the Network Extension entitlement**, embedded in the app —
automatic signing won't always produce that for the Developer ID path. When we
reach the first real release, **confirm the current rules against Apple's docs**
(see Sources) rather than assuming the dev-signing flow carries over. This is the
single most likely thing to bite a NE app at release time.

## Don't reach for this yet

Phase 2 (dev signing) is done; Phase 3 (the working tunnel) comes first. This
skill exists so the release step is a known quantity, not a scramble. Sources:
- Apple — Notarizing macOS software before distribution: https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution
- Apple — Customizing the notarization workflow (notarytool/stapler): https://developer.apple.com/documentation/security/customizing-the-notarization-workflow
- Apple — Signing Mac software with Developer ID: https://developer.apple.com/developer-id/
