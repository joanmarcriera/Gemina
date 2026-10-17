#!/usr/bin/env bash
# Test: GEMINA-EGRESS jump is rule 1 of DOCKER-USER (above the ACCEPTs), the chain
# is rebuilt each run, jumps never duplicate, stale per-range DROPs are removed,
# even when WAN_IF changes between runs. Fake iptables models chains as files.
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
  -X) rm -f "\$f" ;;
  -C) grep -qxF -- "\$*" "\$f" ;;
  -A) echo "\$*" >>"\$f" ;;
  -D) grep -qxF -- "\$*" "\$f" || exit 1
      awk -v r="\$*" '!d && \$0==r {d=1; next} {print}' "\$f" >"\$f.n"; mv "\$f.n" "\$f" ;;
  -S) [[ -e \$f ]] && sed "s/^/-A \$chain /" "\$f" ;;
  -I) pos="\$1"; shift; touch "\$f"
      awk -v p=\$pos -v r="\$*" 'NR==p{print r} {print} END{if(NR<p)print r}' "\$f" >"\$f.n"; mv "\$f.n" "\$f" ;;
esac
STUB
chmod +x "$tmp/$t"; done
export PATH="$tmp:$PATH"
# shellcheck source=/dev/null
GEMINA_SETUP_LIB_ONLY=1 source "$here/setup-exit-host.sh"
fail() { echo "FAIL: $*" >&2; exit 1; }

for t in iptables ip6tables; do
  du="$tmp/state/$t/DOCKER-USER"; ch="$tmp/state/$t/GEMINA-EGRESS"
  # Simulate the old fail-open design: stale per-range DROP sitting in DOCKER-USER.
  "$t" -I DOCKER-USER 1 -i gemina0 -d "$([[ $t == ip6tables ]] && echo fe80::/10 || echo 10.0.0.0/8)" -j DROP
  for wan in eth0 ens3; do   # second run changes WAN_IF
    "$t" -I DOCKER-USER 1 -i gemina0 -o "$wan" -j ACCEPT
    exit_egress_apply "$t" gemina0
    [[ "$(head -1 "$du")" == "-i gemina0 -j GEMINA-EGRESS" ]] || fail "$t/$wan: jump not rule 1"
    [[ $(grep -c GEMINA-EGRESS "$du") -eq 1 ]] || fail "$t/$wan: duplicate jumps"
    grep -q -- "-j DROP" "$du" && fail "$t/$wan: stale DROP left in DOCKER-USER"
    [[ "$(tail -1 "$ch")" == "-j RETURN" ]] || fail "$t: chain must end in RETURN"
  done
  # (b) chain rebuilt: junk added to the chain must vanish on the next run.
  echo "-d 8.8.8.8/32 -j ACCEPT" >>"$ch"; exit_egress_apply "$t" gemina0
  grep -q 8.8.8.8 "$ch" && fail "$t: chain not rebuilt"
done
n4=$(grep -c -- "-j DROP" "$tmp/state/iptables/GEMINA-EGRESS"); n6=$(grep -c -- "-j DROP" "$tmp/state/ip6tables/GEMINA-EGRESS")
[[ $n4 -eq 7 && $n6 -eq 3 ]] || fail "rule counts v4=$n4 v6=$n6"
for c in 169.254.0.0/16 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16 100.64.0.0/10; do
  grep -q -- "-d $c -j DROP" "$tmp/state/iptables/GEMINA-EGRESS" || fail "missing v4 $c"; done
for c in fc00::/7 fe80::/10; do
  grep -q -- "-d $c -j DROP" "$tmp/state/ip6tables/GEMINA-EGRESS" || fail "missing v6 $c"; done
echo "PASS exit-egress"
