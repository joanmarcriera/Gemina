# Architecture Overview

## Stage

Stage 1 dual-path UDP probe.

Only the initial Go packet identity, first-copy duplicate-suppression core,
fixture-driven path-candidate classifier and conservative Darwin interface-state
collector exist. No production transport, packet framing, live macOS path
binding, NetworkExtension packet handling, gateway networking, entitlement
service, payment flow or real infrastructure resource exists yet.

## Product Shape

The commercial product is a macOS continuity VPN. The client will protect traffic, send duplicate protected copies over public Wi-Fi and Android USB tethering, and deliver them to one Hetzner gateway. The gateway will accept the first valid copy and discard duplicates.

The product is not an aggregate-bandwidth or multi-client-platform product in its first release.

## Planned Components

* `apps/macos/`: Swift and SwiftUI macOS application plus Packet Tunnel Extension source directories.
* `cmd/`: Go command entry points for future gateway, control and test tools.
* `internal/`: private Go implementation packages.
* `pkg/`: public Go packages only where cross-boundary reuse is justified.
* `bridge/`: C-compatible Swift/Go bridge boundary notes and future build outputs.
* `api/`: API and protocol documents.
* `deploy/`: OpenTofu, cloud-init, Ansible, systemd, nftables and container artefacts.
* `research/`: upstream manifest and clean-room research notes.

Stage 1 probe boundaries are recorded in `docs/architecture/stage-1-probe.md`.

## Security Posture

WireGuard is the planned cryptographic foundation. This project must not design new cryptographic primitives.

Secrets, raw access keys and private traffic must not be logged. Future implementation work must update the threat model before introducing security-sensitive code.

## Provenance Posture

No upstream implementation code is committed to product directories. Research sources are fetched into `.research-src/`, which is Git-ignored.

Engarde and OpenMPTCProuter are inspiration-only unless a later legal decision explicitly changes the distribution model.
