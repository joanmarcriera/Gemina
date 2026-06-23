# Prometheus

Scrape config for the gateway's `/metrics` endpoint: [`scrape-gateway.yml`](scrape-gateway.yml).
Metric names and labels are documented in [`../METRICS.md`](../METRICS.md). The
endpoint is enabled by setting `CONTINUITY_GATEWAY_METRICS_ADDR`.
