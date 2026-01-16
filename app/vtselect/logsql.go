package vtselect

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/metrics"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/logsql"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage"
)

// ---------------------------- LogsQL Dependency-----------------------------
// VictoriaLogs and VictoriaTraces share the query language LogsQL before TracesQL is available.
// The LogsQL related functions can be updated after a new version of VictoriaLogs is released.
//
// steps:
// 1. copy-paste `vlselect/logsql/` and replace vlstorage import to vtstorage.
// 2. copy-paste LogsQL handlers in `vlselect/main.go` and replace logsql import from VictoriaLogs repo to VictoriaTraces repo.
// 3. replace metrics prefix from `vl_` to `vt_`.

var (
	logsqlFacetsRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/facets"}`)
	logsqlFacetsDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/logsql/facets"}`)

	logsqlFieldNamesRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/field_names"}`)
	logsqlFieldNamesDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/logsql/field_names"}`)

	logsqlFieldValuesRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/field_values"}`)
	logsqlFieldValuesDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/logsql/field_values"}`)

	logsqlHitsRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/hits"}`)
	logsqlHitsDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/logsql/hits"}`)

	logsqlQueryRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/query"}`)
	logsqlQueryDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/logsql/query"}`)

	logsqlStatsQueryRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/stats_query"}`)
	logsqlStatsQueryDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/logsql/stats_query"}`)

	logsqlStatsQueryRangeRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/stats_query_range"}`)
	logsqlStatsQueryRangeDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/logsql/stats_query_range"}`)

	logsqlStreamFieldNamesRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/stream_field_names"}`)
	logsqlStreamFieldNamesDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/logsql/stream_field_names"}`)

	logsqlStreamFieldValuesRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/stream_field_values"}`)
	logsqlStreamFieldValuesDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/logsql/stream_field_values"}`)

	logsqlStreamIDsRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/stream_ids"}`)
	logsqlStreamIDsDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/logsql/stream_ids"}`)

	logsqlStreamsRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/streams"}`)
	logsqlStreamsDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/logsql/streams"}`)

	tenantIDsRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/tenant_ids"}`)
	tenantIDsDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/tenant_ids"}`)

	// no need to track duration for tail requests, as they usually take long time
	logsqlTailRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/tail"}`)

	// no need to track the duration for query_time_range requests, since they are instant
	logsqlQueryTimeRangeRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/query_time_range"}`)

	// no need to track duration for /delete/* requests, because they are asynchornous
	deleteRunTaskRequests     = metrics.NewCounter(`vt_http_requests_total{path="/delete/run_task"}`)
	deleteStopTaskRequests    = metrics.NewCounter(`vt_http_requests_total{path="/delete/stop_task"}`)
	deleteActiveTasksRequests = metrics.NewCounter(`vt_http_requests_total{path="/delete/active_tasks"}`)

	slowQueries = metrics.NewCounter(`vt_slow_queries_total`)
)

func logRequestErrorIfNeeded(ctx context.Context, w http.ResponseWriter, r *http.Request, startTime time.Time) {
	err := ctx.Err()
	switch err {
	case nil:
		// nothing to do
	case context.Canceled:
		// do not log canceled requests, since they are expected and legal.
	case context.DeadlineExceeded:
		err = &httpserver.ErrorWithStatusCode{
			Err: fmt.Errorf("the request couldn't be executed in %.3f seconds; possible solutions: "+
				"to increase -search.maxQueryDuration=%s; to pass bigger value to 'timeout' query arg", time.Since(startTime).Seconds(), maxQueryDuration),
			StatusCode: http.StatusServiceUnavailable,
		}
		httpserver.Errorf(w, r, "%s", err)
	default:
		httpserver.Errorf(w, r, "unexpected error: %s", err)
	}
}

func processSelectRequest(ctx context.Context, w http.ResponseWriter, r *http.Request, path string) bool {
	httpserver.EnableCORS(w, r)
	startTime := time.Now()
	switch path {
	case "/select/logsql/query_time_range":
		logsqlQueryTimeRangeRequests.Inc()
		logsql.ProcessQueryTimeRangeRequest(ctx, w, r)
		return true
	case "/select/logsql/facets":
		logsqlFacetsRequests.Inc()
		logsql.ProcessFacetsRequest(ctx, w, r)
		logsqlFacetsDuration.UpdateDuration(startTime)
		return true
	case "/select/logsql/field_names":
		logsqlFieldNamesRequests.Inc()
		logsql.ProcessFieldNamesRequest(ctx, w, r)
		logsqlFieldNamesDuration.UpdateDuration(startTime)
		return true
	case "/select/logsql/field_values":
		logsqlFieldValuesRequests.Inc()
		logsql.ProcessFieldValuesRequest(ctx, w, r)
		logsqlFieldValuesDuration.UpdateDuration(startTime)
		return true
	case "/select/logsql/hits":
		logsqlHitsRequests.Inc()
		logsql.ProcessHitsRequest(ctx, w, r)
		logsqlHitsDuration.UpdateDuration(startTime)
		return true
	case "/select/logsql/query":
		logsqlQueryRequests.Inc()
		logsql.ProcessQueryRequest(ctx, w, r)
		logsqlQueryDuration.UpdateDuration(startTime)
		return true
	case "/select/logsql/stats_query":
		logsqlStatsQueryRequests.Inc()
		logsql.ProcessStatsQueryRequest(ctx, w, r)
		logsqlStatsQueryDuration.UpdateDuration(startTime)
		return true
	case "/select/logsql/stats_query_range":
		logsqlStatsQueryRangeRequests.Inc()
		logsql.ProcessStatsQueryRangeRequest(ctx, w, r)
		logsqlStatsQueryRangeDuration.UpdateDuration(startTime)
		return true
	case "/select/logsql/stream_field_names":
		logsqlStreamFieldNamesRequests.Inc()
		logsql.ProcessStreamFieldNamesRequest(ctx, w, r)
		logsqlStreamFieldNamesDuration.UpdateDuration(startTime)
		return true
	case "/select/logsql/stream_field_values":
		logsqlStreamFieldValuesRequests.Inc()
		logsql.ProcessStreamFieldValuesRequest(ctx, w, r)
		logsqlStreamFieldValuesDuration.UpdateDuration(startTime)
		return true
	case "/select/logsql/stream_ids":
		logsqlStreamIDsRequests.Inc()
		logsql.ProcessStreamIDsRequest(ctx, w, r)
		logsqlStreamIDsDuration.UpdateDuration(startTime)
		return true
	case "/select/logsql/streams":
		logsqlStreamsRequests.Inc()
		logsql.ProcessStreamsRequest(ctx, w, r)
		logsqlStreamsDuration.UpdateDuration(startTime)
		return true
	case "/select/tenant_ids":
		tenantIDsRequests.Inc()
		logsql.ProcessTenantIDsRequest(ctx, w, r)
		tenantIDsDuration.UpdateDuration(startTime)
		return true
	default:
		return false
	}
}

func deleteHandler(w http.ResponseWriter, r *http.Request, path string) {
	ctx := r.Context()

	switch path {
	case "/delete/run_task":
		deleteRunTaskRequests.Inc()
		processDeleteRunTaskRequest(ctx, w, r)
	case "/delete/stop_task":
		deleteStopTaskRequests.Inc()
		processDeleteStopTaskRequest(ctx, w, r)
	case "/delete/active_tasks":
		deleteActiveTasksRequests.Inc()
		processDeleteActiveTasksRequest(ctx, w, r)
	default:
		httpserver.Errorf(w, r, "unsupported path requested: %q", path)
	}
}

func processDeleteRunTaskRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tenantID, err := logstorage.GetTenantIDFromRequest(r)
	if err != nil {
		httpserver.Errorf(w, r, "cannot obtain tenantID: %s", err)
		return
	}

	fStr := r.FormValue("filter")
	f, err := logstorage.ParseFilter(fStr)
	if err != nil {
		httpserver.Errorf(w, r, "cannot parse filter [%s]: %s", fStr, err)
		return
	}

	// Generate taskID from the current timestamp in nanoseconds
	timestamp := time.Now().UnixNano()
	taskID := fmt.Sprintf("%d", timestamp)

	tenantIDs := []logstorage.TenantID{tenantID}
	if err := vtstorage.DeleteRunTask(ctx, taskID, timestamp, tenantIDs, f); err != nil {
		httpserver.Errorf(w, r, "cannot run delete task: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"task_id":%q}`, taskID)
}

func processDeleteStopTaskRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	taskID := r.FormValue("task_id")
	if taskID == "" {
		httpserver.Errorf(w, r, "missing task_id arg")
		return
	}

	if err := vtstorage.DeleteStopTask(ctx, taskID); err != nil {
		httpserver.Errorf(w, r, "cannot stop task with task_id=%q: %s", taskID, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok"}`)
}

func processDeleteActiveTasksRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tasks, err := vtstorage.DeleteActiveTasks(ctx)
	if err != nil {
		httpserver.Errorf(w, r, "cannot obtain active delete tasks: %s", err)
		return
	}

	data := logstorage.MarshalDeleteTasksToJSON(tasks)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", data)
}
