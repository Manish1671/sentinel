# Grafana

Provisioned from this directory.

- Datasource: Prometheus at `http://prometheus:9090`
- Dashboards loaded from `dashboards/` into folder **Sentinel**

Local UI: [http://localhost:3002](http://localhost:3002) (mapped from container `:3000`).

## Dashboards

| UID | Title | Content |
| --- | --- | --- |
| `sentinel-overview` | Sentinel Overview | HTTP rate/latency/errors, Kafka throughput/lag, active incidents |
| `sentinel-detection` | Detection & Incidents | Events processed, alerts, correlation, rule triggers |
| `sentinel-ai-remediation` | AI & Remediation | Investigations, tools, remediation and verification |

All panel queries use live Prometheus series. Empty graphs mean no traffic yet, not synthetic fill.

Anonymous Viewer is enabled for local development. Admin password defaults to `admin`.
