package query

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alibaba/UnifiedModel/internal/query/planrender"
	apperrors "github.com/alibaba/UnifiedModel/pkg/errors"
)

// lokiLogsRenderer renders the label-logs query family: log backends queried
// with LogQL over HTTP (Grafana Loki and API-compatible stores).
//
// Like the sql-table renderer it consumes the shared Constraint IR
// (ResolveConstraints) instead of re-implementing entity binding and filter
// parsing. The IR is formatted as a LogQL stream selector: every constraint
// becomes a label matcher, so the plan selects log streams by label only. It
// does not append line filters (|=, |~); a caller that wants to filter the line
// text adds them to the returned query.
//
// The service returns a plan, not a result. The plan carries the endpoint, API
// prefix, limit, direction and tenant header for the executor to run; time
// range stays in the plan envelope, as it does for the PromQL and SQL plans.
type lokiLogsRenderer struct{}

const defaultLokiLimit = 1000

// lokiLabelName is the label-name grammar Loki accepts. Names come from the
// dataset/storage field mappings, not from a query, so an unchecked name could
// otherwise break out of the selector.
var lokiLabelName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func (lokiLogsRenderer) Family() string { return "label-logs" }

func (lokiLogsRenderer) SupportsMethod(m planrender.Method) bool {
	return m == planrender.MethodGetLogs
}

func (lokiLogsRenderer) Render(req planrender.Request) (map[string]any, error) {
	constraints, raw := ResolveConstraints(req)
	// Fail closed, as the sql-table renderer does: a filter the IR cannot express
	// would be silently dropped and the plan would select more streams than asked.
	if len(raw) > 0 {
		return nil, apperrors.WithDetails(apperrors.CodeQueryPlanError,
			"label-logs renderer cannot express some filters; refusing to emit an under-constrained query",
			map[string]string{"unsupported_filters": strings.Join(raw, " ; ")})
	}

	selector, err := lokiSelector(constraints)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = intValue(req.Storage.Spec["default_limit"])
	}
	if limit <= 0 {
		limit = defaultLokiLimit
	}
	out := map[string]any{
		"dialect":    "loki_logql",
		"endpoint":   req.Storage.Spec["endpoint"],
		"api_prefix": firstNonEmpty(stringValue(req.Storage.Spec["api_prefix"]), "/loki/api/v1"),
		"query":      selector,
		"direction":  firstNonEmpty(stringValue(req.Storage.Spec["default_direction"]), "backward"),
		"limit":      limit,
	}
	if tenant := stringValue(req.Storage.Spec["tenant"]); tenant != "" {
		out["tenant"] = tenant
	}
	if tenantHeader := stringValue(req.Storage.Spec["tenant_header"]); tenantHeader != "" {
		out["tenant_header"] = tenantHeader
	}
	return out, nil
}

// lokiSelector formats constraints as a LogQL stream selector.
func lokiSelector(constraints []Constraint) (string, error) {
	matchers := make([]string, 0, len(constraints))
	positive := false
	for _, c := range constraints {
		if !lokiLabelName.MatchString(c.Field) {
			return "", apperrors.WithDetails(apperrors.CodeQueryPlanError,
				"label-logs renderer: field is not a valid Loki label name",
				map[string]string{"label": c.Field})
		}
		op, value, isPositive, err := lokiMatcher(c)
		if err != nil {
			return "", err
		}
		positive = positive || isPositive
		matchers = append(matchers, c.Field+op+value)
	}
	// Loki rejects a selector with no positive matcher ("queries require at least
	// one regexp or equality matcher that does not have an empty-compatible
	// value"). Fail here with the cause rather than at the backend, and never
	// emit a selector that would match every stream.
	if !positive {
		return "", apperrors.New(apperrors.CodeQueryPlanError,
			"label-logs renderer needs at least one positive (equality or regexp) matcher to select log streams; bind the entity to a stream label through the data/storage link field mappings")
	}
	return "{" + strings.Join(matchers, ", ") + "}", nil
}

// lokiMatcher returns the operator, the quoted value, and whether the matcher
// is positive (= or =~) for one constraint.
func lokiMatcher(c Constraint) (op, value string, positive bool, err error) {
	switch c.Op {
	case "eq":
		if len(c.Values) != 1 {
			return "", "", false, lokiOpError(c)
		}
		return "=", quoteLogQL(c.Values[0]), true, nil
	case "neq":
		if len(c.Values) != 1 {
			return "", "", false, lokiOpError(c)
		}
		return "!=", quoteLogQL(c.Values[0]), false, nil
	case "in":
		if len(c.Values) == 1 {
			return "=", quoteLogQL(c.Values[0]), true, nil
		}
		return "=~", quoteLogQL(regexAlternation(c.Values)), true, nil
	case "notin":
		if len(c.Values) == 1 {
			return "!=", quoteLogQL(c.Values[0]), false, nil
		}
		return "!~", quoteLogQL(regexAlternation(c.Values)), false, nil
	default:
		return "", "", false, lokiOpError(c)
	}
}

func lokiOpError(c Constraint) error {
	return apperrors.WithDetails(apperrors.CodeQueryPlanError,
		"label-logs renderer cannot express this constraint",
		map[string]string{"field": c.Field, "op": c.Op, "values": fmt.Sprint(len(c.Values))})
}

// regexAlternation joins values into an anchored-by-Loki alternation. Loki
// anchors label regexps itself, so no ^$ is added; each value is escaped so a
// dot or pipe inside it stays literal.
func regexAlternation(values []string) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = regexp.QuoteMeta(v)
	}
	return strings.Join(parts, "|")
}

// quoteLogQL returns s as a double-quoted LogQL string literal.
func quoteLogQL(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(s) + `"`
}
