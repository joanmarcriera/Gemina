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
#      The TUN->WAN ACCEPT lives INSIDE the egress chain (below), after DROP rules
#      that stop tunnel clients reaching cloud metadata (link-local 169.254/16),
#      RFC1918, 100.64/10, loopback, reserved space and IPv6 ULA/link-local.
#      With no DOCKER-USER chain the script FAILS rather than forward unfiltered
#      (set HOOK_CHAIN=FORWARD on a host without Docker).
#   6. Drops tunnel traffic addressed to the exit host itself (INPUT), so
#      clients cannot reach host services (sshd, metrics, ...) via 10.99.0.1 or
#      the host's public address through the tunnel.
#
# After this, start the gateway with deploy/systemd/gemina-gateway-data.service.
#
# Usage:
#   sudo scripts/setup-exit-host.sh                 # WAN_IF = default-route iface
#   sudo WAN_IF=enp0s3 scripts/setup-exit-host.sh
#   sudo WAN_IF=enp1s0 POOL=10.99.0.0/16 GW_ADDR=10.99.0.1/16 scripts/setup-exit-host.sh
#   sudo RUN_AS=1000:1000 scripts/setup-exit-host.sh   # match GEMINA_GATEWAY_RUN_AS
set -euo pipefail

TUN="${GEMINA_TUN:-gemina0}"
POOL="${POOL:-10.99.0.0/16}"             # must match GEMINA_GATEWAY_POOL
# Gateway's own address inside the pool: the pool's .1 (internal/exit reserves
# it), derived from POOL (given as its network address) unless set explicitly.
if [[ -z "${GW_ADDR:-}" ]]; then
  IFS=. read -r _o1 _o2 _o3 _o4 <<<"${POOL%/*}"
  GW_ADDR="$_o1.$_o2.$_o3.$((_o4 + 1))/${POOL#*/}"
fi
TUN_MTU="${TUN_MTU:-1280}"               # must match GEMINA_GATEWAY_TUN_MTU
RUN_AS="${RUN_AS:-65532:65532}"         # uid[:gid] the gateway drops to (GEMINA_GATEWAY_RUN_AS)
CONTAINER_UID="${CONTAINER_UID:-${RUN_AS%%:*}}"
if [[ "$RUN_AS" == *:* ]]; then CONTAINER_GID="${CONTAINER_GID:-${RUN_AS#*:}}"; else CONTAINER_GID="${CONTAINER_GID:-$CONTAINER_UID}"; fi
STATE_DIR="${STATE_DIR:-/var/lib/gemina}"
WAN_IF="${WAN_IF:-}"
HOOK_CHAIN="${HOOK_CHAIN:-DOCKER-USER}"  # FORWARD-path chain we own rules in

say() { printf '\n== %s\n' "$1"; }

# Egress deny-list for the Gemina exit host.
#
# Tunnel clients are untrusted. Once the FORWARD ACCEPT for the TUN exists they
# could otherwise reach, THROUGH the exit host, the cloud metadata service
# (on link-local 169.254/16) and any private network the host can route to
# (its own VCN/VPC, RFC1918, CGNAT, loopback, IPv6 ULA/link-local), and send
# multicast.
#
# Design (order-safe and gap-free across re-runs, even if WAN_IF/TUN change):
# the DROPs AND the TUN->WAN ACCEPT live in one of two dedicated chains,
# GEMINA-EGRESS-A / -B (DROPs first, then `-o $WAN -j ACCEPT`, then RETURN).
# Each run builds the rules into the chain that is NOT currently jumped to, so
# the live chain is never flushed in place. Only when the new chain is complete
# (ends in RETURN) is `-i $TUN -j <new>` inserted at position 1 of $HOOK_CHAIN;
# then the old jump(s) are deleted and the old chain flushed and removed. At
# every instant the jump points at a complete chain and no TUN ACCEPT exists
# outside it, so tunnel traffic can never be accepted unfiltered — including
# when WAN_IF changes (the old WAN's ACCEPT disappears with the old chain).
# Stale rules from earlier designs (per-range DROPs, top-level TUN ACCEPTs,
# the legacy single GEMINA-EGRESS chain) are cleaned up.
#
# Functions (take the tool name as $1):
#   exit_egress_v4_ranges / exit_egress_v6_ranges   print the denied CIDRs
#   exit_egress_apply <iptables|ip6tables> <tun> [wan]
#                                  build new chain (+ ACCEPT out wan), swap jump
#   exit_egress_remove <iptables|ip6tables>         undo (jumps + chains)
EGRESS_CHAINS=(GEMINA-EGRESS-A GEMINA-EGRESS-B GEMINA-EGRESS)   # last = legacy name

exit_egress_v4_ranges() {
  # 240.0.0.0/4 (reserved) also covers 255.255.255.255.
  printf '%s\n' 0.0.0.0/8 10.0.0.0/8 100.64.0.0/10 127.0.0.0/8 \
    169.254.0.0/16 172.16.0.0/12 192.0.0.0/24 192.168.0.0/16 198.18.0.0/15 \
    224.0.0.0/4 240.0.0.0/4
}

exit_egress_v6_ranges() {
  # fc00::/7 covers the fd00::/8 ULA space actually used (incl. cloud IPv6 metadata).
  # ::ffff:0:0/96 = IPv4-mapped (would bypass the v4 list).
  printf '%s\n' ::/128 ::1/128 ::ffff:0:0/96 fc00::/7 fe80::/10 ff00::/8
}

# Print the egress chain $HOOK_CHAIN currently jumps to (first match), if any.
# awk reads all input (no early exit), so pipefail never sees a SIGPIPE, and a
# missing chain yields empty output rather than aborting the caller under set -e.
exit_egress_current() {
  { "$1" -S "$HOOK_CHAIN" 2>/dev/null || true; } |
    awk '!f && match($0, / -j GEMINA-EGRESS[A-Z-]*$/) { print substr($0, RSTART + 4); f = 1 }'
}

# Delete every $HOOK_CHAIN rule matching ERE $2, except the exact `-S` line $3.
exit_hook_prune() {
  local tool="$1" re="$2" keep="${3:-}" line
  local -a rule
  while IFS= read -r line; do
    [[ -n "$line" && "$line" != "$keep" ]] || continue
    read -ra rule <<<"$line"               # -A <chain> <spec...>
    "$tool" -D "$HOOK_CHAIN" "${rule[@]:2}" || true
  done < <({ "$tool" -S "$HOOK_CHAIN" 2>/dev/null || true; } | grep -E -- "$re" || true)
}

# Delete every jump from $HOOK_CHAIN to an egress chain other than $2 (may be empty).
exit_egress_unjump_except() {
  local tool="$1" keep="${2:-}" c
  for c in "${EGRESS_CHAINS[@]}"; do
    [[ "$c" == "$keep" ]] && continue
    exit_hook_prune "$tool" " -j $c\$"
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
  local tool="$1" tun="$2" wan="${3:-}" ranges cidr cur new
  if [[ "$tool" == ip6tables ]]; then ranges="$(exit_egress_v6_ranges)"; else ranges="$(exit_egress_v4_ranges)"; fi
  cur="$(exit_egress_current "$tool")"
  if [[ "$cur" == GEMINA-EGRESS-A ]]; then new=GEMINA-EGRESS-B; else new=GEMINA-EGRESS-A; fi
  # The target chain is not referenced (it is not the live one): safe to reset.
  exit_egress_unjump_except "$tool" "$cur"   # drop any stray jump to $new before reuse
  "$tool" -N "$new" 2>/dev/null || "$tool" -F "$new"
  while IFS= read -r cidr; do
    "$tool" -A "$new" -d "$cidr" -j DROP
    # Stale per-range DROPs from the first (fail-open) design.
    while "$tool" -D "$HOOK_CHAIN" -i "$tun" -d "$cidr" -j DROP 2>/dev/null; do :; done
  done <<<"$ranges"
  # Forward to the WAN only AFTER every DROP; nothing else accepts TUN traffic.
  [[ -z "$wan" ]] || "$tool" -A "$new" -o "$wan" -j ACCEPT
  "$tool" -A "$new" -j RETURN
  # Complete chain: make it live FIRST, then retire the old one.
  "$tool" -I "$HOOK_CHAIN" 1 -i "$tun" -j "$new"
  exit_egress_unjump_except "$tool" "$new"
  exit_egress_drop_chains_except "$tool" "$new"
  # Top-level TUN->any ACCEPTs from earlier designs (now inside the chain).
  exit_hook_prune "$tool" "^-A $HOOK_CHAIN -i $tun -o [^ ]+ -j ACCEPT\$"
}

exit_egress_remove() {
  local tool="$1"
  exit_egress_unjump_except "$tool" ""
  exit_egress_drop_chains_except "$tool" ""
}

# Tests source this file for the functions above only (scripts/test-exit-egress.sh).
[[ -n "${GEMINA_SETUP_LIB_ONLY:-}" ]] && return 0

die() { echo "error: $*" >&2; exit 1; }
for id in "$CONTAINER_UID" "$CONTAINER_GID"; do
  [[ "$id" =~ ^[0-9]+$ && "$id" -gt 0 ]] || die "RUN_AS/CONTAINER_UID must be a numeric non-root uid[:gid], got '$RUN_AS'"
done

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

say "1/6 enable IPv4 forwarding"
echo 'net.ipv4.ip_forward=1' > /etc/sysctl.d/99-gemina-exit.conf
sysctl -q -w net.ipv4.ip_forward=1

say "2/6 persistent TUN $TUN (owner uid $CONTAINER_UID)"
if ip link show "$TUN" >/dev/null 2>&1; then
  # A persistent TUN keeps its owner; recreate it if RUN_AS changed since.
  owner="$(cat "/sys/class/net/$TUN/owner" 2>/dev/null || true)"
  if [[ "$owner" == "$CONTAINER_UID" ]]; then
    echo "  $TUN already exists (owner $owner); leaving it"
  else
    echo "  $TUN owner is '${owner}', want $CONTAINER_UID; recreating"
    ip link del "$TUN"
  fi
fi
ip link show "$TUN" >/dev/null 2>&1 || ip tuntap add dev "$TUN" mode tun user "$CONTAINER_UID"

say "3/6 address $GW_ADDR, mtu $TUN_MTU on $TUN + up"
ip addr replace "$GW_ADDR" dev "$TUN"
ip link set "$TUN" mtu "$TUN_MTU" up

say "4/6 MASQUERADE $POOL out $WAN_IF"
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

say "5/6 egress deny list + $TUN <-> $WAN_IF forward accepts in $HOOK_CHAIN"
# Fail closed: forwarding without the deny list would expose metadata + private ranges.
command -v iptables >/dev/null 2>&1 || die "iptables not found; refusing to forward without the egress deny list"
iptables -n -L "$HOOK_CHAIN" >/dev/null 2>&1 || \
  die "no $HOOK_CHAIN chain (Docker not running?). Start Docker and re-run, or set HOOK_CHAIN=FORWARD on a host without Docker."
# Return traffic only (conntrack); stale copies for a previous WAN_IF are pruned.
ret=(-i "$WAN_IF" -o "$TUN" -m conntrack --ctstate "RELATED,ESTABLISHED" -j ACCEPT)
iptables -C "$HOOK_CHAIN" "${ret[@]}" 2>/dev/null || iptables -I "$HOOK_CHAIN" 1 "${ret[@]}"
exit_hook_prune iptables " -o $TUN .*RELATED,ESTABLISHED -j ACCEPT\$" "-A $HOOK_CHAIN ${ret[*]}"
# DROPs then the TUN->WAN ACCEPT, in a chain swapped in atomically (see top).
exit_egress_apply iptables "$TUN" "$WAN_IF"
if command -v ip6tables >/dev/null 2>&1 && ip6tables -n -L "$HOOK_CHAIN" >/dev/null 2>&1; then
  exit_egress_apply ip6tables "$TUN"   # deny only: the pool is IPv4
fi
iptables -S "$HOOK_CHAIN" | sed 's/^/  /'

say "6/6 drop tunnel traffic addressed to this host (INPUT)"
# Clients reach the internet via FORWARD; nothing on the host is meant for them
# (DNS defaults to a public resolver), so sshd/metrics/etc. stay unreachable.
for tool in iptables ip6tables; do
  command -v "$tool" >/dev/null 2>&1 || continue
  "$tool" -C INPUT -i "$TUN" -j DROP 2>/dev/null || "$tool" -I INPUT 1 -i "$TUN" -j DROP
done

say "state dir $STATE_DIR (writable by $CONTAINER_UID:$CONTAINER_GID for the Ed25519 identity)"
mkdir -p "$STATE_DIR"
chown "$CONTAINER_UID:$CONTAINER_GID" "$STATE_DIR"

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
         iptables -D $HOOK_CHAIN <the $TUN return ACCEPT and the GEMINA-EGRESS-A/-B jump> ;
         iptables -F GEMINA-EGRESS-A/-B ; iptables -X GEMINA-EGRESS-A/-B (whichever exists) ;
         iptables -D INPUT -i $TUN -j DROP  (same three for ip6tables) ; rm /etc/sysctl.d/99-gemina-exit.conf
EOF
