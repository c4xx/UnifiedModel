package query

import (
	"strings"
	"testing"

	"github.com/alibaba/UnifiedModel/internal/query/planrender"
	"github.com/alibaba/UnifiedModel/pkg/model"
)

func lokiStorage(spec map[string]any) model.UModelElement {
	if spec == nil {
		spec = map[string]any{}
	}
	if _, ok := spec["endpoint"]; !ok {
		spec["endpoint"] = "http://loki:3100"
	}
	return model.UModelElement{Kind: "loki", Domain: "env", Name: "logs", Spec: spec}
}

func renderLoki(t *testing.T, req planrender.Request) map[string]any {
	t.Helper()
	req.Method = planrender.MethodGetLogs
	out, err := lokiLogsRenderer{}.Render(req)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return out
}

func renderLokiErr(t *testing.T, req planrender.Request) error {
	t.Helper()
	req.Method = planrender.MethodGetLogs
	out, err := lokiLogsRenderer{}.Render(req)
	if err == nil {
		t.Fatalf("expected an error, got a plan: %v", out)
	}
	return err
}

// 实体数据按 data_link / storage_link 的 fields_mapping 变成流选择器；
// 存储里的标签名（k8s_cluster）来自 storage_link，不是实体字段名。
func TestLokiSelectorFromEntityData(t *testing.T) {
	out := renderLoki(t, planrender.Request{
		Storage:            lokiStorage(nil),
		DataLinkMapping:    map[string]any{"cluster": "cluster", "namespace": "namespace", "pod": "pod"},
		StorageLinkMapping: map[string]any{"cluster": "k8s_cluster"},
		EntityData: &model.EntityData{
			Header: []string{"cluster", "namespace", "pod"},
			Data:   [][]string{{"local", "demo", "api-1"}},
		},
	})
	want := `{k8s_cluster="local", namespace="demo", pod="api-1"}`
	if out["query"] != want {
		t.Fatalf("query = %v, want %s", out["query"], want)
	}
	if out["dialect"] != "loki_logql" {
		t.Errorf("dialect = %v", out["dialect"])
	}
	if out["endpoint"] != "http://loki:3100" {
		t.Errorf("endpoint = %v", out["endpoint"])
	}
	if out["api_prefix"] != "/loki/api/v1" {
		t.Errorf("api_prefix default = %v", out["api_prefix"])
	}
}

// 多个值用正则；值里的正则元字符必须转义，否则 a.b 会匹配 aXb。
func TestLokiMultipleValuesUseEscapedRegex(t *testing.T) {
	out := renderLoki(t, planrender.Request{
		Storage:            lokiStorage(nil),
		DataLinkMapping:    map[string]any{"pod": "pod"},
		StorageLinkMapping: map[string]any{},
		EntityData: &model.EntityData{
			Header: []string{"pod"},
			Data:   [][]string{{"a.b"}, {"c|d"}},
		},
	})
	// 字符串字面量里反斜杠要再转义一次
	want := `{pod=~"a\\.b|c\\|d"}`
	if out["query"] != want {
		t.Fatalf("query = %v, want %s", out["query"], want)
	}
}

// 值里的引号、反斜杠、换行不能逃出字符串字面量。
func TestLokiValueEscaping(t *testing.T) {
	out := renderLoki(t, planrender.Request{
		Storage:         lokiStorage(nil),
		DataFilter:      "",
		EntityIDs:       []string{"x\"} | drop\n\\"},
		DataLinkMapping: map[string]any{"id": "pod"},
	})
	q := out["query"].(string)
	if want := `{pod="x\"} | drop\n\\"}`; q != want {
		t.Fatalf("query = %s, want %s", q, want)
	}
}

func TestLokiFiltersEqNeqNotIn(t *testing.T) {
	out := renderLoki(t, planrender.Request{
		Storage:     lokiStorage(map[string]any{"search_filter": "tenant = acme"}),
		DataFilter:  "level != debug",
		EntityQuery: "ns not in (kube-system, monitoring)",
		MethodQuery: "app = api",
	})
	q := out["query"].(string)
	for _, want := range []string{`tenant="acme"`, `level!="debug"`, `ns!~"kube-system|monitoring"`, `app="api"`} {
		if !strings.Contains(q, want) {
			t.Errorf("query %s lacks %s", q, want)
		}
	}
	if !strings.HasPrefix(q, "{") || !strings.HasSuffix(q, "}") {
		t.Errorf("query is not a bare stream selector: %s", q)
	}
}

// Loki 拒绝只有否定匹配的选择器；在这里提前报错，错误信息指向原因。
func TestLokiRejectsSelectorWithoutPositiveMatcher(t *testing.T) {
	err := renderLokiErr(t, planrender.Request{Storage: lokiStorage(nil), DataFilter: "level != debug"})
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("error should explain the missing positive matcher: %v", err)
	}
	// 完全没有约束同理：不能渲染成匹配所有日志流的查询。
	err = renderLokiErr(t, planrender.Request{Storage: lokiStorage(nil)})
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("empty constraints should be refused too: %v", err)
	}
}

// 解析不了的过滤不能悄悄丢掉：那会查出比要求更宽的日志。
func TestLokiRefusesUnparseableFilter(t *testing.T) {
	err := renderLokiErr(t, planrender.Request{
		Storage:         lokiStorage(nil),
		EntityIDs:       []string{"p1"},
		DataLinkMapping: map[string]any{"id": "pod"},
		DataFilter:      "a = 1 or b = 2",
	})
	if !strings.Contains(err.Error(), "filters") {
		t.Errorf("error should mention the unsupported filters: %v", err)
	}
}

// 标签名要符合 Loki 的规则；映射里的恶意字段名不能拼进查询。
func TestLokiRejectsInvalidLabelName(t *testing.T) {
	err := renderLokiErr(t, planrender.Request{
		Storage:            lokiStorage(nil),
		EntityIDs:          []string{"p1"},
		DataLinkMapping:    map[string]any{"id": "pod"},
		StorageLinkMapping: map[string]any{"pod": `pod"} or {a="b`},
	})
	if !strings.Contains(err.Error(), "label") {
		t.Errorf("error should name the invalid label: %v", err)
	}
}

func TestLokiLimitDirectionAndTenant(t *testing.T) {
	base := planrender.Request{
		EntityIDs:       []string{"p1"},
		DataLinkMapping: map[string]any{"id": "pod"},
	}

	req := base
	req.Storage = lokiStorage(nil)
	out := renderLoki(t, req)
	if out["limit"] != 1000 || out["direction"] != "backward" {
		t.Errorf("defaults: limit=%v direction=%v", out["limit"], out["direction"])
	}

	req = base
	req.Storage = lokiStorage(map[string]any{"default_limit": 200, "default_direction": "forward", "tenant": "t1", "tenant_header": "X-Scope-OrgID", "api_prefix": "/api/prom"})
	out = renderLoki(t, req)
	if out["limit"] != 200 || out["direction"] != "forward" || out["tenant"] != "t1" || out["tenant_header"] != "X-Scope-OrgID" || out["api_prefix"] != "/api/prom" {
		t.Errorf("storage settings not applied: %v", out)
	}

	req = base
	req.Storage = lokiStorage(map[string]any{"default_limit": 200})
	req.Limit = 50
	out = renderLoki(t, req)
	if out["limit"] != 50 {
		t.Errorf("request limit should win, got %v", out["limit"])
	}
}

// 只处理 get_logs；get_metrics 不属于这个家族。
func TestLokiRendererOnlySupportsLogs(t *testing.T) {
	r := lokiLogsRenderer{}
	if r.Family() != "label-logs" {
		t.Errorf("family = %q", r.Family())
	}
	if !r.SupportsMethod(planrender.MethodGetLogs) || r.SupportsMethod(planrender.MethodGetMetrics) {
		t.Error("label-logs should render get_logs only")
	}
}

// loki 这个 kind 不用声明 spec.family 就能走到渲染器。
func TestLokiKindRoutesByDefault(t *testing.T) {
	if got := defaultFamilyForKind("loki"); got != "label-logs" {
		t.Fatalf("defaultFamilyForKind(loki) = %q", got)
	}
	e := NewExecutor(nil)
	logSet := model.UModelElement{Kind: "log_set", Name: "k8s.log.pod"}
	out, err := e.buildLogStorageQuery(logSet, lokiStorage(nil), map[string]any{"id": "pod"}, nil, []string{"p1"}, "", "", "", nil, 100)
	if err != nil {
		t.Fatalf("buildLogStorageQuery: %v", err)
	}
	if out["dialect"] != "loki_logql" || out["query"] != `{pod="p1"}` {
		t.Fatalf("not routed to the LogQL renderer: %v", out)
	}
}
