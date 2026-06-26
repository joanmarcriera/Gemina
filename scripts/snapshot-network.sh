#!/usr/bin/env bash
# Capture a baseline of the Mac's network configuration BEFORE a test cycle, so
# any change made while experimenting with the bonded uplinks can be diffed and
# reverted. See docs/dev/test-environment.md for the operating protocol.
#
# Output goes to a git-ignored local file (.netcheck/baseline.txt by default).
# It contains raw host network state (addresses, routes) and must NEVER be
# committed — .netcheck/ is git-ignored for exactly this reason.
set -euo pipefail

UNIT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$UNIT/.netcheck"
OUT="${1:-$OUT_DIR/baseline.txt}"
mkdir -p "$(dirname "$OUT")"

{
  echo "# gemina network baseline"
  echo "# captured: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo
  echo "## network service order (the default-route priority list)"
  networksetup -listnetworkserviceorder
  echo
  echo "## hardware ports"
  networksetup -listallhardwareports
  echo
  echo "## interface flags + addresses"
  ifconfig -a
  echo
  echo "## routing table"
  netstat -rn
  echo
  echo "## network information (primary service / default path)"
  scutil --nwi
} >"$OUT"

echo "baseline written to $OUT"
echo "management tip: confirm your cabled service is FIRST in the service order"
echo "above; if not, run: scripts/restore-network.sh"
