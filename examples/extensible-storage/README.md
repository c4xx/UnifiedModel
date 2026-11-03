# Extensible storage: adding backends with configuration

UModel routes a storage to a query-plan renderer by **query family**, not by a
hardcoded storage kind. A family is a query model — `label-timeseries` (PromQL),
`document-search` (Elasticsearch DSL), `sql-table` (SQL), `label-logs` (LogQL). A storage selects its
family with `spec.family`. This example shows the two ways a new backend is added.

## Same family — configuration only

[VictoriaMetrics](https://victoriametrics.com/) speaks the Prometheus query API,
so it shares the `label-timeseries` family. It needs **no Go code at all**:

```yaml
kind: victoriametrics
spec:
  family: label-timeseries
  endpoint: "http://localhost:8428"
```

`get_metrics` on an entity backed by this storage produces the same
`prometheus_promql` plan it would for native Prometheus — the existing
`label-timeseries` renderer handles it. See [`victoriametrics.storage.yaml`](victoriametrics.storage.yaml).

## New family — one renderer, then configuration

[ClickHouse](https://clickhouse.com/) uses SQL, a query model entirely different
from PromQL. The `sql-table` family adds one renderer for that model; it consumes
the same resolved constraints (entity bindings + filters) the other families do
and formats them as a `WHERE` clause. A ClickHouse backend then routes to it by
configuration:

```yaml
kind: clickhouse
spec:
  family: sql-table
  endpoint: "http://localhost:8123"
  table: service_metrics
```

`get_metrics` renders to SQL — `SELECT toStartOfInterval(ts, INTERVAL 1 MINUTE), avg(value) … WHERE … GROUP BY …` — with `{from}` / `{to}` placeholders the caller fills from the plan's time range. See [`clickhouse.storage.yaml`](clickhouse.storage.yaml).

Every backend in the `sql-table` family after the first is configuration only,
exactly like VictoriaMetrics in `label-timeseries`.

## Log family — LogQL stream selectors

[Grafana Loki](https://grafana.com/oss/loki/) is queried with LogQL. The
`label-logs` family adds one renderer for it, again built on the shared
constraints: every constraint becomes a label matcher in a stream selector.
The `loki` kind routes to this family by default, so no `spec.family` is needed:

```yaml
kind: loki
spec:
  endpoint: "http://localhost:3100"
```

`get_logs` renders to `{k8s_cluster="local", namespace="demo", pod="api-1"}`
with the endpoint, API prefix, direction, limit and tenant header alongside. See
[`loki.storage.yaml`](loki.storage.yaml).

The renderer is deliberately strict. It refuses to render when a filter cannot
be expressed as a label matcher (it would silently select more streams than
asked), when no positive matcher is left (Loki rejects those selectors), or when
a mapped field name is not a valid Loki label. It selects streams only; add line
filters (`|=`, `|~`) to the returned query if you need them.

## How a kind is added

The two config-only kinds (VictoriaMetrics, ClickHouse) were schema-only:

- [`schemas/core/storage/victoriametrics.schema.yaml`](../../schemas/core/storage/victoriametrics.schema.yaml) and [`clickhouse.schema.yaml`](../../schemas/core/storage/clickhouse.schema.yaml) define the kinds. [`loki.schema.yaml`](../../schemas/core/storage/loki.schema.yaml) does too, plus one renderer (`internal/query/loki_renderer.go`) for the new `label-logs` family.
- The Go, Python, and Java SDK types and the reference docs are generated from those schemas by `make expand` and `make docs-schema`.
