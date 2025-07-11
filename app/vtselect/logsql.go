package vtselect

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/metrics"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/logsql"
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

	// no need to track duration for tail requests, as they usually take long time
	logsqlTailRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/logsql/tail"}`)
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
	default:
		return false
	}
}
