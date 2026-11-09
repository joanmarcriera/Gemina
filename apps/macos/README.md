# macOS app

The Swift half of Gemina: the menu-bar app and the `NEPacketTunnelProvider`
extension, linked to the Go transport core through the c-archive bridge
(`bridge/geminacore`, header `bridge/include/geminacore.h`).

## Xcode project

The Xcode project is generated with XcodeGen, not checked in:

```sh
cd apps/macos
xcodegen generate                          # Gemina.xcodeproj: app + GeminaTunnel extension
xcodegen generate --spec project-dev.yml   # GeminaDev.xcodeproj: menu-bar app only (free Personal Team)
```

`project.yml` needs a paid Apple Developer Program membership for the Network
Extension entitlement; its extension target builds the Go c-archive as a
pre-build script. Signing and entitlements: `docs/dev/xcode-signing.md`.

## Swift package

`Package.swift` is the headless, Xcode-free build used by `make test-macos` and
CI. Besides the library targets it carries self-checking executables that run
with the plain toolchain:

```sh
swift run --package-path apps/macos --disable-sandbox GeminaVPNCoreCheck
swift run --package-path apps/macos --disable-sandbox WiFiPathSenderCheck
swift run --package-path apps/macos --disable-sandbox CoreTransportCheck
```
