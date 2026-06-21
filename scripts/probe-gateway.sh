#!/usr/bin/env bash
# End-to-end verification of a deployed Stage-1 gateway: send probes (including a
# deliberate duplicate) to HOST:PORT over a chosen interface using the real
# `continuityctl probe` subcommand, then show the gateway's decision logs SINCE a
# timestamp marker so stale scrollback can never be mistaken for fresh arrival.
#
#   scripts/probe-gateway.sh                       # host "oracle", port 51820, en0
#   GATEWAY_HOST=oracle GATEWAY_PORT=51820 GATEWAY_IFACE=en0 scripts/probe-gateway.sh
#
# Sends from this machine over the public internet, so both the host firewall
# and any cloud firewall (Oracle Cloud VCN security list) must allow ingress
# UDP on the port, with the ingress SOURCE port range set to "All".
set -euo pipefail

HOST="${GATEWAY_HOST:-oracle}"
PORT="${GATEWAY_PORT:-51820}"
IFACE="${GATEWAY_IFACE:-en0}"
UNIT="continuity-gateway.service"

UNIT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$UNIT_ROOT"

# Resolve the host's public address from the SSH config so probes traverse the
# real network path rather than the SSH tunnel.
TARGET="$(ssh -G "$HOST" 2>/dev/null | awk '/^hostname /{print $2; exit}')"
TARGET="${TARGET:-$HOST}"

bin="$(mktemp -t continuityctl.XXXXXX)"
trap 'rm -f "$bin"' EXIT
go build -o "$bin" ./cmd/continuityctl

MARK="$(ssh -o BatchMode=yes "$HOST" 'date -u +"%Y-%m-%d %H:%M:%S"')"

echo "== sending probes to $TARGET:$PORT over $IFACE (marker $MARK UTC)"
"$bin" probe -interface "$IFACE" -to "$TARGET:$PORT" -path wifi -count 2 -duplicate

echo "== gateway decisions since marker (fresh arrival only)"
sleep 2
ssh -o BatchMode=yes "$HOST" \
  "sudo journalctl -u '$UNIT' --since '$MARK' --no-pager | grep '\"msg\":\"probe\"' \
   || echo 'NO fresh probe decisions — check the VCN ingress rule (source port range must be All)'"
