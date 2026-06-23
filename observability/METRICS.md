# Metrics

The single source of truth for the system's metric names, label sets and the
allowed label-value tokens. The gateway exposes these in the Prometheus text
format at `GET /metrics` on `CONTINUITY_GATEWAY_METRICS_ADDR` (unset = disabled).

**Redaction invariant.** Every label *value* is a fixed coarse token from the
enums below — never a session id, IP, MAC, port or serial. Call sites pass only
these tokens; `internal/gateway` never builds a label from network data, and a
test asserts the rendered output carries no identifier.

## Gateway metrics (implemented)

### `continuity_packets_total` (counter)

Probe datagrams handled, by outcome and path. This is the **failover hero**:
first-copy-by-path shows which uplink is delivering; duplicate-by-path shows the
redundancy actually working.

| Label | Allowed values |
|---|---|
| `decision` | `first-copy`, `duplicate`, `rejected` |
| `path` | `wi-fi`, `android-usb-tether`, `unknown` |

### `continuity_rejected_total` (counter)

Datagrams rejected before deduplication, by reason.

| Label | Allowed values |
|---|---|
| `reason` | `short`, `bad-magic`, `unsupported-version`, `invalid-identity`, `other` |

## Client metrics (planned — defined now, implemented with the macOS app)

The app will expose the failover-survival view best seen client-side, using the
same `path` tokens:

- `continuity_path_state{path}` (gauge) — 1 up, 0 down, per uplink.
- `continuity_failovers_survived_total` (counter) — increments when one path goes
  quiet but the session continues over the other.
- `continuity_active_paths` (gauge) — uplinks currently carrying traffic.

## Reading failover effectiveness

- **Which path carries traffic:** `sum by (path) (rate(continuity_packets_total{decision="first-copy"}[5m]))`.
- **Redundancy working:** a healthy `duplicate` rate alongside `first-copy` means
  both paths are delivering the same packets.
- **Survived path loss:** one path's `first-copy` series falls to zero while the
  other continues — the session rode through a link drop.
- **Outage:** `first-copy` rate at zero across all paths (see the alert rules).
