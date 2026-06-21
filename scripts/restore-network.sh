#!/usr/bin/env bash
# Reassert a safe network state after (or during) a bonded-uplink test cycle:
# put the cabled management service back at the top of the service order so it
# owns the default route, then show the resulting default path so you can
# confirm your link to the outside world (and to Claude) is intact.
#
# Usage:
#   scripts/restore-network.sh                 # auto-detect a wired service
#   scripts/restore-network.sh "Service Name"  # pin a specific service first
#   MGMT_SERVICE="USB 10/100/1000 LAN" scripts/restore-network.sh
#
# Scope today: service-order reassertion + diagnostics (Stage 1 is socket-bind
# only, so there is no global route or tunnel to tear down yet). When the
# NEPacketTunnelProvider lands, add its disable step here.
set -euo pipefail

mgmt="${1:-${MGMT_SERVICE:-}}"

services="$(networksetup -listallnetworkservices | tail -n +2)"

if [ -z "$mgmt" ]; then
  # Prefer a wired service (Ethernet / Thunderbolt / LAN), never Wi-Fi.
  mgmt="$(printf '%s\n' "$services" \
    | grep -iE 'ethernet|thunderbolt|lan' \
    | grep -ivE 'bridge' \
    | head -n1 || true)"
fi

if [ -z "$mgmt" ]; then
  echo "could not auto-detect a wired management service. Available:" >&2
  printf '%s\n' "$services" >&2
  echo "re-run with the exact name, e.g.: scripts/restore-network.sh \"Thunderbolt Ethernet Slot 2\"" >&2
  exit 1
fi

echo "pinning management service first: $mgmt"
# Build a new order with the management service in front, others after.
rest="$(printf '%s\n' "$services" | grep -vxF "$mgmt" || true)"
# shellcheck disable=SC2086
IFS=$'\n' read -r -d '' -a ordered < <(printf '%s\n%s\0' "$mgmt" "$rest")
networksetup -ordernetworkservices "${ordered[@]}"

echo
echo "resulting default path:"
scutil --nwi | sed -n '1,12p'
echo
echo "if the primary interface above is your cable, your management link is safe."
