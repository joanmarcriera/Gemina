---
name: gemina-gateway-ops
description: Provision, deploy, and operate the gemina dedup/exit gateway host (currently an Oracle Cloud arm64 VM, SSH alias `oracle`, region uk-london-1 — NOT the Hetzner joanmarcriera.es box). Covers scripts/deploy-dev-gateway.sh, scripts/setup-exit-host.sh, scripts/probe-gateway.sh, scripts/deploy-monitoring.sh, the two systemd units (gemina-gateway.service Stage-1 probe / gemina-gateway-data.service Stage-2 data+exit, DRAFT), host nftables/iptables NAT + firewalld + Oracle VCN security-list ingress, and the Prometheus/Grafana monitoring stack. Use for "deploy the gateway", "set up the exit host", "probe the gateway", "gateway won't dedupe/exit", "stand up monitoring for the gateway", or any change to deploy/{docker,systemd,nftables,cloud-init,ansible,tofu,monitoring}. Distinct from run-geminactl (client/CLI build+test, explicitly scopes OUT gateway runtime) and from riera-selfhost-ops (the Hetzner Traefik /opt/stacks product stack — a different host entirely).
---

# gemina gateway ops

The gateway is the server half of gemina's dual-path continuity VPN: it
receives duplicate copies of each packet over the client's two uplinks and
delivers the first valid copy. **Ground truth: the only host wired up today is
an Oracle Cloud arm64 VM, SSH alias `oracle`, region `uk-london-1`, remote dir
`/opt/gemina`** (`docs/dev/gateway-deploy.md`). No Hetzner gateway host exists
in this repo — `.research-src/terraform-provider-hcloud` is inspiration-only
research (see "stubs" below). Don't invent Hetzner gateway infra; if Marc
moves this host, update this skill first.

## Two gateway modes

- **Stage-1 probe** (**live, deployed today**) — `gemina-gateway.service`,
  container `--read-only`, no host networking; only dedups probe packets and
  logs decisions. No prereqs. Env: `GEMINA_GATEWAY_ADDR`, `_DEDUP_CAPACITY`,
  `_LOG_LEVEL`.
- **Stage-2 data+exit** (**DRAFT, unvalidated on hardware since 2026-06-28**) —
  `gemina-gateway-data.service`, not read-only, `--network host`, `--cap-add
  NET_ADMIN`, `--device /dev/net/tun`; real handshake + encrypted data plane +
  internet exit via TUN. Requires `scripts/setup-exit-host.sh` first. Env adds
  `GEMINA_GATEWAY_MODE=data`, `_EXIT=on`, `_IDENTITY`, `_POOL`, `_TUN`.

Never run both on the same port. Both the Stage-2 unit and
`setup-exit-host.sh` carry a DRAFT header — validate on hardware during WS-F
(`docs/superpowers/plans/2026-06-26-phase3-wifi-tunnel.md`) before treating
either as production-ready.

## Deploy / redeploy the probe gateway

```bash
scripts/deploy-dev-gateway.sh                 # host "oracle", UDP 51820
GATEWAY_HOST=oracle GATEWAY_PORT=51820 scripts/deploy-dev-gateway.sh
```

Idempotent: rsyncs first-party source to `oracle:/opt/gemina` (excludes
`.git`, `.research-src`, build dirs — GPL research source must never ship),
builds `gemina-gateway:latest` natively (arm64, no cross-compile, no registry)
from `deploy/docker/gateway.Dockerfile`, installs/refreshes the systemd unit,
restarts it, opens the port in the **host** firewall (`firewall-cmd`).

**Two firewalls exist.** The script only opens the host one; the **Oracle
Cloud VCN security list** is separate, changeable only from the OCI console
(region uk-london-1) or OCI CLI. Classic mistake: leaving the ingress rule's
**source port range** as the gateway port instead of **All** — the client's
source port is ephemeral, so a narrow rule silently drops every probe.
Console steps: `docs/dev/gateway-deploy.md`. Remove:
`systemctl disable --now gemina-gateway.service` + delete the unit + `docker
rmi gemina-gateway:latest` + remove `/opt/gemina` + `firewall-cmd
--remove-port=<port>/udp --permanent --reload`.

## Bring up Stage-2 data+exit (DRAFT — validate before trusting)

```bash
scripts/deploy-dev-gateway.sh                                          # 1. ship the image
ssh oracle 'cd /opt/gemina && sudo WAN_IF=<egress-iface> scripts/setup-exit-host.sh'  # 2. host NAT
ssh oracle 'sudo install -m0644 /opt/gemina/deploy/systemd/gemina-gateway-data.service \
  /etc/systemd/system/ && sudo systemctl daemon-reload && \
  sudo systemctl enable --now gemina-gateway-data.service'             # 3. start it
ssh oracle 'sudo journalctl -u gemina-gateway-data.service | grep public_key | tail -1'  # 4. get the pinned key
```

`setup-exit-host.sh` (idempotent, root) enables `net.ipv4.ip_forward`
(`/etc/sysctl.d/99-gemina-exit.conf`); creates a **persistent TUN `gemina0`**
owned by uid `65532` (distroless `:nonroot` uid) so it exists before the
container starts; addresses it (`10.99.0.1/16` in pool `10.99.0.0/16` — must
match `GEMINA_GATEWAY_POOL`); adds a MASQUERADE rule out `WAN_IF` (nftables
preferred, iptables fallback; defaults to the default-route interface).
`internal/exit/nat_linux.go` never touches the firewall — this script is the
only place the NAT rule is written (`deploy/nftables/` is still an empty
placeholder). Ed25519 identity persists at `$STATE_DIR/gateway-identity.key`
(default `/var/lib/gemina`) — losing it breaks every client's pin. Remove:
disable the unit, `ip link del gemina0`, drop the MASQUERADE rule, `rm
/etc/sysctl.d/99-gemina-exit.conf`.

## Verify end-to-end

```bash
scripts/probe-gateway.sh                       # host "oracle", port 51820, iface en0
GATEWAY_HOST=oracle GATEWAY_PORT=51820 GATEWAY_IFACE=en0 scripts/probe-gateway.sh
```

Builds `geminactl` locally, marks a UTC timestamp from the remote clock, sends
two probes incl. a deliberate duplicate (`geminactl probe -duplicate`), then
greps `journalctl` **since that marker only**. Expect `"decision":"first-copy"`
and `"decision":"duplicate"`; the gateway never logs the source address
(redaction invariant — see `run-geminactl`). Silent no-result almost always
means the VCN source port range isn't "All". On-box: `ssh oracle`, probe
`localhost:51820`, read `journalctl -u gemina-gateway.service -n 20`.

## Monitoring (Prometheus + Grafana)

```bash
scripts/deploy-monitoring.sh                   # host defaults to "oracle"
```

Rsyncs `deploy/monitoring/`, creates the shared `gemina-mon` docker network,
refreshes the gateway unit so `/metrics` is reachable *on that private network
only* (never published to the host), generates a random Grafana admin
password into `deploy/monitoring/.env` (chmod 600, git-ignored, first run
only), runs `docker compose up -d`. **Both bind to `127.0.0.1` only** — no VCN
rule needed; `ssh -L 3000:localhost:3000 oracle` → `http://localhost:3000`.
Dashboard "Failover Effectiveness": first-copy rate, duplicate rate, rejected
rate by reason, delivery share by path, scrape up/down. Alert rules
(`observability/alerts/gateway-rules.yml`) evaluate in Prometheus' *Alerts*
tab but **Alertmanager is not deployed**.

## Stubs, and related skills

`deploy/cloud-init/`, `deploy/ansible/`, `deploy/tofu/modules/` are placeholder
READMEs ("will be added" / "not implemented yet"); `tofu` `dev`/`production`
envs are a bare `terraform{}` + `locals{}`, no provider, no resources — host
setup today is entirely the hand-run scripts above over SSH, not IaC.
`run-geminactl` = client/CLI build+test, excludes gateway runtime.
`riera-selfhost-ops` = Marc's Hetzner Traefik `/opt/stacks` stack, unrelated host.
