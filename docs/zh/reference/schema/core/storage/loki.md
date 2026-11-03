# loki

Loki 存储，用于描述 Loki 或 LogQL 兼容日志后端的配置和查询规划信息。get_logs 默认通过 label-logs 查询家族渲染出 LogQL 查询计划，UModel 不直接执行查询，也不保存明文凭据。

**Kind**: `loki`

> 每个元素共享标准信封 `kind` · [metadata](../../shared-types#metadata) · [schema](../../shared-types#schema).

## `spec` 字段

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| `family` | `string` |  | `label-logs` | 该存储渲染所用的查询家族。默认 label-logs，即 LogQL 日志查询；兼容 LogQL 的其他后端也可以复用它。 |
| `endpoint` | `string` | 是 |  | Loki 或 LogQL 兼容服务的访问地址，例如 http://loki:3100。 |
| `api_prefix` | `string` |  | `/loki/api/v1` | Loki HTTP API 前缀。默认值为 /loki/api/v1。 |
| `default_direction` | enum: `backward`, `forward` |  | `backward` | 默认返回顺序。backward 表示从新到旧，forward 表示从旧到新。 |
| `default_limit` | `integer` |  |  | 未指定条数时的默认最大返回条数。 |
| `tenant` | `string` |  |  | 可选租户标识，对应多租户 Loki 的 X-Scope-OrgID。 |
| `tenant_header` | `string` |  |  | 多租户系统使用的租户 HTTP 头名称，例如 X-Scope-OrgID。 |
| `credential_ref` | `string` |  |  | 凭据引用标识，例如 secret://loki-prod-readonly。不得在 UModel 中保存明文用户名、密码或 Token。 |
| `tls_verify` | `boolean` |  | `true` | 是否校验 TLS 证书。默认值为 true。 |
| `external_labels` | map&lt;string, string&gt; |  |  | 外部标签，用于 query planning 时补充查询上下文或结果来源说明。 |
| `headers` | map&lt;string, string&gt; |  |  | 非敏感 HTTP 头，用于租户、路由等查询上下文。不得存放认证密钥。 |
| `properties` | map&lt;string, string&gt; |  |  | Loki 的额外非敏感配置，以键值对形式存储。 |
| `tags` | map&lt;string, string&gt; |  |  | 用于标注该 Loki 存储的标签。 |
