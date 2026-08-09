package jaeger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/VictoriaMetrics/metrics"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/query"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/tracecommon"
)

// Jaeger Query API v3 metrics
var (
	jaegerV3ServicesRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/jaeger/api/v3/services"}`)
	jaegerV3ServicesDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/jaeger/api/v3/services"}`)

	jaegerV3OperationsRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/jaeger/api/v3/operations"}`)
	jaegerV3OperationsDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/jaeger/api/v3/operations"}`)

	jaegerV3TracesRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/jaeger/api/v3/traces"}`)
	jaegerV3TracesDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/jaeger/api/v3/traces"}`)

	jaegerV3TraceRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/jaeger/api/v3/traces/*"}`)
	jaegerV3TraceDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/jaeger/api/v3/traces/*"}`)
)

// defaultSpanKind is reported for every operation, since the span name list is stored without
// the span kind. Jaeger reports the same value when its storage doesn't provide one.
// See https://github.com/jaegertracing/jaeger/blob/main/cmd/jaeger/internal/extension/jaegerquery/internal/apiv3/http_gateway.go
const defaultSpanKind = "internal"

// defaultSearchDepth is the number of traces returned by /api/v3/traces when the request
// doesn't ask for a specific amount.
const defaultSearchDepth = 100

// requestHandlerV3 handles the Jaeger HTTP API v3 requests.
//
// The v3 API is defined as a gRPC service and served over HTTP by a grpc-gateway in Jaeger,
// so both the paths and the response shapes come from the api_v3 query service proto.
// Jaeger UI 2.15 and newer calls these paths instead of the v1 ones.
// See:
// 1. https://github.com/jaegertracing/jaeger-idl/blob/main/proto/api_v3/query_service.proto
// 2. https://github.com/jaegertracing/jaeger/blob/main/cmd/jaeger/internal/extension/jaegerquery/internal/apiv3/http_gateway.go
func requestHandlerV3(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	startTime := time.Now()
	path := r.URL.Path
	if path == "/select/jaeger/api/v3/services" {
		jaegerV3ServicesRequests.Inc()
		processGetServicesRequestV3(ctx, w, r)
		jaegerV3ServicesDuration.UpdateDuration(startTime)
		return true
	} else if path == "/select/jaeger/api/v3/operations" {
		jaegerV3OperationsRequests.Inc()
		processGetOperationsRequestV3(ctx, w, r)
		jaegerV3OperationsDuration.UpdateDuration(startTime)
		return true
	} else if path == "/select/jaeger/api/v3/traces" {
		jaegerV3TracesRequests.Inc()
		processFindTracesRequestV3(ctx, w, r)
		jaegerV3TracesDuration.UpdateDuration(startTime)
		return true
	} else if strings.HasPrefix(path, "/select/jaeger/api/v3/traces/") && len(path) > len("/select/jaeger/api/v3/traces/") {
		jaegerV3TraceRequests.Inc()
		processGetTraceRequestV3(ctx, w, r)
		jaegerV3TraceDuration.UpdateDuration(startTime)
		return true
	}
	return false
}

// processGetServicesRequestV3 handles the Jaeger /api/v3/services API request.
func processGetServicesRequestV3(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cp, err := tracecommon.GetCommonParams(r)
	if err != nil {
		writeErrorResponseV3(w, http.StatusBadRequest, fmt.Sprintf("incorrect query params: %s", err))
		return
	}

	serviceList, err := query.GetServiceNameList(ctx, cp)
	if err != nil {
		writeErrorResponseV3(w, http.StatusInternalServerError, fmt.Sprintf("cannot get services list: %s", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	WriteGetServicesResponseV3(w, serviceList)
}

// processGetOperationsRequestV3 handles the Jaeger /api/v3/operations API request.
//
// Unlike the v1 API, the service name is passed in the `service` query arg instead of the path.
func processGetOperationsRequestV3(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cp, err := tracecommon.GetCommonParams(r)
	if err != nil {
		writeErrorResponseV3(w, http.StatusBadRequest, fmt.Sprintf("incorrect query params: %s", err))
		return
	}

	serviceName := r.URL.Query().Get("service")
	if serviceName == "" {
		writeErrorResponseV3(w, http.StatusBadRequest, "service is required")
		return
	}

	operationList, err := query.GetSpanNameList(ctx, cp, serviceName)
	if err != nil {
		writeErrorResponseV3(w, http.StatusInternalServerError, fmt.Sprintf("cannot get operation list: %s", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	WriteGetOperationsResponseV3(w, operationList)
}

// processGetTraceRequestV3 handles the Jaeger /api/v3/traces/<trace_id> API request.
func processGetTraceRequestV3(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cp, err := tracecommon.GetCommonParams(r)
	if err != nil {
		writeErrorResponseV3(w, http.StatusBadRequest, fmt.Sprintf("incorrect query params: %s", err))
		return
	}

	traceID := r.URL.Path[len("/select/jaeger/api/v3/traces/"):]
	rows, err := query.GetTrace(ctx, cp, traceID)
	if err != nil {
		writeErrorResponseV3(w, http.StatusInternalServerError, fmt.Sprintf("cannot get trace: %s", err))
		return
	}

	writeTracesResponseV3(w, rows)
}

// processFindTracesRequestV3 handles the Jaeger /api/v3/traces API request.
func processFindTracesRequestV3(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cp, err := tracecommon.GetCommonParams(r)
	if err != nil {
		writeErrorResponseV3(w, http.StatusBadRequest, fmt.Sprintf("incorrect query params: %s", err))
		return
	}

	param, err := parseJaegerV3TraceQueryParam(r)
	if err != nil {
		writeErrorResponseV3(w, http.StatusBadRequest, fmt.Sprintf("incorrect trace query params: %s", err))
		return
	}

	_, rows, err := query.GetTraceList(ctx, cp, param)
	if err != nil {
		writeErrorResponseV3(w, http.StatusInternalServerError, fmt.Sprintf("get trace list error: %s", err))
		return
	}

	writeTracesResponseV3(w, rows)
}

// writeTracesResponseV3 writes rows as an OTLP/JSON TracesData wrapped into the grpc-gateway
// streaming envelope.
//
// Jaeger answers with 404 and an error body when no trace matches, so the same is done here.
func writeTracesResponseV3(w http.ResponseWriter, rows []*tracecommon.Row) {
	if len(rows) == 0 {
		writeErrorResponseV3(w, http.StatusNotFound, "No traces found")
		return
	}

	resourceSpans, err := tracecommon.RowsToResourceSpans(rows)
	if err != nil {
		writeErrorResponseV3(w, http.StatusInternalServerError, fmt.Sprintf("cannot convert rows to spans: %s", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	WriteTracesResponseV3(w, resourceSpans)
}

// writeErrorResponseV3 writes the error shape returned by the Jaeger grpc-gateway.
func writeErrorResponseV3(w http.ResponseWriter, httpCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	WriteErrorResponseV3(w, httpCode, message)
}

// getQueryArgV3 returns the value of the canonical query arg, falling back to the snake_case
// alias which Jaeger keeps for backwards compatibility.
func getQueryArgV3(r *http.Request, canonical, deprecated string) string {
	q := r.URL.Query()
	if v := q.Get(canonical); v != "" {
		return v
	}
	return q.Get(deprecated)
}

// parseJaegerV3TraceQueryParam parses the /api/v3/traces query args into query.TraceQueryParam.
//
// The time range args are RFC3339 timestamps here, while the v1 API passes microseconds.
func parseJaegerV3TraceQueryParam(r *http.Request) (*query.TraceQueryParam, error) {
	p := &query.TraceQueryParam{
		StartTimeMin: time.Unix(0, 0),
		StartTimeMax: time.Now(),
		Limit:        min(defaultSearchDepth, *tracecommon.TraceMaxTraces),
	}

	p.ServiceName = getQueryArgV3(r, "query.serviceName", "query.service_name")
	if p.ServiceName == "" {
		return nil, fmt.Errorf("query.serviceName is required")
	}
	p.SpanName = getQueryArgV3(r, "query.operationName", "query.operation_name")

	if s := getQueryArgV3(r, "query.startTimeMin", "query.start_time_min"); s != "" {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return nil, fmt.Errorf("cannot parse query.startTimeMin [%s]: %w", s, err)
		}
		p.StartTimeMin = t
	}
	if s := getQueryArgV3(r, "query.startTimeMax", "query.start_time_max"); s != "" {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return nil, fmt.Errorf("cannot parse query.startTimeMax [%s]: %w", s, err)
		}
		p.StartTimeMax = t
	}
	if !p.StartTimeMin.Before(p.StartTimeMax) {
		return nil, fmt.Errorf("query.startTimeMin must be before query.startTimeMax")
	}

	if s := getQueryArgV3(r, "query.durationMin", "query.duration_min"); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("cannot parse query.durationMin [%s]: %w", s, err)
		}
		p.DurationMin = d
	}
	if s := getQueryArgV3(r, "query.durationMax", "query.duration_max"); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("cannot parse query.durationMax [%s]: %w", s, err)
		}
		p.DurationMax = d
	}

	if s := getQueryArgV3(r, "query.searchDepth", "query.search_depth"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("cannot parse query.searchDepth [%s]: %w", s, err)
		}
		if n < 0 || n > *tracecommon.TraceMaxTraces {
			return nil, fmt.Errorf("query.searchDepth %d out of range [0, %d]", n, *tracecommon.TraceMaxTraces)
		}
		p.Limit = n
	}

	if s := r.URL.Query().Get("query.attributes"); s != "" {
		if err := json.Unmarshal([]byte(s), &p.Attributes); err != nil {
			return nil, fmt.Errorf("cannot parse query.attributes [%s]: %w", s, err)
		}
	}
	p.Attributes = toStorageAttributeFilter(p.Attributes)

	return p, nil
}
