---
name: run-geminactl
description: Run, build, test, verify, and drive the gemina Go project and its geminactl CLI (including the darwin-evidence diagnostic). Use to validate a work cycle (vet + race tests + benchmark + gofmt + docs/licence gates) in one command and to orient quickly without re-reading every doc.
---

# Run & verify gemina (geminactl)

`gemina` is a Go monorepo for a macOS continuity VPN. Today the only
runnable binary is **`geminactl`**; `gateway`, `entitlement-api`,
`packet-generator`, `network-simulator` are stage-marker stubs. The whole thing
is driven by one script: **`.claude/skills/run-geminactl/smoke.sh`** (build +
drive the CLI + assert redaction invariants; `verify` adds the full gate).

Paths below are relative to the repo root (the unit). Run on **macOS** — the
`darwin-evidence` subcommand reads live BSD interface state, `networksetup` and
`ioreg`.

## Orientation (read this instead of re-reading every doc)

- **Stage:** Stage 1 — dual-path UDP probe. Stage 0 is complete and reviewed.
- **What exists & is tested:** `internal/protocol` (packet identity),
  `internal/dedup` (ring-buffer first-copy suppression, benchmarked),
  `internal/paths` (Wi-Fi / Android-USB-tether candidate classification, no
  hard-coded interface names), `internal/platform/darwin` (snapshot boundary +
  live BSD collector + evidence-derived link kinds + command-backed evidence),
  `internal/diagnostics` + `cmd/geminactl darwin-evidence` (redacted JSON).
- **What does NOT exist:** UDP egress, gateway runtime, encryption, entitlement,
  payments, NetworkExtension. Don't claim dual-path success without packet
  captures (see the bottom of `PROJECT_STATE.md`).
- **Single source of truth for tasks:** `TASKS.md` (top = next exact action).
  Decisions: `DECISIONS.md`. Lean handover: `PROJECT_STATE.md`. Don't re-scatter
  tasks into other files.
- **Hard rules** (`AGENTS.md`): no GPL source in product dirs, no invented
  crypto, no hard-coded interface names, never log/store raw access keys or leak
  host identifiers, British English in docs.

## Prerequisites

```bash
go version      # Go toolchain (module: gemina, see go.mod)
python3 --version   # used by the driver only to validate JSON
```

## Run (agent path) — the driver

```bash
# App smoke: build, run both subcommands, assert no MAC/IPv4 leak in the report
.claude/skills/run-geminactl/smoke.sh

# Full per-cycle gate: app smoke + go vet + go test -race + dedup benchmark
# + gofmt + docs-check + licence-check
.claude/skills/run-geminactl/smoke.sh verify
```

`verify` is the one command to run before committing a cycle — it is the whole
validation suite the project expects, so you don't reconstruct it from the
Makefile and `AGENTS.md` each time. Exit 0 = green; the first failing assertion
prints `FAIL <what>` and stops.

## Run the CLI directly

```bash
go build -o /tmp/geminactl ./cmd/geminactl
/tmp/geminactl                 # -> geminactl:stage-1-probe
/tmp/geminactl darwin-evidence # -> redacted Stage 1 evidence JSON
/tmp/geminactl preflight       # -> one-line compatibility verdict + next step
/tmp/geminactl preflight -json # -> redacted compatibility report (app/website)
/tmp/geminactl probe -h        # -> per-path UDP probe (incl. -interface2 dual-path)
```

`darwin-evidence` reports raw path evidence; `preflight` is the user-facing
**compatibility verdict** (`internal/diagnostics/compatibility.go`): supported /
needs-android / needs-wifi / needs-both / unsupported-macos, with one actionable
next step. Key rule (all-Android, minimal friction): an RNDIS tether function
present ⇒ **supported**, because the app's bundled userspace driver
(`research/usb-rndis-spike/`, skill `userspace-rndis-dataplane`) drives any
Android; native NCM is supported without the driver.

With an Android phone USB-tethered, `darwin-evidence` reports a usable `wi-fi`
candidate plus a `device_functions` `android-rndis` entry; `preflight` returns
`supported`. Without tethering, `darwin-evidence` is `incomplete` and `preflight`
returns `needs-android` — expected, not a failure.

## Gotchas

- **`darwin-evidence` "incomplete" is normal** off the train / without USB
  tethering. The tool reports candidates honestly; it never fakes success. Its
  `"claim"` is always `diagnostic-only-not-path-success`.
- **Redaction is an invariant, not a nicety.** The driver greps the output for
  MAC and IPv4 patterns and fails if either appears. If you add evidence fields,
  keep them coarse tokens (`present`/`absent`/`wifi`/`android-rndis`) — never raw
  IPs, MACs, serials or IORegistry product strings.
- **Evidence vocabulary is centralised.** Producers (`live_evidence.go`) and the
  consumer (`evidence.go`) share the `EvidenceKey*`/`EvidenceValue*` constants.
  Change a token in one place only; `evidence_test.go` fails if the sides drift.
- **`go test` caches.** Use `go test -race ./...` (the driver does) or
  `go clean -testcache` if you need a forced re-run.
- **No GUI, no Xcode project yet.** The Swift scaffold (`apps/macos`) is
  build-only via SwiftPM; there's nothing to screenshot.

## Troubleshooting

- `python3: command not found` → install it, or drop the JSON-validity step; the
  redaction and shape checks use `grep` only.
- `darwin-evidence` non-zero exit → it surfaces a real collection error
  (`networksetup`/`ioreg` failure); the driver aborts via `set -e`. Run the bare
  command to see the stderr message.
