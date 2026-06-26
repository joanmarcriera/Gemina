#!/usr/bin/env bash
# Repeatable deployment of the Stage-1 probe gateway to a remote arm64 host.
#
# It rsyncs the first-party source to the host, builds the container image
# natively there, installs/refreshes a systemd unit that runs the container, and
# opens the UDP port in the host firewall. Re-run it to ship a new release: the
# image is rebuilt and the service restarted in place.
#
#   scripts/deploy-dev-gateway.sh                 # deploy to host "oracle"
#   GATEWAY_HOST=oracle GATEWAY_PORT=51820 scripts/deploy-dev-gateway.sh
#
# Requirements on the remote: docker, systemd, firewalld, passwordless sudo.
# Provenance: .research-src (GPL) is never shipped — see the rsync excludes.
set -euo pipefail

HOST="${GATEWAY_HOST:-oracle}"
PORT="${GATEWAY_PORT:-51820}"
REMOTE_DIR="${GATEWAY_REMOTE_DIR:-/opt/gemina}"
IMAGE="gemina-gateway:latest"
UNIT="gemina-gateway.service"

UNIT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$UNIT_ROOT"

say() { printf '\n== %s\n' "$1"; }

say "preflight: reach $HOST"
ssh -o BatchMode=yes -o ConnectTimeout=15 "$HOST" 'true'

say "sync source -> $HOST:$REMOTE_DIR (excluding .git, .research-src, build dirs)"
ssh "$HOST" "sudo mkdir -p '$REMOTE_DIR' && sudo chown \"\$(id -u):\$(id -g)\" '$REMOTE_DIR'"
rsync -az --delete \
  --exclude '.git/' \
  --exclude '.research-src/' \
  --exclude '.build/' \
  --exclude '.netcheck/' \
  --exclude 'apps/macos/.build/' \
  --exclude 'research/usb-rndis-spike/rndis_probe' \
  ./ "$HOST:$REMOTE_DIR/"

say "build image $IMAGE on $HOST (native arm64)"
ssh "$HOST" "cd '$REMOTE_DIR' && sudo docker build -t '$IMAGE' -f deploy/docker/gateway.Dockerfile ."

say "install/refresh systemd unit $UNIT (port $PORT)"
ssh "$HOST" "
  set -e
  sudo install -m 0644 '$REMOTE_DIR/deploy/systemd/$UNIT' '/etc/systemd/system/$UNIT'
  sudo sed -i 's/^Environment=GATEWAY_PORT=.*/Environment=GATEWAY_PORT=$PORT/' '/etc/systemd/system/$UNIT'
  sudo systemctl daemon-reload
  sudo systemctl enable '$UNIT'
  sudo systemctl restart '$UNIT'
"

say "open UDP $PORT in host firewall"
ssh "$HOST" "
  sudo firewall-cmd --add-port=$PORT/udp --permanent
  sudo firewall-cmd --reload
" || echo "warning: firewalld change failed; check manually"

say "status"
ssh "$HOST" "sudo systemctl --no-pager --full status '$UNIT' | head -n 12; echo '--- recent logs ---'; sudo journalctl -u '$UNIT' -n 8 --no-pager"

cat <<EOF

Deployed. Host firewall opened for UDP $PORT.

NOTE: if $HOST is behind a cloud firewall (e.g. Oracle Cloud VCN security
list / network security group), you must also allow ingress UDP $PORT there —
that is configured in the cloud console, not on the host. Verify reachability
with scripts/probe-gateway.sh once both layers are open.
EOF
