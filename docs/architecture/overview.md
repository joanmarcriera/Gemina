# Architecture Overview

## Stage

Stage 2: real tunnel. Stage 1 (dual-path UDP probe) is proven. The current gate
and task list are in `TASKS.md` and `PROJECT_STATE.md`.

What exists in code: packet identity and duplicate suppression (`internal/dedup`,
including an RFC 6479 replay window), path classification and Darwin evidence
(`internal/paths`, `internal/platform/darwin`), bound-socket UDP egress
(`internal/transport`), the authenticated handshake and AES-256-GCM data plane
(`pkg/clientcore`), the gateway in probe and data modes with an optional TUN exit
(`cmd/gateway`, `internal/gateway`, `internal/exit`), the cgo bridge
(`bridge/geminacore`) and a Wi-Fi-only `NEPacketTunnelProvider` tunnel in
`apps/macos/` that is not yet verified on hardware. Payments and a hosted
entitlement service do not exist yet.

## Product Shape

The commercial product is a macOS continuity VPN. The client will protect traffic, send duplicate protected copies over public Wi-Fi and Android USB tethering, and deliver them to one gateway VPS (host-agnostic; the dev gateway runs on Oracle Cloud). The gateway will accept the first valid copy and discard duplicates.

The product is not an aggregate-bandwidth or multi-client-platform product in its first release.

## Planned Components

* `apps/macos/`: Swift and SwiftUI macOS application plus Packet Tunnel Extension source directories.
* `cmd/`: Go command entry points for future gateway, control and test tools.
* `internal/`: private Go implementation packages.
* `pkg/`: public Go packages only where cross-boundary reuse is justified.
* `bridge/`: C-compatible Swift/Go bridge (`bridge/geminacore`, cgo c-archive) and boundary notes.
* `api/`: API and protocol documents.
* `deploy/`: OpenTofu, cloud-init, Ansible, systemd, nftables and container artefacts.
* `research/`: upstream manifest and clean-room research notes.

Stage 1 probe boundaries are recorded in `docs/architecture/stage-1-probe.md`.

## Security Posture

Packets are encrypted with AES-256-GCM bound to the packet identity (ADR-0006), with session keys from an X25519 + HKDF-SHA256 exchange and a pinned Ed25519 gateway identity (ADR-0007), all from the Go standard library. WireGuard was considered and not used for the data plane (see ADR-0006). This project must not design new cryptographic primitives.

Secrets, raw access keys and private traffic must not be logged. Future implementation work must update the threat model before introducing security-sensitive code.

## Provenance Posture

No upstream implementation code is committed to product directories. Research sources are fetched into `.research-src/`, which is Git-ignored.

Engarde and OpenMPTCProuter are inspiration-only unless a later legal decision explicitly changes the distribution model.
