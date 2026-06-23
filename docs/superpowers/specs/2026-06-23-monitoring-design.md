# Monitoring & Observability — Design

Date: 2026-06-23
Status: Approved (owner), ready to implement

## Goal

Make the system observable, gateway-first, with **failover effectiveness** as the
headline signal — proving the product actually keeps connections alive — without
ever leaking a host identifier. Open and self-hostable (no vendor lock-in), the
same instrumentation serving both our hosted ops and self-hosters.

## Decisions

- **Scope:** both client and gateway, but build the gateway now; define the
  client signal vocabulary for when the macOS app lands.
- **Stack:** a Prometheus `/metrics` endpoint on the gateway + the existing
  redacted JSON logs; Grafana dashboards and Prometheus alert rules in
  `observability/`.
- **Implementation:** a **stdlib-only** metrics registry (no new dependencies),
  rendering the Prometheus text exposition format. Counters + gauges cover the
  failover signal; histograms are deferred.
- **Hero signal:** failover effectiveness via per-path delivery + dedup.

## Architecture

```
UDP 51820 ── gateway Server (Handle) ──┐
                                        ├─ increments metrics.Registry
HTTP <metrics-addr> ── /metrics ────────┘   (counters/gauges, coarse labels)
                          │
                          └─ Prometheus scrape ─ Grafana / Alertmanager
```

- `internal/metrics` — a small thread-safe registry: `Counter`/`Gauge` vectors
  with labels, and `Render()` producing the Prometheus text format. ~150 LOC.
- The gateway holds a `*metrics.Registry`; its hot path (already counting
  first-copy / duplicate / rejected) increments series. The data plane updates an
  active-sessions gauge.
- `cmd/gateway` starts a second HTTP listener on
  `CONTINUITY_GATEWAY_METRICS_ADDR` (off when unset) serving `GET /metrics`.

## Metric vocabulary (redaction-safe)

Every label value is a fixed coarse token — never a session id, IP, MAC, port or
serial.

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `continuity_packets_total` | counter | `decision`, `path` | the hero: first-copy/duplicate/rejected per path |
| `continuity_rejected_total` | counter | `reason` | malformed / unknown-session / auth-failure |
| `continuity_active_sessions` | gauge | — | live data-plane sessions |
| `continuity_build_info` | gauge | `version` | always 1; build label |

- `decision ∈ {first-copy, duplicate, rejected}`
- `path ∈ {wi-fi, android-usb-tether, unknown}`
- `reason ∈ {bad-magic, short, unsupported-version, invalid-identity, unknown-session, auth-failure}`

**Failover effectiveness** is read off these in Grafana: first-copy-by-path shows
which link is delivering; duplicate-by-path shows the redundancy working; a drop
of one path's series to zero while the session count holds is a survived
path-loss.

## Client signal vocabulary (defined now, implemented with the app)

Same tokens. To live in the macOS app later:

- `path_state{path}` — up/down per uplink.
- `failovers_survived_total` — increments when one path goes quiet but the
  session continues over the other (path-loss survival, best seen client-side).
- `active_paths` — count currently carrying traffic.

## Observability assets

- `observability/prometheus/` — a scrape config snippet for the gateway target.
- `observability/grafana/` — one dashboard JSON: packets-by-decision/path,
  per-path delivery share, rejected rate, active sessions.
- `observability/alerts/` — rules: gateway target down; **zero first-copies for N
  minutes while sessions > 0 = outage**; rejected-rate spike.
- `observability/METRICS.md` — documents every metric name, label set and the
  allowed token enums; the single source of truth shared with the client.

## Redaction (enforced)

- The gateway passes only the fixed enum tokens as labels; it never constructs a
  label from network data.
- A test renders the registry after driving the gateway with traffic and asserts
  the output matches an IP/MAC/session regex nowhere; the existing smoke
  redaction grep is extended to the `/metrics` body.

## Testing (TDD)

- `internal/metrics`: counter increment + render format; labelled series; gauge
  set; valid Prometheus exposition (HELP/TYPE/sample); concurrent `Inc` under
  `-race`.
- gateway wiring: a `Handle` of each decision bumps the right series with the
  right tokens.
- redaction: rendered output carries only allowed tokens.

## Out of scope (now)

Histograms/latency, the client-side implementation (no app yet), per-entitlement
usage metering (billing), and shipping a full Alertmanager/Grafana stack — only
the assets/config live in the repo.
