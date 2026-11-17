#!/usr/bin/env bash
# Test: scripts/licence-check.sh passes on a clean fixture, still fails (and
# prints the offending lines) on pending or malformed pins, and never writes
# to fixed, predictable /tmp paths that another local user could pre-plant.
# Usage: scripts/test-licence-check.sh   (no root needed)
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fail() { echo "FAIL: $*" >&2; exit 1; }

grep -q '/tmp/' "$here/licence-check.sh" && fail "licence-check.sh references a fixed /tmp path"

fixture() {
	local dir="$1" pin="$2"
	mkdir -p "$dir/research" "$dir/docs/legal"
	for f in docs/legal/upstream-licences.md docs/legal/code-provenance.md docs/legal/dependency-inventory.md NOTICE; do
		: >"$dir/$f"
	done
	printf 'sources:\n  - name: example\n    %s\n' "$pin" >"$dir/research/upstream-manifest.yaml"
}

run() { (cd "$1" && sh "$here/licence-check.sh") 2>&1; }

fixture "$tmp/ok" "commit: $(printf 'a%.0s' $(seq 40))"
run "$tmp/ok" | grep -q 'licence/provenance structure ok' || fail "clean fixture did not pass"

fixture "$tmp/pending" "pin_status: pending"
if out="$(run "$tmp/pending")"; then fail "pending pin passed"; fi
grep -q 'pin_status: pending' <<<"$out" || fail "pending pin line not reported"

fixture "$tmp/short" "commit: abc123"
if out="$(run "$tmp/short")"; then fail "short commit pin passed"; fi
grep -q 'commit: abc123' <<<"$out" || fail "invalid pin line not reported"

echo "PASS licence-check"
