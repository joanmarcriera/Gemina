---
name: macos-app-xcode-build
description: Build, sign, run and verify the continuity-vpn macOS app (menu-bar app + NetworkExtension packet tunnel) from the XcodeGen project, linking the Go transport core via a cgo c-archive. Use whenever working in apps/macos, on the Xcode project, code signing, entitlements, the Network Extension, the Swift app/views, or diagnosing any macOS build/signing failure. Reach for this for anything touching project.yml, xcodegen, xcodebuild, provisioning, "Developer Program / Personal Team", the cgo bridge, or the NEPacketTunnelProvider.
---

# Building the continuity-vpn macOS app

The macOS app is an XcodeGen-generated project, not a hand-built one. The Xcode
project is regenerated from `apps/macos/project.yml`; never edit the `.xcodeproj`
by hand (it is git-ignored). The Go transport core links in as a cgo c-archive.

**You can verify almost everything headlessly with `xcodebuild`** once
`xcode-select` points at Xcode — do that and iterate before asking the owner to
touch the GUI. The only owner-only steps are choosing a signing Team and clicking
Run.

## Layout

- `apps/macos/Package.swift` — SwiftPM, used only for **headless logic checks**
  (it compiles `ContinuityVPNCore`, `Shared`, the C module, the extension
  skeleton, and runs `ContinuityVPNCoreCheck`). It does **not** build the app.
- `apps/macos/project.yml` — the real Xcode project (XcodeGen): app +
  NetworkExtension + frameworks. `project-dev.yml` is the no-extension variant.
- `apps/macos/AppUI/` — the SwiftUI app (Xcode-only; not in SwiftPM).
- `apps/macos/Core/` — `ContinuityVPNCore` framework (PathPolicy, ProtectionStatus,
  Consent, Impact — pure, tested).
- `apps/macos/PacketTunnelExtension/` — the NE provider + the cgo glue
  (`CoreTransport.swift`, `Bridging-Header.h`, `DualPathTransport.swift`).
- `apps/macos/CContinuityCore/` — the C ABI module (header + modulemap); mirrors
  `bridge/include/continuitycore.h`.

Targets: `Continuity` (app), `ContinuityTunnel` (packet-tunnel app-extension),
`ContinuityVPNCore`, `ContinuityVPNShared` (frameworks).

## Prerequisites (one-time, owner runs the sudo bits)

`xcode-select` must point at Xcode, not the Command Line Tools:

```bash
sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
sudo xcodebuild -license accept
xcodebuild -version   # confirm it works
```

## Generate the project

```bash
cd apps/macos
xcodegen generate                       # full app + NE  -> Continuity.xcodeproj
xcodegen generate --spec project-dev.yml  # no NE        -> ContinuityDev.xcodeproj
```

Re-run `xcodegen generate` after any `project.yml` change, then in Xcode the
project reloads (quit/reopen if it doesn't).

## Headless verification (do this to iterate)

```bash
# Compile + link only — isolates code errors from signing. Use this to iterate.
xcodebuild -project Continuity.xcodeproj -scheme Continuity \
  -destination 'platform=macOS' clean build CODE_SIGNING_ALLOWED=NO

# Surface the real signing/provisioning error (needs the owner's account):
xcodebuild -project Continuity.xcodeproj -scheme Continuity \
  -destination 'platform=macOS' build -allowProvisioningUpdates

# What signing certs/teams are installed:
security find-identity -v -p codesigning
```

Grep the output for `error:|BUILD FAILED|BUILD SUCCEEDED|warning:.*macOS version`.

## Signing the full app + Network Extension (PROVEN 2026-06-25)

The paid Apple Developer Program is **active**. The signed build works
end-to-end. Recipe:

**Team IDs — there are two, do not confuse them:**
- **`D427C2J4RG`** = the paid team (`teamType: Individual`,
  `isFreeProvisioningTeam: false`). **This is the one to pass as
  `DEVELOPMENT_TEAM`.**
- `476YVP24U6` = the old free Personal Team / the existing keychain cert
  ("Apple Development: joanmarcriera@gmail.com"). Xcode reconciles that cert
  against the `D427C2J4RG` profiles, so signing succeeds even though the cert's
  team string differs.

**One owner GUI step is required first** (CLI alone cannot do it): the Apple ID
must be signed into Xcode → Settings → Accounts (GUI + 2FA), and the paid team
selected in the project's Signing & Capabilities on **both** the `Continuity` app
target AND the `ContinuityTunnel` extension target. That registers the team in
Xcode's provisioning store and fetches the two profiles (app + tunnel) into
`~/Library/Developer/Xcode/UserData/Provisioning Profiles/`. After that the build
is fully headless:

```bash
cd apps/macos
xcodebuild -project Continuity.xcodeproj -scheme Continuity \
  -destination 'platform=macOS' -allowProvisioningUpdates \
  DEVELOPMENT_TEAM=D427C2J4RG clean build      # -> ** BUILD SUCCEEDED **
```

**Verify the signature + that the NE entitlement actually made it in:**

```bash
DD=$(xcodebuild -project apps/macos/Continuity.xcodeproj -scheme Continuity \
  -showBuildSettings 2>/dev/null | awk -F' = ' '/ BUILT_PRODUCTS_DIR /{print $2}')
APP="$DD/Continuity.app"; APPEX="$APP/Contents/PlugIns/ContinuityTunnel.appex"
codesign --verify --deep --strict --verbose=2 "$APP"     # valid on disk + DR
codesign -dv "$APP" 2>&1 | grep TeamIdentifier            # -> D427C2J4RG
codesign -d --entitlements - --xml "$APPEX" | plutil -p - | \
  grep -i 'networkextension\|packet-tunnel'                # -> packet-tunnel-provider
```

The `packet-tunnel-provider` entitlement on a signed `.appex` is the definitive
proof the paid membership is active — a free team is refused this capability.

**Gotcha that burned time:** before the account is signed into Xcode AND the team
picked in the GUI, `xcodebuild` fails with **"No Account for Team …"**. That is a
missing account *session*, NOT a membership verdict — don't misread it as "not
enrolled yet". The CLI cannot establish the session; only the GUI sign-in can.

`DEVELOPMENT_TEAM` is deliberately **not** committed in `project.yml` (open-core
repo — avoid leaking a personal team ID). Pass it on the command line, or put it
in a gitignored xcconfig.

## Linking the Go core (the hard-won recipe)

The extension links `bridge/continuitycore` built as a c-archive. In `project.yml`
the `ContinuityTunnel` target:

- **Pre-build script** builds the archive, matched to the deployment target so the
  linker does not warn:
  ```sh
  export PATH="/opt/homebrew/bin:$PATH"
  export MACOSX_DEPLOYMENT_TARGET=14.0
  export CGO_CFLAGS="-mmacosx-version-min=$MACOSX_DEPLOYMENT_TARGET"
  cd "$SRCROOT/../.." && go build -buildmode=c-archive \
    -o "$SRCROOT/build/libcontinuitycore.a" ./bridge/continuitycore
  ```
- **Bridging header** (`PacketTunnelExtension/Bridging-Header.h`) `#import`s
  `continuitycore.h` (set `SWIFT_OBJC_BRIDGING_HEADER`). Swift then calls `cc_*`
  directly — no Swift module.
- `CoreTransport.swift` guards `import CContinuityCore` with
  `#if canImport(CContinuityCore)` so it works in **SwiftPM** (the module) and
  **Xcode** (the bridging header).
- Link flags: `OTHER_LDFLAGS = -lcontinuitycore -lresolv -framework CoreFoundation
  -framework Security`; `HEADER_SEARCH_PATHS = $(SRCROOT)/CContinuityCore/include`;
  `LIBRARY_SEARCH_PATHS = $(SRCROOT)/build`.

`apps/macos/build/` is git-ignored (regenerated by the pre-build script).

## Gotchas (each cost real time)

1. **Network Extensions require the PAID Apple Developer Program — active.**
   ✅ Resolved 2026-06-25: membership is Active (team `D427C2J4RG`) and the NE
   signs. See "Signing the full app + Network Extension" above for the recipe. A
   free **Personal Team** is refused with *"Personal development teams … do not
   support the Network Extensions capability."*; the no-NE `ContinuityDev`
   variant (`project-dev.yml`) remains the fallback for pure UI iteration.
2. **Framework targets need `GENERATE_INFOPLIST_FILE: "YES"`** or the app's embed
   step fails: *"Framework … did not contain an Info.plist."*
3. **Command Line Tools ship no XCTest / Swift-Testing.** `swift test` fails with
   *no such module 'XCTest'* / *'Testing'*. The logic is verified by the
   self-checking executable `swift run ContinuityVPNCoreCheck`; it ports to a test
   target once full Xcode is the selected toolchain.
4. **`object file was built for newer 'macOS' version` warning** — the Go archive
   targeted the host OS; fixed by the `MACOSX_DEPLOYMENT_TARGET`/`CGO_CFLAGS` in
   the pre-build script.
5. **It is a menu-bar (agent) app** (`INFOPLIST_KEY_LSUIElement: "YES"`): no Dock
   icon, no window — look for the antenna in the top-right menu bar. "Nothing
   happened" usually means they're looking in the Dock.
6. **Keep the vendored C header in sync** with `bridge/include/continuitycore.h`
   (`diff` them).

## Owner GUI steps (after a green headless build)

1. Open the project (`open apps/macos/Continuity.xcodeproj`).
2. Project → target → **Signing & Capabilities** → **Automatically manage
   signing** → pick **Team** (set it on the app AND the extension if the extension
   shows a signing error).
3. Scheme = the app target → **▶**.

## Phase status

- **Phase 1 — done & run:** the menu-bar app (`AppUI/ContinuityApp.swift`,
  `MenuBarExtra`) renders `ProtectionStatus` over preview data. Built, signed, ran
  on the Personal Team.
- **Phase 2 — DONE & SIGNED (2026-06-25):** the NE extension embedded + the Go
  core linked, app + `.appex` code-signed with the paid team `D427C2J4RG`, NE
  carrying the `packet-tunnel-provider` entitlement. Verified with `codesign
  --verify --strict`. See the signing section above.
- **Phase 3 — next:** implement the real `NEPacketTunnelProvider` —
  `makeRelay()` building the two `PathSender`s (a Wi-Fi `IP_BOUND_IF` socket + the
  userspace RNDIS uplink from `research/usb-rndis-spike/`), driving the on-wire
  handshake (`clientcore.BeginClientHandshake`/`Complete`) from Swift, and feeding
  live path state into the menu bar + the policy. Much of it is headless-buildable;
  see [[userspace-rndis-dataplane]].
