#!/usr/bin/env bash
# Test: the GEMINA-EGRESS-A/-B jump is rule 1 of DOCKER-USER, the chain holds the
# DROPs and then the only TUN->WAN ACCEPT, it is rebuilt each run into the idle twin and
# swapped in, so the jump ALWAYS points at a complete chain and no top-level TUN ACCEPT
# ever sits above it (both checked after every iptables call by the stub: no fail-open
# window), jumps never duplicate, stale/legacy rules are removed (incl. the old WAN's
# ACCEPT when WAN_IF changes), reserved/multicast/v4-mapped ranges are denied, and a
# duplicate jump does not abort under pipefail. Fake iptables models chains as files.
# Usage: scripts/test-exit-egress.sh   (no root needed)
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
for t in iptables ip6tables; do
cat >"$tmp/$t" <<STUB
#!/usr/bin/env bash
d="$tmp/state/$t"; mkdir -p "\$d"
op="\$1"; chain="\$2"; shift 2; f="\$d/\$chain"
case "\$op" in
  -N) [[ -e \$f ]] && exit 1; : >"\$f" ;;
  -F) [[ -e \$f ]] || exit 1; : >"\$f" ;;
  -X) if grep -qE -- " -j \$chain\$" "$tmp/state/$t/DOCKER-USER" 2>/dev/null; then echo "FAIL: delete of referenced chain \$chain" >>"$tmp/violations"; fi; rm -f "\$f" ;;
  -C) grep -qxF -- "\$*" "\$f" ;;
  -A) echo "\$*" >>"\$f" ;;
  -D) grep -qxF -- "\$*" "\$f" || exit 1
      awk -v r="\$*" '!d && \$0==r {d=1; next} {print}' "\$f" >"\$f.n"; mv "\$f.n" "\$f" ;;
  -S) [[ -e \$f ]] && sed "s/^/-A \$chain /" "\$f" ;;
  -I) pos="\$1"; shift; touch "\$f"
      awk -v p=\$pos -v r="\$*" 'NR==p{print r} {print} END{if(NR<p)print r}' "\$f" >"\$f.n"; mv "\$f.n" "\$f" ;;
esac
rc=\$?   # the stub's exit status is the operation's, not the invariant checks
# Invariant after every call: each DOCKER-USER jump targets a COMPLETE chain (ends in RETURN).
for j in \$(sed -n 's/^.* -j \(GEMINA-EGRESS[A-Z-]*\)\$/\1/p' "\$d/DOCKER-USER" 2>/dev/null); do
  [[ "\$(tail -1 "\$d/\$j" 2>/dev/null)" == "-j RETURN" ]] || echo "FAIL: jump to incomplete chain \$j after \$op" >>"$tmp/violations"
done
# ...and no top-level TUN->WAN ACCEPT is evaluated before the egress jump.
awk '/ -j GEMINA-EGRESS/ {exit} /^-i gemina0 -o [^ ]+ -j ACCEPT\$/ {print "FAIL: TUN ACCEPT above the egress jump"}' \
  "\$d/DOCKER-USER" 2>/dev/null >>"$tmp/violations"
exit "\$rc"

STUB
chmod +x "$tmp/$t"; done
export PATH="$tmp:$PATH"
# shellcheck source=/dev/null
GEMINA_SETUP_LIB_ONLY=1 source "$here/setup-exit-host.sh"
fail() { echo "FAIL: $*" >&2; exit 1; }
: >"$tmp/violations"

for t in iptables ip6tables; do
  du="$tmp/state/$t/DOCKER-USER"
  # Simulate the old fail-open designs: stale per-range DROP and a top-level TUN ACCEPT
  # in DOCKER-USER, and the legacy single GEMINA-EGRESS chain (complete) with its jump.
  "$t" -N GEMINA-EGRESS; "$t" -A GEMINA-EGRESS -d 192.0.2.4/32 -j DROP; "$t" -A GEMINA-EGRESS -j RETURN
  "$t" -I DOCKER-USER 1 -i gemina0 -j GEMINA-EGRESS
  "$t" -A DOCKER-USER -i gemina0 -o eth9 -j ACCEPT     # legacy: below the jump
  "$t" -A DOCKER-USER -i gemina0 -d "$([[ $t == ip6tables ]] && echo fe80::/10 || echo 10.0.0.0/8)" -j DROP
  prev=""
  for wan in eth0 ens3 eth0; do   # re-runs, WAN_IF changes
    exit_egress_apply "$t" gemina0 "$wan"
    [[ "$(head -1 "$du")" == "-i gemina0 -j GEMINA-EGRESS-"[AB] ]] || fail "$t/$wan: jump not rule 1"
    cur="$(exit_egress_current "$t")"
    [[ "$cur" != "$prev" ]] || fail "$t/$wan: live chain did not alternate"
    prev="$cur"
    [[ $(grep -c GEMINA-EGRESS "$du") -eq 1 ]] || fail "$t/$wan: duplicate jumps"
    grep -q -- "-j DROP" "$du" && fail "$t/$wan: stale DROP left in DOCKER-USER"
    grep -q -- "-j ACCEPT" "$du" && fail "$t/$wan: top-level TUN ACCEPT left in DOCKER-USER"
    # ACCEPT out the CURRENT wan only, after every DROP, right before RETURN.
    [[ "$(tail -2 "$tmp/state/$t/$cur" | head -1)" == "-o $wan -j ACCEPT" ]] || fail "$t/$wan: ACCEPT not last before RETURN"
    [[ $(grep -c -- "-j ACCEPT" "$tmp/state/$t/$cur") -eq 1 ]] || fail "$t/$wan: stale WAN ACCEPT kept"
    [[ "$(tail -1 "$tmp/state/$t/$cur")" == "-j RETURN" ]] || fail "$t: chain must end in RETURN"
    # Only the live chain remains (idle twin and legacy chain deleted).
    n=$(find "$tmp/state/$t" -name "GEMINA-EGRESS*" | wc -l); [[ $n -eq 1 ]] || fail "$t/$wan: $n egress chains left"
  done
  # Junk added to the idle twin must not survive the next run.
  idle=GEMINA-EGRESS-A; [[ "$cur" == GEMINA-EGRESS-A ]] && idle=GEMINA-EGRESS-B
  { echo "-d 203.0.113.8/32 -j ACCEPT"; } >"$tmp/state/$t/$idle"; exit_egress_apply "$t" gemina0 eth0
  grep -rq 203.0.113.8 "$tmp/state/$t" && fail "$t: chain not rebuilt"
  # A stray duplicate jump must not abort exit_egress_current (pipefail/SIGPIPE).
  cur="$(exit_egress_current "$t")"
  "$t" -I DOCKER-USER 2 -i gemina0 -j "$cur"
  [[ "$(exit_egress_current "$t")" == "$cur" ]] || fail "$t: current with duplicate jump"
  exit_egress_apply "$t" gemina0 eth0
  [[ $(grep -c GEMINA-EGRESS "$du") -eq 1 ]] || fail "$t: duplicate jump not repaired"
  # remove undoes everything
  exit_egress_remove "$t"
  [[ $(grep -c GEMINA-EGRESS "$du" || true) -eq 0 ]] || fail "$t: remove left a jump"
  exit_egress_apply "$t" gemina0
done
[[ ! -s "$tmp/violations" ]] || { cat "$tmp/violations" >&2; fail "jump pointed at an incomplete chain (fail-open window) or a referenced chain was deleted"; }

live4="$tmp/state/iptables/$(exit_egress_current iptables)"; live6="$tmp/state/ip6tables/$(exit_egress_current ip6tables)"
n4=$(grep -c -- "-j DROP" "$live4"); n6=$(grep -c -- "-j DROP" "$live6")
[[ $n4 -eq 11 && $n6 -eq 6 ]] || fail "rule counts v4=$n4 v6=$n6"
for c in 169.254.0.0/16 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16 100.64.0.0/10 224.0.0.0/4 \
         240.0.0.0/4 192.0.0.0/24 198.18.0.0/15; do
  grep -q -- "-d $c -j DROP" "$live4" || fail "missing v4 $c"; done
for c in fc00::/7 fe80::/10 ff00::/8 ::1/128 ::/128 ::ffff:0:0/96; do
  grep -q -- "-d $c -j DROP" "$live6" || fail "missing v6 $c"; done
echo "PASS exit-egress"
