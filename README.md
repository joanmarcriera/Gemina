# Continuity VPN

Stage 1 dual-path UDP probe repository for a commercial macOS continuity VPN.

The first release is a reliability product, not an aggregate-bandwidth product. It will duplicate protected traffic over public Wi-Fi and Android USB tethering, send both copies to one Hetzner gateway, accept the first valid packet and discard duplicates.

Authoritative product and engineering scope lives in:

* `docs/product/project-specification.md`
* `AGENTS.md`
* `PROJECT_STATE.md`
* `TASKS.md`
* `DECISIONS.md`

## Current Stage

Stage 1: dual-path UDP probe.

Stage 0 repository bootstrap and source due diligence are complete and reviewed.
Current Stage 1 work is limited to proving the smallest useful probe: explicit
Wi-Fi and Android USB tethering path evidence, duplicated UDP probes, gateway
deduplication and path-loss survival. Do not begin production VPN transport,
NetworkExtension packet handling, payment flows or entitlement implementation
until later gates are explicitly opened.

## Bootstrap

```sh
make bootstrap
make test
make lint
make licence-check
```

`make fetch-research` clones pinned upstream sources into `.research-src/`, which is ignored by Git. It must not copy upstream implementation files into product directories.

## Source Rules

* Do not copy GPL implementation code into proprietary product directories.
* Engarde and OpenMPTCProuter are inspiration-only.
* WireGuard Apple and wireguard-go are permitted foundations subject to retained notices and provenance records.
* Do not invent cryptography.
* Do not log secrets or store raw access keys.
