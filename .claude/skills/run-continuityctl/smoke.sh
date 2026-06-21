#!/usr/bin/env bash
# Driver for the continuity-vpn `continuityctl` CLI and the per-cycle gate.
#
#   smoke.sh            build + drive the CLI, assert redaction invariants
#   smoke.sh verify     the above, then the full validation gate
#                       (vet, race tests, bench, gofmt, docs/licence checks)
#
# No arguments needed beyond the optional `verify`. macOS host required for the
# darwin-evidence subcommand (it reads BSD interface state, networksetup, ioreg).
set -euo pipefail

UNIT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$UNIT"
BIN="$(mktemp -t continuityctl.XXXXXX)"
trap 'rm -f "$BIN"' EXIT

pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1" >&2; exit 1; }

echo "== build =="
go build -o "$BIN" ./cmd/continuityctl
pass "go build ./cmd/continuityctl"

echo "== drive: no-arg stage marker =="
got="$("$BIN")"
[ "$got" = "continuityctl:stage-1-probe" ] || fail "stage marker = '$got'"
pass "prints continuityctl:stage-1-probe"

echo "== drive: darwin-evidence =="
out="$("$BIN" darwin-evidence)"            # exits non-zero -> set -e aborts
pass "darwin-evidence exit 0"

printf '%s' "$out" | python3 -c 'import sys,json; json.load(sys.stdin)' \
  || fail "darwin-evidence output is not valid JSON"
pass "output is valid JSON"

for key in '"type"' '"stage"' '"claim"' '"classification_status"'; do
  printf '%s' "$out" | grep -q "$key" || fail "missing top-level key $key"
done
printf '%s' "$out" | grep -q '"claim": "diagnostic-only-not-path-success"' \
  || fail "missing diagnostic-only claim marker"
pass "report shape + diagnostic-only claim"

# Core privacy invariant: the report must never leak raw host identifiers.
if printf '%s' "$out" | grep -qiE '([0-9a-f]{2}:){5}[0-9a-f]{2}'; then
  fail "MAC address leaked into darwin-evidence output"
fi
if printf '%s' "$out" | grep -qE '[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}'; then
  fail "IPv4 address leaked into darwin-evidence output"
fi
pass "no MAC / IPv4 / dotted-quad leakage"

echo "== drive: unknown subcommand exits 2 =="
rc=0; "$BIN" bogus >/dev/null 2>&1 || rc=$?
[ "$rc" -eq 2 ] || fail "unknown subcommand exit = $rc, want 2"
pass "usage error exit code 2"

if [ "${1:-}" != "verify" ]; then
  echo "PASS (app smoke). Run 'smoke.sh verify' for the full gate."
  exit 0
fi

echo "== verify: go vet =="
go vet ./...
pass "go vet ./..."

echo "== verify: go test -race =="
go test -race ./...
pass "go test -race ./..."

echo "== verify: dedup benchmark =="
go test -run '^$' -bench BenchmarkWindowObserveSteadyState -benchmem ./internal/dedup
pass "dedup steady-state benchmark"

echo "== verify: gofmt =="
unformatted="$(gofmt -l internal cmd)"
[ -z "$unformatted" ] || fail "gofmt: $unformatted"
pass "gofmt clean"

echo "== verify: docs + licence checks =="
sh scripts/docs-check.sh
sh scripts/licence-check.sh
pass "docs-check + licence-check"

echo "PASS (app smoke + full gate)."
