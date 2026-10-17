#!/usr/bin/env bash
# Test: egress DROP rules precede the ACCEPT, cover metadata/private ranges, and
# are idempotent. Uses a fake iptables that models DOCKER-USER in a file.
# Usage: scripts/test-exit-egress.sh   (no root needed)
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
for t in iptables ip6tables; do
cat >"$tmp/$t" <<STUB
#!/usr/bin/env bash
f="$tmp/$t.chain"; touch "\$f"; shift_args=("\$@")
op="\$1"; shift; shift  # op, chain
case "\$op" in
  -C) grep -qxF -- "\$*" "\$f" ;;
  -A) echo "\$*" >>"\$f" ;;
  -I) pos="\$1"; shift; { awk -v p=\$pos -v r="\$*" 'NR==p{print r} {print} END{if(NR<p)print r}' "\$f"; } >"\$f.n"; mv "\$f.n" "\$f" ;;
esac
STUB
chmod +x "$tmp/$t"; done
export PATH="$tmp:$PATH"
GEMINA_SETUP_LIB_ONLY=1 source "$here/setup-exit-host.sh"
fail() { echo "FAIL: $*" >&2; exit 1; }
for t in iptables ip6tables; do
  "$t" -I DOCKER-USER 1 -i gemina0 -o eth0 -j ACCEPT
  exit_egress_apply "$t" gemina0
  first_accept=$(grep -n ACCEPT "$tmp/$t.chain" | head -1 | cut -d: -f1)
  last_drop=$(grep -n DROP "$tmp/$t.chain" | tail -1 | cut -d: -f1)
  [[ $last_drop -lt $first_accept ]] || fail "$t: DROP not before ACCEPT"
  before=$(cat "$tmp/$t.chain"); exit_egress_apply "$t" gemina0
  [[ "$before" == "$(cat "$tmp/$t.chain")" ]] || fail "$t: not idempotent"
done
for c in 169.254.0.0/16 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16 100.64.0.0/10; do
  grep -q -- "-d $c -j DROP" "$tmp/iptables.chain" || fail "missing v4 $c"; done
for c in fc00::/7 fe80::/10; do
  grep -q -- "-d $c -j DROP" "$tmp/ip6tables.chain" || fail "missing v6 $c"; done
echo "PASS exit-egress"
