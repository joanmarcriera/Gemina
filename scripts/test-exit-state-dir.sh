#!/usr/bin/env bash
# Test: exit_state_dir_apply hands STATE_DIR AND an existing gateway-identity.key to the
# RUN_AS uid:gid (the gateway reads the 0600 key only after dropping privileges), keeps
# the key's bytes and mode, works when no key exists yet, refuses a symlinked key, and
# never lets a key swapped for a symlink after the check redirect the root chown (TOCTOU).
# Usage: scripts/test-exit-state-dir.sh   (chown needs root; skips otherwise)
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
if [[ $EUID -ne 0 ]]; then echo "SKIP exit-state-dir (needs root for chown)"; exit 0; fi
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
# shellcheck source=/dev/null
GEMINA_SETUP_LIB_ONLY=1 source "$here/setup-exit-host.sh"
fail() { echo "FAIL: $*" >&2; exit 1; }
owner() { stat -c '%u:%g' "$1"; }

# Upgrade from a root-run gateway / changed RUN_AS: key owned by the old uid.
dir="$tmp/state"; mkdir -p "$dir"; key="$dir/gateway-identity.key"
head -c 64 /dev/urandom >"$key"; chmod 0600 "$key"; chown 0:0 "$key"
sum="$(sha256sum <"$key")"
exit_state_dir_apply "$dir" 65532 65532
[[ "$(owner "$dir")" == 65532:65532 ]] || fail "dir owner $(owner "$dir")"
[[ "$(owner "$key")" == 65532:65532 ]] || fail "key owner $(owner "$key"), want new RUN_AS"
[[ "$(stat -c '%a' "$key")" == 600 ]] || fail "key mode changed to $(stat -c '%a' "$key")"
[[ "$(sha256sum <"$key")" == "$sum" ]] || fail "key contents changed"
# RUN_AS changes again.
exit_state_dir_apply "$dir" 1000 1001
[[ "$(owner "$key")" == 1000:1001 ]] || fail "key not migrated on second RUN_AS change"

# Fresh host: no key yet (the gateway creates it on first start).
exit_state_dir_apply "$tmp/fresh" 65532 65532
[[ "$(owner "$tmp/fresh")" == 65532:65532 ]] || fail "fresh dir owner"
[[ ! -e "$tmp/fresh/gateway-identity.key" ]] || fail "key created by setup"

# A symlinked key must not redirect a root chown elsewhere.
target="$tmp/victim"; : >"$target"; chown 0:0 "$target"
mkdir -p "$tmp/link"; ln -s "$target" "$tmp/link/gateway-identity.key"
if exit_state_dir_apply "$tmp/link" 65532 65532 2>/dev/null; then fail "symlinked key accepted"; fi
[[ "$(owner "$target")" == 0:0 ]] || fail "symlink target was chowned"

# TOCTOU: the key is a regular file at check time, but the gateway uid (which owns the
# dir) swaps it for a symlink to a root-owned file right before the chown. Simulate the
# race by shadowing chown: on the key's chown, replace the key first, then chown.
victim="$tmp/root-owned"; : >"$victim"; chown 0:0 "$victim"
race="$tmp/race"; mkdir -p "$race"; rkey="$race/gateway-identity.key"
: >"$rkey"; chmod 0600 "$rkey"
# shellcheck disable=SC2329  # invoked indirectly by exit_state_dir_apply
chown() {
  if [[ "${*: -1}" == "$rkey" && ! -L "$rkey" ]]; then rm -f "$rkey"; ln -s "$victim" "$rkey"; fi
  command chown "$@"
}
exit_state_dir_apply "$race" 65532 65532 || true
unset -f chown
[[ -L "$rkey" ]] || fail "race simulation did not swap the key"
[[ "$(owner "$victim")" == 0:0 ]] || fail "swapped-in symlink target chowned to $(owner "$victim")"
echo "PASS exit-state-dir"
