#!/usr/bin/env bash
# Host-side setup for the Stage-2 data + exit gateway.
#
# Validated on the Oracle Cloud arm64 host (Oracle Linux 9, Docker 29,
# iptables-nft) during WS-F, 2026-09-25 (Vikunja #2609 / GitHub issue #5).
#
# What it does (idempotent — safe to re-run; the data unit re-runs it as
# ExecStartPre on every start, so a reboot or docker restart re-applies it):
#   1. Enables IPv4 forwarding (persisted via sysctl.d).
#   2. Creates a PERSISTENT TUN device (gemina0) owned by the container uid, so
#      the gateway opens an already-configured interface (deterministic ordering:
#      the host configures addr/MTU/route/NAT before the container starts).
#      A persistent TUN does not survive a reboot — hence the ExecStartPre.
#   3. Gives gemina0 the gateway-side address in the lease pool, sets its MTU
#      (1280, matches GEMINA_GATEWAY_TUN_MTU) and brings it up. With the MTU
#      already right, the gateway skips SIOCSIFMTU entirely.
#   4. Adds a MASQUERADE NAT rule for the pool out the WAN interface (the Go exit
#      engine deliberately does NOT touch the firewall — see internal/exit/nat_linux.go).
#   5. Lets pool traffic through Docker's FORWARD policy. Docker sets the
#      iptables FORWARD policy to DROP, so without ACCEPT rules in the
#      DOCKER-USER chain (the chain Docker reserves for operators and never
#      flushes) pool traffic never reaches WAN_IF even with MASQUERADE in place.
#      nftables cannot override this from our own table: a drop verdict in any
#      base chain is final, so the accept has to live in Docker's chain.
#      Before those ACCEPTs, DROP rules (below) stop tunnel
#      clients reaching cloud metadata (169.254.0.0/16), RFC1918, 100.64/10,
#      loopback and IPv6 ULA/link-local through the exit.
#
# After this, start the gateway with deploy/systemd/gemina-gateway-data.service.
#
# Usage:
#   sudo scripts/setup-exit-host.sh                 # WAN_IF = default-route iface
#   sudo WAN_IF=enp0s3 scripts/setup-exit-host.sh
#   sudo WAN_IF=enp1s0 POOL=10.99.0.0/16 GW_ADDR=10.99.0.1/16 scripts/setup-exit-host.sh
set -euo pipefail

TUN="${GEMINA_TUN:-gemina0}"
POOL="${POOL:-10.99.0.0/16}"             # must match GEMINA_GATEWAY_POOL
GW_ADDR="${GW_ADDR:-10.99.0.1/16}"       # gateway's own address inside the pool
TUN_MTU="${TUN_MTU:-1280}"               # must match GEMINA_GATEWAY_TUN_MTU
CONTAINER_UID="${CONTAINER_UID:-65532}"  # uid the gateway drops to (GEMINA_GATEWAY_RUN_AS)
STATE_DIR="${STATE_DIR:-/var/lib/gemina}"
WAN_IF="${WAN_IF:-}"

say() { printf '\n== %s\n' "$1"; }

# Egress deny-list for the Gemina exit host.
#
# Tunnel clients are untrusted. Once the FORWARD ACCEPT for the TUN exists they
# could otherwise reach, THROUGH the exit host, the cloud metadata service
# (169.254.169.254, link-local) and any private network the host can route to
# (its own VCN/VPC, RFC1918, CGNAT, loopback, IPv6 ULA/link-local), and send
# multicast.
#
# Design (order-safe and gap-free across re-runs, even if WAN_IF/TUN change):
# the DROPs live in one of two dedicated chains, GEMINA-EGRESS-A / -B. Each run
# builds the rules into the chain that is NOT currently jumped to, so the live
# chain is never flushed in place. Only when the new chain is complete (ends in
# RETURN) is `-i $TUN -j <new>` inserted at position 1 of DOCKER-USER (after the
# ACCEPTs); then the old jump(s) are deleted and the old chain flushed and
# removed. At every instant the jump points at a complete chain, so there is no
# window where tunnel traffic reaches the ACCEPTs unfiltered. Stale per-range
# DROPs from the earliest design and the legacy single GEMINA-EGRESS chain are
# cleaned up.
#
# Functions (take the tool name as $1):
#   exit_egress_v4_ranges / exit_egress_v6_ranges   print the denied CIDRs
#   exit_egress_apply <iptables|ip6tables> <tun>    build new chain, swap jump
#   exit_egress_remove <iptables|ip6tables>         undo (jumps + chains)
# Callers must add their ACCEPT rules FIRST, then call exit_egress_apply.
EGRESS_CHAINS=(GEMINA-EGRESS-A GEMINA-EGRESS-B GEMINA-EGRESS)   # last = legacy name

exit_egress_v4_ranges() {
  printf '%s\n' 0.0.0.0/8 10.0.0.0/8 100.64.0.0/10 127.0.0.0/8 \
    169.254.0.0/16 172.16.0.0/12 192.168.0.0/16 224.0.0.0/4
}

exit_egress_v6_ranges() {
  # fc00::/7 covers the fd00::/8 ULA space actually used (incl. cloud IPv6 metadata).
  printf '%s\n' ::1/128 fc00::/7 fe80::/10 ff00::/8
}

# Print the egress chain DOCKER-USER currently jumps to (first match), if any.
exit_egress_current() {
  "$1" -S DOCKER-USER 2>/dev/null | sed -n 's/^.* -j \(GEMINA-EGRESS[A-Z-]*\)$/\1/p' | head -1
}

# Delete every jump from DOCKER-USER to an egress chain other than $2 (may be empty).
exit_egress_unjump_except() {
  local tool="$1" keep="${2:-}" line c
  local -a rule
  for c in "${EGRESS_CHAINS[@]}"; do
    [[ "$c" == "$keep" ]] && continue
    while line="$("$tool" -S DOCKER-USER 2>/dev/null | grep -m1 -e "-j $c\$")"; do
      read -ra rule <<<"$line"             # -A DOCKER-USER <spec...>
      "$tool" -D DOCKER-USER "${rule[@]:2}" || break
    done
  done
}

# Flush and delete every egress chain except $2 (call only after unjumping them).
exit_egress_drop_chains_except() {
  local tool="$1" keep="${2:-}" c
  for c in "${EGRESS_CHAINS[@]}"; do
    [[ "$c" == "$keep" ]] && continue
    if "$tool" -F "$c" 2>/dev/null; then "$tool" -X "$c" 2>/dev/null || true; fi
  done
}

exit_egress_apply() {
  local tool="$1" tun="$2" ranges cidr cur new
  if [[ "$tool" == ip6tables ]]; then ranges="$(exit_egress_v6_ranges)"; else ranges="$(exit_egress_v4_ranges)"; fi
  cur="$(exit_egress_current "$tool")"
  if [[ "$cur" == GEMINA-EGRESS-A ]]; then new=GEMINA-EGRESS-B; else new=GEMINA-EGRESS-A; fi
  # The target chain is not referenced (it is not the live one): safe to reset.
  exit_egress_unjump_except "$tool" "$cur"   # drop any stray jump to $new before reuse
  "$tool" -N "$new" 2>/dev/null || "$tool" -F "$new"
  while IFS= read -r cidr; do
    "$tool" -A "$new" -d "$cidr" -j DROP
    # Stale per-range DROPs from the first (fail-open) design.
    while "$tool" -D DOCKER-USER -i "$tun" -d "$cidr" -j DROP 2>/dev/null; do :; done
  done <<<"$ranges"
  "$tool" -A "$new" -j RETURN
  # Complete chain: make it live FIRST, then retire the old one.
  "$tool" -I DOCKER-USER 1 -i "$tun" -j "$new"
  exit_egress_unjump_except "$tool" "$new"
  exit_egress_drop_chains_except "$tool" "$new"
}

exit_egress_remove() {
  local tool="$1"
  exit_egress_unjump_except "$tool" ""
  exit_egress_drop_chains_except "$tool" ""
}

# Tests source this file for the functions above only (scripts/test-exit-egress.sh).
[[ -n "${GEMINA_SETUP_LIB_ONLY:-}" ]] && return 0

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

say "1/5 enable IPv4 forwarding"
echo 'net.ipv4.ip_forward=1' > /etc/sysctl.d/99-gemina-exit.conf
sysctl -q -w net.ipv4.ip_forward=1

say "2/5 persistent TUN $TUN (owner uid $CONTAINER_UID)"
if ! ip link show "$TUN" >/dev/null 2>&1; then
  ip tuntap add dev "$TUN" mode tun user "$CONTAINER_UID"
else
  echo "  $TUN already exists; leaving it"
fi

say "3/5 address $GW_ADDR, mtu $TUN_MTU on $TUN + up"
ip addr replace "$GW_ADDR" dev "$TUN"
ip link set "$TUN" mtu "$TUN_MTU" up

say "4/5 MASQUERADE $POOL out $WAN_IF"
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

say "5/5 allow $TUN <-> $WAN_IF through Docker's FORWARD policy (DOCKER-USER)"
if command -v iptables >/dev/null 2>&1 && iptables -n -L DOCKER-USER >/dev/null 2>&1; then
  # -C tests for each rule first, so re-runs never stack duplicates. -I inserts
  # at the top, ahead of any operator DROP rules later in the chain.
  iptables -C DOCKER-USER -i "$TUN" -o "$WAN_IF" -j ACCEPT 2>/dev/null || \
    iptables -I DOCKER-USER 1 -i "$TUN" -o "$WAN_IF" -j ACCEPT
  iptables -C DOCKER-USER -i "$WAN_IF" -o "$TUN" -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || \
    iptables -I DOCKER-USER 1 -i "$WAN_IF" -o "$TUN" -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
  # Deny metadata + private ranges BEFORE the ACCEPTs: jump inserted last, at the top.
  exit_egress_apply iptables "$TUN"
  if command -v ip6tables >/dev/null 2>&1 && ip6tables -n -L DOCKER-USER >/dev/null 2>&1; then
    exit_egress_apply ip6tables "$TUN"
  fi
  iptables -S DOCKER-USER | sed 's/^/  /'
else
  # No Docker FORWARD chain yet: either Docker is not installed (FORWARD is not
  # forced to DROP) or it has not started. The data unit orders itself
  # After=docker.service, so under systemd the chain exists by now.
  echo "  no DOCKER-USER chain (Docker not running?); skipping — re-run after Docker starts"
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

To undo: ip link del $TUN ; nft delete table inet gemina (or the iptables NAT rule) ;
         iptables -D DOCKER-USER <the $TUN ACCEPT rules and the GEMINA-EGRESS-A/-B jump> ;
         iptables -F GEMINA-EGRESS-A/-B ; iptables -X GEMINA-EGRESS-A/-B (whichever exists; same for ip6tables) ; rm /etc/sysctl.d/99-gemina-exit.conf
EOF
