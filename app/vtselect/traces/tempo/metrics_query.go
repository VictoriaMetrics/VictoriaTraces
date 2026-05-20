package tempo

import (
	"fmt"
	"strings"

	"github.com/VictoriaMetrics/VictoriaTraces/lib/traceql"
)

// metricsQueryTranslation holds the result of translating a TraceQL metrics query.
type metricsQueryTranslation struct {
	baseQuery string   // LogsQL stats query (for non-compare queries)
	byFields  []string // mapped VT field names
	isCompare bool

	// scaleDurationToSeconds is true when the aggregation is over the `duration` field
	// (stored as nanoseconds). Sample values need /1e9 so Grafana renders them as seconds.
	scaleDurationToSeconds bool

	// valueColumnLabelKey, when non-empty, instructs the executor to attach a
	// synthetic label `{valueColumnLabelKey: valueColumnLabels[columnName]}` to
	// each emitted point. Used for quantile_over_time where Tempo returns one
	// series per quantile with a `p=<value>` label.
	valueColumnLabelKey string
	valueColumnLabels   map[string]string // LogsQL column alias → label value

	// compare-specific fields (only set when isCompare=true)
	baseFilter       string // LogsQL filter for base (outer filter)
	compareFilter    string // LogsQL filter for selection (inner filter)
	topN             int
	selectionStartNs int64
	selectionEndNs   int64
}

// translateMetricsQuery translates a TraceQL metrics query into LogsQL stats query string(s).
//
// For compare() queries, both baseQuery and selectionQuery are populated.
// For all other queries, only baseQuery is populated.
func translateMetricsQuery(traceQLStr string, timestamp int64) (*metricsQueryTranslation, error) {
	q, err := traceql.ParseQueryAtTimestamp(traceQLStr, timestamp)
	if err != nil {
		return nil, fmt.Errorf("cannot parse TraceQL query: %w", err)
	}

	funcName, fieldName, quantiles, compareParams, traceQLByFields, err := q.MetricsComponents()
	if err != nil {
		return nil, err
	}

	// Only span streams should be taken into account. Internal streams such as index and service graph streams
	// must be excluded.
	baseFilterStr := `{resource_attr:service.name!=""} AND `

	// Get the filter part in LogsQL format.
	filterStr := q.Filter()
	if filterStr == "*" || filterStr == "" {
		filterStr = "*"
	}

	filterStr = baseFilterStr + filterStr

	// Map the by-fields from TraceQL to VT field names.
	vtByFields := make([]string, len(traceQLByFields))
	for i, f := range traceQLByFields {
		vtByFields[i] = traceql.TraceQLFieldToVTField(f)
	}

	result := &metricsQueryTranslation{
		byFields:   vtByFields,
		baseFilter: filterStr, // used by exemplar collection and compare queries
	}

	if funcName == "compare" && compareParams != nil {
		result.isCompare = true
		result.baseFilter = filterStr
		result.compareFilter = compareParams.Filter
		result.topN = compareParams.TopN
		result.selectionStartNs = compareParams.StartNs
		result.selectionEndNs = compareParams.EndNs
		return result, nil
	}

	// Standard metrics query.
	statsExpr, valueColumnLabels, err := metricsToLogsQLStats(funcName, fieldName, quantiles)
	if err != nil {
		return nil, err
	}
	result.baseQuery = buildStatsQuery(filterStr, vtByFields, statsExpr)
	if len(valueColumnLabels) > 0 {
		// Tempo emits `quantile_over_time` series with a `p=<quantile>` label,
		// regardless of whether one or many quantiles were requested.
		result.valueColumnLabelKey = "p"
		result.valueColumnLabels = valueColumnLabels
	}

	// Duration values are stored as nanoseconds in VictoriaTraces, but Grafana
	// renders metric values on duration panels as seconds. Aggregations like
	// min/max/avg/sum/quantile over duration need to be scaled down.
	// histogram_over_time already produces bucket labels in seconds.
	if fieldName == "duration" && funcName != "histogram_over_time" && funcName != "count_over_time" && funcName != "rate" {
		result.scaleDurationToSeconds = true
	}
	return result, nil
}

// buildStatsQuery assembles a LogsQL stats query from filter, by-fields, and stats expression.
func buildStatsQuery(filterStr string, vtByFields []string, statsExpr string) string {
	var sb strings.Builder
	sb.WriteString(filterStr)
	sb.WriteString(" | stats")
	if len(vtByFields) > 0 {
		sb.WriteString(" by (")
		for i, f := range vtByFields {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(quoteLogsQLField(f))
		}
		sb.WriteString(")")
	}
	sb.WriteString(" ")
	sb.WriteString(statsExpr)
	return sb.String()
}

// metricsToLogsQLStats converts a TraceQL metrics function to its LogsQL stats expression.
//
// For quantile_over_time it returns one `quantile(...) as <alias>` clause per
// quantile and a map of alias→quantile-value so the executor can attach a `p`
// label that distinguishes the resulting series.
func metricsToLogsQLStats(funcName, fieldName string, quantiles []string) (statsExpr string, valueColumnLabels map[string]string, err error) {
	vtField := quoteLogsQLField(traceql.TraceQLFieldToVTField(fieldName))
	switch funcName {
	case "rate":
		return "rate() as value", nil, nil
	case "count_over_time":
		return "count() as value", nil, nil
	case "min_over_time":
		return "min(" + vtField + ") as value", nil, nil
	case "max_over_time":
		return "max(" + vtField + ") as value", nil, nil
	case "avg_over_time":
		return "avg(" + vtField + ") as value", nil, nil
	case "sum_over_time":
		return "sum(" + vtField + ") as value", nil, nil
	case "histogram_over_time":
		return "histogram(" + vtField + ") as value", nil, nil
	case "quantile_over_time":
		if len(quantiles) == 0 {
			return "", nil, fmt.Errorf("quantile_over_time requires at least one quantile")
		}
		var sb strings.Builder
		labels := make(map[string]string, len(quantiles))
		for i, q := range quantiles {
			if i > 0 {
				sb.WriteString(", ")
			}
			alias := quantileAlias(i)
			sb.WriteString("quantile(")
			sb.WriteString(q)
			sb.WriteString(", ")
			sb.WriteString(vtField)
			sb.WriteString(") as ")
			sb.WriteString(alias)
			labels[alias] = q
		}
		return sb.String(), labels, nil
	default:
		return "", nil, fmt.Errorf("unsupported metrics function: %s", funcName)
	}
}

// quantileAlias returns a deterministic LogsQL-safe alias for the i-th
// quantile in a quantile_over_time output. It is decoupled from the
// quantile value (which can contain '.') to keep aliases identifier-safe.
func quantileAlias(i int) string {
	return fmt.Sprintf("p_%d", i)
}

// quoteLogsQLField quotes a field name for LogsQL if it contains special characters.
func quoteLogsQLField(s string) string {
	for _, c := range s {
		if c == ':' || c == ' ' || c == '"' || c == '(' || c == ')' || c == '|' || c == ',' {
			return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
		}
	}
	return s
}
