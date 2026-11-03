# 可扩展存储：用配置新增后端

UModel 按**查询家族**而非写死的存储 kind 把存储路由到查询计划渲染器。家族就是查询模型——
`label-timeseries`（PromQL）、`document-search`（Elasticsearch DSL）、`sql-table`（SQL）、`label-logs`（LogQL）。存储通过
`spec.family` 选择家族。本示例展示新增后端的两种方式。

## 同家族——只写配置

[VictoriaMetrics](https://victoriametrics.com/) 使用 Prometheus 查询 API，与 Prometheus 共享
`label-timeseries` 家族，**完全不需要 Go 代码**：

```yaml
kind: victoriametrics
spec:
  family: label-timeseries
  endpoint: "http://localhost:8428"
```

对以该存储为后端的实体执行 `get_metrics`，生成的计划与原生 Prometheus 完全一致（`prometheus_promql`）：
复用现有的 `label-timeseries` 渲染器。见 [`victoriametrics.storage.yaml`](victoriametrics.storage.yaml)。

## 新家族——写一个渲染器，之后只写配置

[ClickHouse](https://clickhouse.com/) 用 SQL，与 PromQL 截然不同的查询模型。`sql-table` 家族为该模型
新增一个渲染器；它复用与其他家族相同的已解析约束（实体绑定 + 过滤），把它们格式化为 `WHERE` 子句。
ClickHouse 后端随后通过配置路由到它：

```yaml
kind: clickhouse
spec:
  family: sql-table
  endpoint: "http://localhost:8123"
  table: service_metrics
```

`get_metrics` 渲染为 SQL——`SELECT toStartOfInterval(ts, INTERVAL 1 MINUTE), avg(value) … WHERE … GROUP BY …`——
其中 `{from}` / `{to}` 占位符由调用方用计划的时间范围填充。见 [`clickhouse.storage.yaml`](clickhouse.storage.yaml)。

`sql-table` 家族此后的每个后端都只是配置，与 `label-timeseries` 里的 VictoriaMetrics 一样。

## 日志家族——LogQL 流选择器

[Grafana Loki](https://grafana.com/oss/loki/) 用 LogQL 查询。`label-logs` 家族为它新增一个渲染器，同样基于
共享的已解析约束：每条约束变成流选择器里的一个标签匹配。`loki` 这个 kind 默认路由到该家族，无需声明
`spec.family`：

```yaml
kind: loki
spec:
  endpoint: "http://localhost:3100"
```

`get_logs` 渲染为 `{k8s_cluster="local", namespace="demo", pod="api-1"}`，同时带上端点、API 前缀、
方向、条数和租户头。见 [`loki.storage.yaml`](loki.storage.yaml)。

这个渲染器刻意从严：过滤无法表达为标签匹配时拒绝渲染（否则会悄悄选出比要求更多的日志流）；去掉后没有
正向匹配时拒绝（Loki 会拒绝这类选择器）；映射出的字段名不是合法的 Loki 标签名时也拒绝。它只选日志流；
需要行过滤（`|=`、`|~`）的话，在返回的查询后面自行追加。

## kind 是怎么新增的

两个只写配置的 kind（VictoriaMetrics、ClickHouse）都只动了 schema：

- [`schemas/core/storage/victoriametrics.schema.yaml`](../../schemas/core/storage/victoriametrics.schema.yaml) 与 [`clickhouse.schema.yaml`](../../schemas/core/storage/clickhouse.schema.yaml) 定义 kind。
- Go、Python、Java 三套 SDK 类型与参考文档由 `make expand`、`make docs-schema` 从这些 schema 生成。
- [`loki.schema.yaml`](../../schemas/core/storage/loki.schema.yaml) 同样定义 kind，另外为新的 `label-logs` 家族加了一个渲染器（`internal/query/loki_renderer.go`）。
