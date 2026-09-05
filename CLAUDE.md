# Gemina VPN — Developer Onboarding

**What it is:** A macOS reliability tool that keeps calls and SSH sessions alive by duplicating encrypted traffic over two independent uplinks at once (Wi-Fi + Android USB tether / cellular / second broadband). The gateway deduplicates incoming copies; if one link drops, the other carries the session seamlessly. Pre-release: dual-path UDP proof is **done** (2026-06-23); Wi-Fi-only tunnel is **code-complete** (2026-06-28); awaiting on-hardware verification (WS-F).

## How to build and test

```bash
make bootstrap      # Prepare Go workspace, Swift package cache, docs
make test           # Go race tests + Swift build check + docs validation
make lint           # gofmt, golangci-lint (if installed), swiftlint (if installed)
make ci             # Run all local CI gates in one shot
```

The project is **Go + Swift** (`go 1.26`, Xcode project in `apps/macos/`):

- **Go core:** `cmd/` (CLI + gateway), `internal/` (transport, dedup, diagnostics), `pkg/clientcore` (client/gateway core)
- **Swift app:** `apps/macos/` (menu-bar + NetworkExtension packet tunnel, linked to Go via cgo c-archive bridge)
- **Research only:** `.research-src/` (Git-ignored, pinned upstream sources for due diligence)

## Key conventions

1. **Multi-stage delivery:** Stage 0 (done), Stage 1 (probe; proven), Stage 2 (data+exit; done), Stage 3 (Wi-Fi tunnel; code-complete). Read `PROJECT_STATE.md` and `TASKS.md` for the current gate.

2. **Commit style:** Conventional Commits (`type(scope): summary`). End every commit with the co-author trailer. Changelog auto-generated from commits via `cliff.toml`.

3. **Decisions and ADRs:** Material choices go in `DECISIONS.md`; formal architecture records in `docs/adr/0*.md` (see `AGENTS.md` for the ADR-triggering scope).

4. **Tests first, TDD:** Table-driven Go tests, race-detector enabled (`go test -race ./...`). All new logic must be unit tested; benchmarks for performance-sensitive paths.

5. **Go formatting:** `gofmt` is enforced on every Edit/Write via `.claude/hooks/gofmt-on-edit.sh`; CI gates it with `go vet ./...`.

6. **Dual-license:** AGPL-3.0 gateway (`cmd/gateway/`, `internal/gateway/`), Apache-2.0 client + core (`apps/macos/`, `pkg/`, `cmd/geminactl`, shared `internal/`).

7. **No GPL code in product:** `scripts/fetch-research-sources.sh` clones upstream (Engarde, OpenMPTCProuter, WireGuard) to `.research-src/` **only**. WireGuard Apple and wireguard-go are permitted as foundations; do not copy GPL into product directories.

## Project layout

```
.
├── apps/macos/          # Swift app (menu-bar, NetworkExtension provider)
├── bridge/geminacore    # cgo C-shared bridge (Go core → Swift ABI)
├── cmd/
│   ├── gateway/         # Server: dedup + exit node
│   └── geminactl/       # CLI: probe, diagnostics (darwin-evidence), preflight
├── internal/
│   ├── dedup/           # FIFO ring (Stage 1 probes) + RFC 6479 sliding-window (data)
│   ├── exit/            # Stage 2 exit node: TUN, IPv4 parse, router, reverse-path filter
│   ├── gateway/         # Server: handshake, admission, DataPlane, metrics
│   ├── paths/           # Path classification (Wi-Fi, Android USB, etc.)
│   ├── platform/darwin/ # macOS USB, BSD interface, ioreg evidence collection
│   ├── diagnostics/     # darwin-evidence JSON report (redacted, host-IPs never logged)
│   ├── entitlement/     # Tokens, Stripe webhook verification, Open/Hosted gate
│   ├── protocol/        # Probe packet codec, SessionID, PacketNumber, PathTag
│   └── transport/       # PathDialer, bound-socket UDP egress (IP_BOUND_IF)
├── pkg/clientcore/      # Client/gateway core: AES-256-GCM per session, key agreement (X25519), handshake
├── docs/
│   ├── adr/             # Architecture Decision Records (ADR-0001 onwards)
│   ├── product/         # Spec, monetisation study, project roadmap
│   ├── legal/           # Privacy policy, ToS drafts, licensing rationale
│   ├── dev/             # Gateway deploy, Xcode build, public-repo audit
│   └── marketing/       # Launch playbook, social/press kit, SEO strategy
├── deploy/              # Docker (gateway), systemd units, Terraform/OpenTofu, monitoring
├── scripts/             # bash: bootstrap, lint, test gates, gateway deploy, research fetch
├── .claude/
│   ├── skills/          # Project-specific skills (gateway-ops, xcode-build, RNDIS, etc.)
│   └── settings.json    # Hooks: gofmt-on-edit, redaction-guard on all Edit/Write
└── DECISIONS.md, PROJECT_STATE.md, TASKS.md  # Session handover
```

## Gotchas

1. **Apple Developer membership required:** NetworkExtension (`NEPacketTunnelProvider`) is unavailable on a free Personal Team. Paid membership activates the entitlement. Builder must have the right team ID (`D427C2J4RG` as of 2026-06-28).

2. **macOS has no native RNDIS driver:** Android phones over USB RNDIS tether require a userspace driver (no kext, no SIP, no root). The proof is in `research/usb-rndis-spike/rndis_dualpath.c`; shipping integration into `NEPacketTunnelProvider` remains (App Sandbox re-confirmation of USB claim pending).

3. **Redaction guard blocks real LAN/public IPs:** `.claude/hooks/redaction-guard.sh` warns on any Edit/Write if a file contains private/public IP or domain names you type (catches commits of real addresses). `scripts/prepare-public.sh` audit must scrub git history before public release.

4. **Two gateway modes:** `cmd/gateway` with `GEMINA_GATEWAY_MODE=probe` (Stage 1 UDP probes) and `GEMINA_GATEWAY_MODE=data` (Stage 2 data plane). Deploy units in `deploy/systemd/` (probe = live on oracle; data+exit = draft, validate on hardware during WS-F).

5. **CI gating incomplete:** PR-triggered checks, branch protection, and SwiftLint pinning are tracked in `TASKS.md` but not yet enforced. Local `make ci` runs all gates, but main is not protected.

6. **Swift build needs `--disable-sandbox`:** SwiftPM `sandbox-exec` fails inside Codex sandbox. Use `--disable-sandbox` flag for local Swift commands.

## Before you start

Read in order:
1. `PROJECT_STATE.md` — current stage, risks, test results
2. `TASKS.md` — ordered work queue, what's blocked
3. `DECISIONS.md` — material choices, revisit conditions
4. `docs/product/project-specification.md` — full scope
5. `AGENTS.md` — contributor rules, ADR triggers

Only change code **within your current stage gate**. Avoid production VPN, NetworkExtension packet handling, or payment work unless explicitly unblocked. Record architectural choices in `DECISIONS.md` or `docs/adr/`.

Run `make ci` before handing over.
