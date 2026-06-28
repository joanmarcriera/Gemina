#!/usr/bin/env bash
# DRAFT — host-side setup for the Stage-2 data + exit gateway.
#
# STATUS: written 2026-06-28, NOT yet validated on hardware. Validate during
# WS-F (docs/superpowers/plans/2026-06-26-phase3-wifi-tunnel.md) and only then
# drop this header. The commands are standard Linux TUN-exit setup but the exact
# WAN interface name, firewall backend (nftables vs iptables) and uid must be
# confirmed on the target box.
#
# What it does (idempotent — safe to re-run):
#   1. Enables IPv4 forwarding (persisted via sysctl.d).
#   2. Creates a PERSISTENT TUN device (gemina0) owned by the container uid, so
#      the gateway opens an already-configured interface (deterministic ordering:
#      the host configures addr/route/NAT before the container starts).
#   3. Gives gemina0 the gateway-side address in the lease pool and brings it up.
#   4. Adds a MASQUERADE NAT rule for the pool out the WAN interface (the Go exit
#      engine deliberately does NOT touch the firewall — see internal/exit/nat_linux.go).
#
# After this, start the gateway with deploy/systemd/gemina-gateway-data.service.
#
# Usage:
#   sudo WAN_IF=eth0 scripts/setup-exit-host.sh
#   sudo WAN_IF=enp1s0 POOL=10.99.0.0/16 GW_ADDR=10.99.0.1/16 scripts/setup-exit-host.sh
set -euo pipefail

TUN="${GEMINA_TUN:-gemina0}"
POOL="${POOL:-10.99.0.0/16}"             # must match GEMINA_GATEWAY_POOL
GW_ADDR="${GW_ADDR:-10.99.0.1/16}"       # gateway's own address inside the pool
CONTAINER_UID="${CONTAINER_UID:-65532}"  # distroless :nonroot uid (see gateway.Dockerfile)
STATE_DIR="${STATE_DIR:-/var/lib/gemina}"
WAN_IF="${WAN_IF:-}"

say() { printf '\n== %s\n' "$1"; }

if [[ $EUID -ne 0 ]]; then
  echo "error: run as root (needs sysctl, ip, and firewall changes)" >&2
  exit 1
fi
if [[ -z "$WAN_IF" ]]; then
  # Best-effort default: the interface holding the default route.
  WAN_IF="$(ip route show default 2>/dev/null | awk '/default/ {print $5; exit}')"
  [[ -n "$WAN_IF" ]] || { echo "error: set WAN_IF=<egress interface>" >&2; exit 1; }
  echo "WAN_IF not set; defaulting to default-route interface: $WAN_IF"
fi

say "1/4 enable IPv4 forwarding"
echo 'net.ipv4.ip_forward=1' > /etc/sysctl.d/99-gemina-exit.conf
sysctl -q -w net.ipv4.ip_forward=1

say "2/4 persistent TUN $TUN (owner uid $CONTAINER_UID)"
if ! ip link show "$TUN" >/dev/null 2>&1; then
  ip tuntap add dev "$TUN" mode tun user "$CONTAINER_UID"
else
  echo "  $TUN already exists; leaving it"
fi

say "3/4 address $GW_ADDR on $TUN + up"
ip addr replace "$GW_ADDR" dev "$TUN"
ip link set "$TUN" up

say "4/4 MASQUERADE $POOL out $WAN_IF"
if command -v nft >/dev/null 2>&1; then
  nft list table inet gemina >/dev/null 2>&1 || nft add table inet gemina
  nft 'list chain inet gemina postrouting' >/dev/null 2>&1 || \
    nft add chain inet gemina postrouting '{ type nat hook postrouting priority 100 ; }'
  # Flush our chain so re-runs do not stack duplicate rules.
  nft flush chain inet gemina postrouting
  nft add rule inet gemina postrouting ip saddr "$POOL" oifname "$WAN_IF" masquerade
else
  # iptables fallback: -C tests for the rule so we do not duplicate it.
  iptables -t nat -C POSTROUTING -s "$POOL" -o "$WAN_IF" -j MASQUERADE 2>/dev/null || \
    iptables -t nat -A POSTROUTING -s "$POOL" -o "$WAN_IF" -j MASQUERADE
fi

say "state dir $STATE_DIR (writable by uid $CONTAINER_UID for the Ed25519 identity)"
mkdir -p "$STATE_DIR"
chown "$CONTAINER_UID:$CONTAINER_UID" "$STATE_DIR"

cat <<EOF

Host prepared for the data + exit gateway.
  TUN:        $TUN  ($GW_ADDR)
  Pool:       $POOL  (NAT MASQUERADE out $WAN_IF)
  Identity:   $STATE_DIR/gateway-identity.key (created on first start)

Next:
  1. Install/start deploy/systemd/gemina-gateway-data.service (GEMINA_GATEWAY_EXIT=on).
  2. Read the printed base64 Ed25519 public_key from the gateway log — the app pins it.
  3. Open ingress UDP 51820 in the cloud firewall (VCN/security group) too.

To undo: ip link del $TUN ; remove the NAT rule ; rm /etc/sysctl.d/99-gemina-exit.conf
EOF
