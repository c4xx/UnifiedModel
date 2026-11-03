# SDK 与客户端指南

English: [SDK And Client Guide](../../en/guides/sdk-clients.md)

公共客户端入口：REST、CLI、MCP、生成模型 SDK 和最小 Go REST client。


## REST API

REST 面向需要直接控制 HTTP 的集成。

契约：

- [OpenAPI](../../../api/openapi/openapi.yaml)

常用 endpoint：

```http
POST /api/v1/workspaces
POST /api/v1/umodel/{workspace}/import
POST /api/v1/entitystore/{workspace}/entities:write
POST /api/v1/query/{workspace}/execute
GET  /api/v1/agent/{workspace}/discover
```

## CLI

`umctl` 覆盖本地开发、示例和文档工作流：

```bash
go run ./cmd/umctl --addr http://localhost:8080 workspace create demo '{"name":"Demo"}'
go run ./cmd/umctl --addr http://localhost:8080 query run demo ".umodel | limit 5"
```

参考：[CLI 参考](../reference/cli.md)。

## Go REST Client

最小 Go REST client 位于 [sdk/go/service](../../../sdk/go/service)。

```go
client := service.NewClient("http://localhost:8080")
ctx := context.Background()

_, err := client.CreateWorkspace(ctx, service.CreateWorkspaceRequest{
    ID:   "demo",
    Name: "Demo",
})
if err != nil {
    return err
}

result, err := client.Query(ctx, "demo", service.QueryRequest{
    Query: ".umodel | limit 5",
})
if err != nil {
    return err
}
_ = result
```

运行测试：

```bash
cd sdk/go
go test ./service
```

## 生成的模型 SDK

生成的模型 SDK 表示 UModel schema 类型：

- Go：[sdk/go/umodel](../../../sdk/go/umodel/README.md)
- Python：[sdk/python/umodel](../../../sdk/python/umodel/README.zh-CN.md)
- Java：[generated/java](../../../generated/java)

重新生成和验证：

```bash
make expand
make verify
```

可运行示例：

- [SDK 示例](../../../examples/sdk/README.zh-CN.md)
- [SDK 端到端示例流程](../../../examples/sdk/full-flow.zh-CN.md)
- Go 模型检查示例：[examples/sdk/go/model-inspector](../../../examples/sdk/go/model-inspector)
- Go REST quick start：[examples/sdk/go/service-quickstart](../../../examples/sdk/go/service-quickstart)
- Python 模型检查示例：[examples/sdk/python/inspect_model_pack.py](../../../examples/sdk/python/inspect_model_pack.py)

## 生成 SDK 集成示例

生成 SDK 工作流：解析模型包、提前发现未知 kind 或 envelope 缺失、读取模型元数据，再把已检查的模型包交给 REST API 或 CLI。运行时实体写入和对象图查询保持在公开服务契约之后。

### Go 中解析并校验模型文件

仓库内 `sdk/go` 示例使用 `umodel_go_cli` 模块路径。外部工程将该路径替换为发布或 vendored 的 SDK 模块路径。

```go
package main

import (
    "fmt"
    "os"

    umodel "umodel_go_cli/umodel"
)

func main() {
    data, err := os.ReadFile("examples/quickstart-multidomain/devops/entity_set/devops.service.yaml")
    if err != nil {
        panic(err)
    }

    obj, err := umodel.ParseYamlUModel(data)
    if err != nil {
        panic(err)
    }
    if err := obj.Validate(); err != nil {
        panic(err)
    }

    metadata := obj.GetMetadata()
    fmt.Printf("%s %s/%s\n", obj.GetKind(), metadata.Domain, metadata.Name)
}
```

### Python 中扫描模型包

从仓库根目录运行时可设置 `PYTHONPATH=sdk/python`，或将生成的 Python SDK 打包进应用环境。

```python
from pathlib import Path

from umodel import (
    get_link_endpoints,
    get_object_metadata,
    is_link_object,
    parse_umodel_yaml,
)

for path in Path("examples/quickstart-multidomain").rglob("*.yaml"):
    obj = parse_umodel_yaml(path.read_bytes())
    metadata = get_object_metadata(obj)

    if is_link_object(obj):
        src, dest = get_link_endpoints(obj)
        if src and dest:
            print(f"{path}: {obj.get_kind()} {src.name} -> {dest.name}")
        else:
            print(f"{path}: {obj.get_kind()} {metadata['domain']}/{metadata['name']}")
    else:
        print(f"{path}: {obj.get_kind()} {metadata['domain']}/{metadata['name']}")
```

### 与 REST Client 组合使用

生成模型 SDK 负责本地校验和元数据检查。服务 client 负责 workspace 导入和运行时查询。

```go
client := service.NewClient("http://localhost:8080")
ctx := context.Background()

_, err := client.CreateWorkspace(ctx, service.CreateWorkspaceRequest{
    ID:   "demo",
    Name: "Demo",
})
if err != nil {
    return err
}

_, err = client.ImportUModel(ctx, "demo", service.UModelImportRequest{
    Path: "examples/quickstart-multidomain",
})
if err != nil {
    return err
}

result, err := client.Query(ctx, "demo", service.QueryRequest{
    Query: ".umodel with(kind='entity_set') | limit 5",
})
if err != nil {
    return err
}
_ = result
```

集成方式：

| 场景 | SDK 负责 | 服务负责 |
|---|---|---|
| 模型包 CI | 解析、校验、列出模型元数据 | 可选的导入冒烟测试 |
| 平台启动 | 加载内置模型定义，遇到不兼容 schema 时提前失败 | 导入目标 workspace |
| 开发者工具 | JSON/YAML 转换、kind 检查、Link 源/目标展示 | 查询 live workspace 状态 |
| Agent 应用 | 读取静态模型元数据，供 prompt 或 tool 描述使用 | 通过 Query Service 或 MCP 获取运行时对象图上下文 |

## MCP

MCP 服务需要 UModel discovery、resources、query examples 或 query tools 的 Agent client。

```bash
go run ./cmd/umodel-mcp --data data --graphstore file.memory
```

当 client 需要 HTTP MCP server 时，使用 Streamable HTTP：

```bash
go run ./cmd/umodel-mcp --transport http --addr 127.0.0.1:8090 --data data --graphstore file.memory
```

MCP envelope 保持 JSON-RPC。Tool/resource 文本 payload 使用 TOON（`text/toon`），tool call 同时返回 `structuredContent` 供偏 JSON 的 client 使用。

参考：[MCP 参考](../reference/mcp.md)。示例：[examples/mcp](../../../examples/mcp/README.zh-CN.md)。

## 嵌入到 Go 程序中

Go 服务可以在进程内运行 UModel，把各服务当作函数调用，不需要第二个进程，也没有 HTTP 往返。`pkg/embed` 返回的 `Runtime` 的字段都是 `pkg/contract` 里的公开接口，宿主程序不会接触到任何 internal 包的类型。

```go
import (
	"github.com/alibaba/UnifiedModel/pkg/embed"
	"github.com/alibaba/UnifiedModel/pkg/graphstore"
	"github.com/alibaba/UnifiedModel/pkg/model"
)

rt, err := embed.New("/data", graphstore.ProviderConfig{Type: "file.memory"})
if err != nil {
	return err
}
defer rt.Close()

out, err := rt.Query.Execute(ctx, "prod", model.QueryRequest{Query: ".umodel | limit 5"})
```

| 字段 | 接口 | 用途 |
|---|---|---|
| `Workspace` | `contract.WorkspaceManager` | 创建和列出 workspace |
| `UModel` | `contract.UModelService` | 校验、导入和写入模型元素 |
| `Entities` | `contract.EntityWriteService` | 写入、过期和删除实体与关系 |
| `Query` | `contract.QueryService` | 唯一读取路径：`.umodel`、`.entity`、`.entity_set`、`.topo` |
| `Search` | `contract.SearchService` | 关键词、向量和混合检索 |
| `Agent` | `contract.AgentGateway` | Agent 发现、工具和资源 |

- 使用自己的 `GraphStore` 时，先用 `pkg/graphstore.RegisterProvider` 注册，再调用 `embed.New`。见 [GraphStore Providers](../graphstore-providers.md)。
- `Runtime.Handler()` 用同一组服务提供 REST API 和 Web UI 端点。它是可选的，需要用 UModel 自带工具调试时再挂载。
- 各服务在编译期与 `pkg/contract` 做了断言，服务的方法签名变化会让构建失败，而不是让宿主程序出错。

## 如何选择

| 需求 | 入口 |
|---|---|
| 人工本地工作流 | CLI 或 Web UI |
| HTTP 集成 | REST API 或 Go REST client |
| Go 服务内直接函数调用 | `pkg/embed` |
| 构造模型对象 | 生成的 SDK |
| Agent 集成 | MCP / AgentGateway |
| 契约检查 | OpenAPI 和 MCP schema |
