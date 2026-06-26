#!/usr/bin/env bash
# Deploy the Prometheus + Grafana monitoring stack to the gateway host.
#
# It rsyncs the first-party source, ensures the shared `gemina-mon` docker
# network exists, refreshes the gateway unit so it exposes /metrics on that
# network, generates a Grafana admin password on first run (never printed,
# never committed), and brings the stack up with docker compose.
#
# Both Prometheus and Grafana bind to localhost only — reach Grafana through an
# SSH tunnel (see deploy/monitoring/README.md). No cloud ingress rule is needed.
#
#   scripts/deploy-monitoring.sh                 # deploy to host "oracle"
#   GATEWAY_HOST=oracle scripts/deploy-monitoring.sh
#
# Requirements on the remote: docker + compose plugin, systemd, passwordless sudo.
set -euo pipefail

HOST="${GATEWAY_HOST:-oracle}"
REMOTE_DIR="${GATEWAY_REMOTE_DIR:-/opt/gemina}"
UNIT="gemina-gateway.service"
MON_DIR="$REMOTE_DIR/deploy/monitoring"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

say() { printf '\n== %s\n' "$1"; }

say "preflight: reach $HOST and check compose"
ssh -o BatchMode=yes -o ConnectTimeout=15 "$HOST" 'sudo docker compose version >/dev/null'

say "sync source -> $HOST:$REMOTE_DIR (excluding .git, .research-src, build dirs, .env)"
ssh "$HOST" "sudo mkdir -p '$REMOTE_DIR' && sudo chown \"\$(id -u):\$(id -g)\" '$REMOTE_DIR'"
rsync -az --delete \
  --exclude '.git/' \
  --exclude '.research-src/' \
  --exclude '.build/' \
  --exclude '.netcheck/' \
  --exclude 'apps/macos/.build/' \
  --exclude 'deploy/monitoring/.env' \
  ./ "$HOST:$REMOTE_DIR/"

say "ensure shared network + refresh gateway unit (exposes /metrics on gemina-mon)"
ssh "$HOST" "
  set -e
  sudo docker network create gemina-mon 2>/dev/null || true
  sudo install -m 0644 '$REMOTE_DIR/deploy/systemd/$UNIT' '/etc/systemd/system/$UNIT'
  sudo systemctl daemon-reload
  sudo systemctl restart '$UNIT'
"

say "generate Grafana admin password on first run (kept in $MON_DIR/.env, chmod 600)"
ssh "$HOST" "
  set -e
  cd '$MON_DIR'
  if [ ! -f .env ]; then
    pw=\$(head -c 24 /dev/urandom | base64 | tr -d '/+=' | cut -c1-24)
    umask 177
    printf 'GRAFANA_ADMIN_PASSWORD=%s\n' \"\$pw\" > .env
    echo 'generated new .env (password not printed)'
  else
    echo '.env already present; keeping existing password'
  fi
"

say "bring up the monitoring stack"
ssh "$HOST" "cd '$MON_DIR' && sudo docker compose pull -q && sudo docker compose up -d"

say "status"
ssh "$HOST" "
  cd '$MON_DIR'
  sudo docker compose ps
  echo '--- prometheus target health (expect up=1 once a scrape completes) ---'
  sleep 6
  curl -fsS 'http://127.0.0.1:9091/api/v1/query?query=up{job=\"gemina-gateway\"}' 2>/dev/null \
    | sed -E 's/[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/<ip>/g' || echo '(query failed; check compose logs)'
"

cat <<EOF

Monitoring deployed (localhost-bound on $HOST). To view the dashboard:

  ssh -L 3000:localhost:3000 $HOST
  # then open http://localhost:3000  (user: admin)
  # password:  ssh $HOST 'cat $MON_DIR/.env'

Prometheus (optional, for debugging):
  ssh -L 9091:localhost:9091 $HOST   # then http://localhost:9091
EOF
