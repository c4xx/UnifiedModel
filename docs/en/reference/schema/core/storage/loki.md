# loki

Loki storage, used to define Loki or LogQL-compatible log backend metadata for query planning. get_logs renders a LogQL query plan through the label-logs query family by default; UModel does not execute queries or sto…

**Kind**: `loki`

> Every element shares the standard envelope `kind` · [metadata](../../shared-types#metadata) · [schema](../../shared-types#schema).

## `spec` fields

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `family` | `string` |  | `label-logs` | The query family this storage renders through. Defaults to label-logs (LogQL log queries); other LogQL-compatible backends can reuse it. |
| `endpoint` | `string` | yes |  | The endpoint URL of Loki or a LogQL-compatible service, for example http://loki:3100. |
| `api_prefix` | `string` |  | `/loki/api/v1` | Loki HTTP API prefix. Defaults to /loki/api/v1. |
| `default_direction` | enum: `backward`, `forward` |  | `backward` | Default result order. backward returns newest first, forward returns oldest first. |
| `default_limit` | `integer` |  |  | Default maximum number of log lines when the request gives no limit. |
| `tenant` | `string` |  |  | Optional tenant identifier, the X-Scope-OrgID of a multi-tenant Loki. |
| `tenant_header` | `string` |  |  | Tenant HTTP header name used by multi-tenant systems, for example X-Scope-OrgID. |
| `credential_ref` | `string` |  |  | Credential reference, for example secret://loki-prod-readonly. Plaintext usernames, passwords, or tokens must not be stored in UModel. |
| `tls_verify` | `boolean` |  | `true` | Whether TLS certificates should be verified. Defaults to true. |
| `external_labels` | map&lt;string, string&gt; |  |  | External labels used to enrich query-planning context or describe result origin. |
| `headers` | map&lt;string, string&gt; |  |  | Non-sensitive HTTP headers for tenant or routing context. Authentication secrets must not be stored here. |
| `properties` | map&lt;string, string&gt; |  |  | Additional non-sensitive Loki configuration, stored as key-value pairs. |
| `tags` | map&lt;string, string&gt; |  |  | Tags used to describe this Loki storage. |
