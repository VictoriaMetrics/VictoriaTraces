package tempo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/tracecommon"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/traceql"
)

type traceSearchProfileResponse struct {
	Mode                   string                           `json:"mode"`
	TraceQL                string                           `json:"traceql"`
	Start                  time.Time                        `json:"start"`
	End                    time.Time                        `json:"end"`
	Limit                  int                              `json:"limit"`
	PlanningDurationNsecs  int64                            `json:"planning_duration_nsecs"`
	ExecutionDurationNsecs int64                            `json:"execution_duration_nsecs,omitempty"`
	Stages                 []traceSearchProfileStage        `json:"stages"`
	Analysis               *traceSearchAnalysis             `json:"analysis,omitempty"`
	Execution              *logstorage.QueryProfileSnapshot `json:"execution,omitempty"`
	Warnings               []string                         `json:"warnings"`
	Error                  string                           `json:"error,omitempty"`
}

type traceSearchProfileStage struct {
	Name          string                     `json:"name"`
	Operation     string                     `json:"operation"`
	DependsOn     []string                   `json:"depends_on"`
	Description   string                     `json:"description"`
	QueryTemplate string                     `json:"query_template"`
	Plan          logstorage.QueryStaticPlan `json:"plan"`
	Dynamic       bool                       `json:"dynamic"`
}

func processProfileRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	planningStart := time.Now()
	cp, err := tracecommon.GetCommonParams(r)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}
	params, err := parseTempoAPIParam(ctx, r, true, *tracecommon.TraceMaxTraces)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}
	if params.q == "" {
		params.q = "{}"
	}
	mode, err := getProfileMode(r)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect profile mode: %s", err)
		return
	}

	filterQuery, err := traceql.ParseQuery(params.q)
	if err != nil {
		httpserver.Errorf(w, r, "cannot parse TraceQL query: %s", err)
		return
	}
	if filterQuery.HasNonHintPipe() {
		httpserver.Errorf(w, r, "Tempo trace search profiling accepts filters and display hints only: %s", params.q)
		return
	}
	filterQuery.StripHintPipes()

	stages, err := buildTraceSearchProfileStages(filterQuery, params.start, params.end, params.limit)
	if err != nil {
		httpserver.Errorf(w, r, "cannot build query plan: %s", err)
		return
	}
	resp := traceSearchProfileResponse{
		Mode:                  mode,
		TraceQL:               params.q,
		Start:                 params.start,
		End:                   params.end,
		Limit:                 params.limit,
		PlanningDurationNsecs: time.Since(planningStart).Nanoseconds(),
		Stages:                stages,
		Warnings: []string{
			"operator durations are summed wall-clock activity across workers, not CPU time",
			"exclusive_active_duration_nsecs excludes synchronous downstream forwarding but can include internal waits",
			"bloom-filter row selectivity and stream-index bytes are omitted because they cannot yet be measured exactly",
		},
	}

	statusCode := http.StatusOK
	if mode == "analyze" {
		collector := logstorage.NewQueryProfileCollector()
		cp.ProfileCollector = collector
		executionStart := time.Now()
		analysis, runErr := analyzeTraceSearch(ctx, cp, filterQuery, params.start, params.end, params.limit)
		resp.ExecutionDurationNsecs = time.Since(executionStart).Nanoseconds()
		resp.Analysis = &analysis
		snapshot := collector.Snapshot()
		if runErr != nil {
			resp.Error = runErr.Error()
			snapshot.Error = runErr.Error()
			statusCode = http.StatusInternalServerError
		}
		resp.Execution = &snapshot
	}

	data, err := json.Marshal(resp)
	if err != nil {
		httpserver.Errorf(w, r, "cannot marshal profile response: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(data)
}

func getProfileMode(r *http.Request) (string, error) {
	mode := strings.ToLower(r.FormValue("mode"))
	if mode == "" {
		analyzeStr := r.FormValue("analyze")
		if analyzeStr == "" {
			return "explain", nil
		}
		analyze, err := strconv.ParseBool(analyzeStr)
		if err != nil {
			return "", fmt.Errorf("cannot parse analyze=%q: %w", analyzeStr, err)
		}
		if analyze {
			return "analyze", nil
		}
		return "explain", nil
	}
	if mode != "explain" && mode != "analyze" {
		return "", fmt.Errorf("unsupported mode %q; want explain or analyze", mode)
	}
	return mode, nil
}

func buildTraceSearchProfileStages(filterQuery *traceql.Query, start, end time.Time, limit int) ([]traceSearchProfileStage, error) {
	traceIDQuery, adjustedEnd, err := newTraceIDListQuery(filterQuery, end, limit)
	if err != nil {
		return nil, err
	}
	traceIDQuery = traceIDQuery.CloneWithTimeFilter(time.Now().UnixNano(), start.UnixNano(), adjustedEnd.UnixNano())
	traceIDPlan := logstorage.GetQueryStaticPlan(traceIDQuery)

	spanQuery, err := newTraceSpansQuery(filterQuery, []string{"PROFILE_TRACE_ID"}, time.Now(), start, end)
	if err != nil {
		return nil, err
	}
	spanPlan := logstorage.GetQueryStaticPlan(spanQuery)

	return []traceSearchProfileStage{
		{
			Name:          "trace_id_search",
			Operation:     "expanding_time_range_scan",
			DependsOn:     []string{},
			Description:   "Searches recent windows first, increasing each window by 5x until enough trace IDs are found or the requested start is reached.",
			QueryTemplate: traceIDQuery.String(),
			Plan:          traceIDPlan,
			Dynamic:       true,
		},
		{
			Name:          "span_fetch",
			Operation:     "dependent_storage_scan",
			DependsOn:     []string{"trace_id_search"},
			Description:   "Fetches spans for selected trace IDs and roots; the runtime lower bound comes from the oldest selected trace and is widened by search.traceMaxDurationWindow.",
			QueryTemplate: spanQuery.String(),
			Plan:          spanPlan,
			Dynamic:       true,
		},
	}, nil
}
