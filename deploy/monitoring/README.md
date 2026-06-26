# Gateway monitoring stack (Prometheus + Grafana)

A self-contained Prometheus + Grafana deployment that scrapes the Stage-1 probe
gateway's `/metrics` endpoint and renders the **Gemina — Failover
Effectiveness** dashboard.

## What it shows

The dashboard answers the only question that matters for a continuity VPN: *is
each uplink delivering, and is the redundancy actually working?*

- **First-copy rate by path** — which uplink (`wi-fi`, `android-usb-tether`) is
  delivering the packet that arrives first.
- **Duplicate rate by path** — the redundant copies, i.e. proof both paths carry
  traffic.
- **Rejected rate by reason** — malformed / version-skewed / invalid datagrams.
- **Delivery share by path** — the failover split over the last hour.
- **Gateway scrape state** — `up`/`down` reachability of the gateway.

All series use only coarse tokens (see `../../observability/METRICS.md`); no IP,
MAC, or session identifier ever reaches Prometheus.

## Security model

- Both Prometheus and Grafana bind to **`127.0.0.1` only**. Nothing is published
  to a public port and **no cloud ingress (VCN) rule is required**.
- The gateway's metrics port is **never published to the host** — Prometheus
  reaches it by container name over the private `gemina-mon` network.
- Reach Grafana through an SSH tunnel:

  ```bash
  ssh -L 3000:localhost:3000 oracle
  # open http://localhost:3000   (user: admin)
  ```

  The admin password is generated on first deploy into `.env` (chmod 600, never
  committed). Retrieve it with:

  ```bash
  ssh oracle 'cat /opt/gemina/deploy/monitoring/.env'
  ```

## Deploy / update

From the repo root:

```bash
scripts/deploy-monitoring.sh           # host defaults to "oracle"
```

The script rsyncs the source, ensures the shared docker network, refreshes the
gateway unit so it exposes `/metrics`, generates the Grafana password on first
run, and brings the stack up. Re-run it to ship config or dashboard changes.

## Layout

| Path | Purpose |
|------|---------|
| `docker-compose.yml` | Prometheus + Grafana services (localhost-bound, shared network) |
| `prometheus/prometheus.yml` | Scrape config (`gemina-gateway:9090`) + alert rule loading |
| `grafana/provisioning/datasources/` | Prometheus datasource (fixed uid) |
| `grafana/provisioning/dashboards/` | Dashboard provider |
| `grafana/dashboards/failover-dashboard.json` | Provisioned dashboard (datasource pinned) |
| `.env` | Grafana admin password (generated, git-ignored) |

Alert rules are mounted read-only from the repo's single source of truth,
`../../observability/alerts/gateway-rules.yml`. Alertmanager is **not** deployed
yet — rules evaluate and are visible in Prometheus' *Alerts* tab but are not
routed to a receiver.
