# macOS Skeleton

This directory contains Stage 0 source directories and a Swift package used for compile-only validation.

An Xcode project has not been generated in Stage 0 because NetworkExtension signing, team identifiers, entitlements and build settings require owner review.

## Manual Xcode Creation Steps

1. Create a macOS App target named `ContinuityVPN`.
2. Add a Packet Tunnel Extension target named `ContinuityVPNPacketTunnelExtension`.
3. Add a shared Swift framework or package target for code under `Shared/`.
4. Link the app and extension to the shared target.
5. Add NetworkExtension entitlements only after the Apple Developer Team ID and signing approach are approved.
6. Add the Go bridge target only after ADR and threat-model updates for Swift/Go memory ownership.

The Swift package here is only a Stage 0 validation scaffold.
