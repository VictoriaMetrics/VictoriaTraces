package apptest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/query"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// QueryOpts contains various params used for querying or ingesting data
type QueryOpts struct {
	Timeout       string
	Start         string
	End           string
	Time          string
	Step          string
	ExtraFilters  []string
	ExtraLabels   []string
	MaxLookback   string
	LatencyOffset string
	Format        string
	Limit         string

	// for ingestion
	HTTPHeaders map[string]string
}

func (qos *QueryOpts) asURLValues() url.Values {
	uv := make(url.Values)
	addNonEmpty := func(name string, values ...string) {
		for _, value := range values {
			if len(value) == 0 {
				continue
			}
			uv.Add(name, value)
		}
	}
	addNonEmpty("start", qos.Start)
	addNonEmpty("end", qos.End)
	addNonEmpty("time", qos.Time)
	addNonEmpty("step", qos.Step)
	addNonEmpty("timeout", qos.Timeout)
	addNonEmpty("extra_label", qos.ExtraLabels...)
	addNonEmpty("extra_filters", qos.ExtraFilters...)
	addNonEmpty("max_lookback", qos.MaxLookback)
	addNonEmpty("latency_offset", qos.LatencyOffset)
	addNonEmpty("format", qos.Format)

	return uv
}

// VictoriaTracesWriteQuerier encompasses the methods for writing, flushing and
// querying the trace data.
type VictoriaTracesWriteQuerier interface {
	OTLPTracesWriter
	JaegerQuerier
	LogsQLQuerier

	StorageFlusher
	StorageMerger
}

// JaegerQuerier contains methods available to Jaeger HTTP API for Querying.
type JaegerQuerier interface {
	JaegerAPIServices(t *testing.T, opts QueryOpts) *JaegerAPIServicesResponse
	JaegerAPIOperations(t *testing.T, serviceName string, opts QueryOpts) *JaegerAPIOperationsResponse
	JaegerAPITraces(t *testing.T, params JaegerQueryParam, opts QueryOpts) *JaegerAPITracesResponse
	JaegerAPITrace(t *testing.T, traceID string, opts QueryOpts) *JaegerAPITraceResponse
	JaegerAPIDependencies(t *testing.T, params JaegerDependenciesParam, opts QueryOpts) *JaegerAPIDependenciesResponse

	JaegerAPIV3Services(t *testing.T, opts QueryOpts) *JaegerAPIV3ServicesResponse
	JaegerAPIV3Operations(t *testing.T, serviceName string, opts QueryOpts) *JaegerAPIV3OperationsResponse
	JaegerAPIV3Traces(t *testing.T, params JaegerV3QueryParam, opts QueryOpts) *JaegerAPIV3TracesResponse
	JaegerAPIV3Trace(t *testing.T, traceID string, opts QueryOpts) *JaegerAPIV3TracesResponse
	JaegerAPIV3TraceSummaries(t *testing.T, params JaegerV3QueryParam, opts QueryOpts) *JaegerAPIV3TraceSummariesResponse
}

type LogsQLQuerier interface {
	LogsQLQuery(t *testing.T, LogsQL string, ops QueryOpts) *LogsQLQueryResponse
}

// OTLPTracesWriter contains methods for writing OTLP trace data.
type OTLPTracesWriter interface {
	OTLPHTTPExportTraces(t *testing.T, request *otelpb.ExportTraceServiceRequest, opts QueryOpts)
	OTLPgRPCExportTraces(t *testing.T, request *otelpb.ExportTraceServiceRequest, opts QueryOpts)

	// low level methods only for tests with raw data. avoid using them unless the methods above can't fulfill the requirement.
	OTLPHTTPExportRawTraces(t *testing.T, data []byte, opts QueryOpts)
}

// StorageFlusher defines a method that forces the flushing of data inserted
// into the storage, so it becomes available for searching immediately.
type StorageFlusher interface {
	ForceFlush(t *testing.T)
}

// StorageMerger defines a method that forces the merging of data inserted
// into the storage.
type StorageMerger interface {
	ForceMerge(t *testing.T)
}

// JaegerQueryParam is a helper structure for implementing extra
// helper functions of `query.TraceQueryParam`.
type JaegerQueryParam struct {
	query.TraceQueryParam
}

// asURLValues add non-empty jaeger query params as URL values.
func (jqp *JaegerQueryParam) asURLValues() url.Values {
	uv := make(url.Values)
	addNonEmpty := func(name string, values ...string) {
		for _, value := range values {
			if len(value) == 0 {
				continue
			}
			uv.Add(name, value)
		}
	}

	addNonEmpty("service", jqp.ServiceName)
	addNonEmpty("operation", jqp.SpanName)

	if len(jqp.Attributes) > 0 {
		b, _ := json.Marshal(jqp.Attributes)
		uv.Add("tags", string(b))
	}
	if jqp.DurationMin > 0 {
		uv.Add("minDuration", strconv.FormatInt(jqp.DurationMin.Milliseconds(), 10)+"ms")
	}
	if jqp.DurationMax > 0 {
		uv.Add("maxDuration", strconv.FormatInt(jqp.DurationMax.Milliseconds(), 10)+"ms")
	}
	if jqp.Limit > 0 {
		uv.Add("limit", strconv.Itoa(jqp.Limit))
	}
	if !jqp.StartTimeMin.IsZero() {
		uv.Add("start", strconv.FormatInt(jqp.StartTimeMin.UnixMicro(), 10))
	}
	if !jqp.StartTimeMax.IsZero() {
		uv.Add("end", strconv.FormatInt(jqp.StartTimeMax.UnixMicro(), 10))
	}

	return uv
}

// JaegerResponse contains the common fields shared by all responses of Jaeger query APIs.
type JaegerResponse struct {
	Errors interface{} `json:"errors"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
	Total  int         `json:"total"`
}

// JaegerAPIServicesResponse is an in-memory representation of the
// /select/jaeger/services response.
type JaegerAPIServicesResponse struct {
	Data []string `json:"data"`
	JaegerResponse
}

// JaegerAPIOperationsResponse is an in-memory representation of the
// /select/jaeger/services/<service_name>/operations response.
type JaegerAPIOperationsResponse struct {
	Data []string `json:"data"`
	JaegerResponse
}

// JaegerAPITracesResponse is an in-memory representation of the
// /select/jaeger/traces response.
type JaegerAPITracesResponse struct {
	Data []TracesResponseData `json:"data"`
	JaegerResponse
}

// JaegerAPITraceResponse is an in-memory representation of the
// /select/jaeger/traces/<trace_id> response.
type JaegerAPITraceResponse struct {
	Data []TracesResponseData `json:"data"`
	JaegerResponse
}

// TracesResponseData is the structure of `data` field of the
// /select/jaeger/traces and /select/jaeger/traces/<trace_id> response.
type TracesResponseData struct {
	Processes map[string]Process `json:"processes"`
	Spans     []Span             `json:"spans"`
	TraceID   string             `json:"traceID"`
	Warnings  interface{}        `json:"warnings"`
}

// Process is the structure for Jaeger Process.
type Process struct {
	ServiceName string `json:"serviceName"`
	Tags        []Tag  `json:"tags"`
}

// Tag is the structure for Jaeger tag.
type Tag struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Span is the structure for Jaeger Span.
type Span struct {
	Duration      int         `json:"duration"`
	Logs          []Log       `json:"logs"`
	OperationName string      `json:"operationName"`
	ProcessID     string      `json:"processID"`
	References    []Reference `json:"references"`
	SpanID        string      `json:"spanID"`
	StartTime     int64       `json:"startTime"`
	Tags          []Tag       `json:"tags"`
	TraceID       string      `json:"traceID"`
	Warnings      interface{} `json:"warnings"`
}

// Log is the structure for Jaeger Log.
type Log struct {
	Timestamp int64 `json:"timestamp"`
	Fields    []Tag `json:"fields"`
}

// Reference is the structure for Jaeger Reference.
type Reference struct {
	RefType string `json:"refType"`
	SpanID  string `json:"spanID"`
	TraceID string `json:"traceID"`
}

// NewJaegerAPIServicesResponse is a test helper function that creates a new
// instance of JaegerAPIServicesResponse by unmarshalling a json string.
func NewJaegerAPIServicesResponse(t *testing.T, s string) *JaegerAPIServicesResponse {
	t.Helper()

	res := &JaegerAPIServicesResponse{}
	if err := json.Unmarshal([]byte(s), res); err != nil {
		t.Fatalf("could not unmarshal query response data=\n%s\n: %v", string(s), err)
	}
	return res
}

// NewJaegerAPIOperationsResponse is a test helper function that creates a new
// instance of JaegerAPIOperationsResponse by unmarshalling a json string.
func NewJaegerAPIOperationsResponse(t *testing.T, s string) *JaegerAPIOperationsResponse {
	t.Helper()

	res := &JaegerAPIOperationsResponse{}
	if err := json.Unmarshal([]byte(s), res); err != nil {
		t.Fatalf("could not unmarshal query response data=\n%s\n: %v", string(s), err)
	}
	return res
}

// NewJaegerAPITracesResponse is a test helper function that creates a new
// instance of JaegerAPITracesResponse by unmarshalling a json string.
func NewJaegerAPITracesResponse(t *testing.T, s string) *JaegerAPITracesResponse {
	t.Helper()

	res := &JaegerAPITracesResponse{}
	if err := json.Unmarshal([]byte(s), res); err != nil {
		t.Fatalf("could not unmarshal query response data=\n%s\n: %v", string(s), err)
	}
	return res
}

// NewJaegerAPITraceResponse is a test helper function that creates a new
// instance of JaegerAPITraceResponse by unmarshalling a json string.
func NewJaegerAPITraceResponse(t *testing.T, s string) *JaegerAPITraceResponse {
	t.Helper()

	res := &JaegerAPITraceResponse{}
	if err := json.Unmarshal([]byte(s), res); err != nil {
		t.Fatalf("could not unmarshal query response data=\n%s\n: %v", string(s), err)
	}
	return res
}

// NewJaegerAPIDependenciesResponse is a test helper function that creates a new
// instance of JaegerAPIDependenciesResponse by unmarshalling a json string.
func NewJaegerAPIDependenciesResponse(t *testing.T, s string) *JaegerAPIDependenciesResponse {
	t.Helper()

	res := &JaegerAPIDependenciesResponse{}
	if err := json.Unmarshal([]byte(s), res); err != nil {
		t.Fatalf("could not unmarshal query response data=\n%s\n: %v", string(s), err)
	}
	return res
}

// JaegerDependenciesParam is a helper structure for implementing extra
// helper functions of `query.ServiceGraphQueryParameters`.
type JaegerDependenciesParam struct {
	query.ServiceGraphQueryParameters
}

// asURLValues add non-empty jaeger dependencies params as URL values.
func (jqp *JaegerDependenciesParam) asURLValues() url.Values {
	uv := make(url.Values)
	addNonEmpty := func(name string, values ...string) {
		for _, value := range values {
			if len(value) == 0 {
				continue
			}
			uv.Add(name, value)
		}
	}

	addNonEmpty("endTs", strconv.FormatInt(jqp.EndTs.UnixMilli(), 10))
	addNonEmpty("lookback", strconv.FormatInt(jqp.Lookback.Milliseconds(), 10))

	return uv
}

type JaegerAPIDependenciesResponse struct {
	Data []DependenciesResponseData `json:"data"`
	JaegerResponse
}

type DependenciesResponseData struct {
	Parent    string `json:"parent"`
	Child     string `json:"child"`
	CallCount int    `json:"callCount"`
}

// LogsQLQueryResponse is an in-memory representation of the
// /select/logsql/query response.
type LogsQLQueryResponse struct {
	LogLines []string
}

// NewLogsQLQueryResponse is a test helper function that creates a new
// instance of LogsQLQueryResponse by unmarshalling a json string.
func NewLogsQLQueryResponse(t *testing.T, s string) *LogsQLQueryResponse {
	t.Helper()

	res := &LogsQLQueryResponse{}
	if len(s) == 0 {
		return res
	}
	bs := bytes.NewBufferString(s)
	for {
		logLine, err := bs.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(logLine) > 0 {
					t.Fatalf("BUG: unexpected non-empty line=%q with io.EOF", logLine)
				}
				break
			}
			t.Fatalf("BUG: cannot read logline from buffer: %s", err)
		}
		var lv map[string]any
		if err := json.Unmarshal([]byte(logLine), &lv); err != nil {
			t.Fatalf("cannot parse log line=%q: %s", logLine, err)
		}
		delete(lv, "_stream_id")
		normalizedLine, err := json.Marshal(lv)
		if err != nil {
			t.Fatalf("cannot marshal parsed logline=%q: %s", logLine, err)
		}
		res.LogLines = append(res.LogLines, string(normalizedLine))
	}

	return res
}

// JaegerV3QueryParam is a helper structure for building the query args of the
// Jaeger /api/v3/traces API.
type JaegerV3QueryParam struct {
	ServiceName   string
	OperationName string
	Attributes    map[string]string
	StartTimeMin  time.Time
	StartTimeMax  time.Time
	SearchDepth   int
}

// asURLValues adds non-empty Jaeger v3 query params as URL values.
//
// Unlike the v1 API, the time range is passed as RFC3339 timestamps.
func (jqp *JaegerV3QueryParam) asURLValues() url.Values {
	uv := make(url.Values)
	if len(jqp.ServiceName) > 0 {
		uv.Add("query.serviceName", jqp.ServiceName)
	}
	if len(jqp.OperationName) > 0 {
		uv.Add("query.operationName", jqp.OperationName)
	}
	if len(jqp.Attributes) > 0 {
		b, _ := json.Marshal(jqp.Attributes)
		uv.Add("query.attributes", string(b))
	}
	if !jqp.StartTimeMin.IsZero() {
		uv.Add("query.startTimeMin", jqp.StartTimeMin.Format(time.RFC3339Nano))
	}
	if !jqp.StartTimeMax.IsZero() {
		uv.Add("query.startTimeMax", jqp.StartTimeMax.Format(time.RFC3339Nano))
	}
	if jqp.SearchDepth > 0 {
		uv.Add("query.searchDepth", strconv.Itoa(jqp.SearchDepth))
	}
	return uv
}

// JaegerAPIV3ServicesResponse is an in-memory representation of the
// /select/jaeger/api/v3/services response.
type JaegerAPIV3ServicesResponse struct {
	Services []string `json:"services"`
}

// JaegerAPIV3OperationsResponse is an in-memory representation of the
// /select/jaeger/api/v3/operations response.
type JaegerAPIV3OperationsResponse struct {
	Operations []JaegerV3Operation `json:"operations"`
}

// JaegerV3Operation is a single entry of the /api/v3/operations response.
type JaegerV3Operation struct {
	Name     string `json:"name"`
	SpanKind string `json:"spanKind"`
}

// JaegerAPIV3TracesResponse is an in-memory representation of the
// /select/jaeger/api/v3/traces and /select/jaeger/api/v3/traces/<trace_id> responses.
//
// The types below mirror OTLP/JSON instead of reusing the otelpb structs, so that the test
// verifies the encoding written by the API rather than sharing it.
type JaegerAPIV3TracesResponse struct {
	Result JaegerV3TracesData `json:"result"`
	Error  *JaegerV3Error     `json:"error"`
}

// JaegerV3Error is the error shape returned by the Jaeger v3 API.
type JaegerV3Error struct {
	HTTPCode int    `json:"httpCode"`
	Message  string `json:"message"`
}

// JaegerV3TracesData is the OTLP TracesData message.
type JaegerV3TracesData struct {
	ResourceSpans []JaegerV3ResourceSpans `json:"resourceSpans"`
}

// JaegerV3ResourceSpans is the OTLP ResourceSpans message.
type JaegerV3ResourceSpans struct {
	Resource   JaegerV3Resource     `json:"resource"`
	ScopeSpans []JaegerV3ScopeSpans `json:"scopeSpans"`
}

// JaegerV3Resource is the OTLP Resource message.
type JaegerV3Resource struct {
	Attributes []JaegerV3KeyValue `json:"attributes"`
}

// JaegerV3ScopeSpans is the OTLP ScopeSpans message.
type JaegerV3ScopeSpans struct {
	Scope JaegerV3Scope  `json:"scope"`
	Spans []JaegerV3Span `json:"spans"`
}

// JaegerV3Scope is the OTLP InstrumentationScope message.
type JaegerV3Scope struct {
	Name       string             `json:"name"`
	Version    string             `json:"version"`
	Attributes []JaegerV3KeyValue `json:"attributes"`
}

// JaegerV3Span is the OTLP Span message.
type JaegerV3Span struct {
	TraceID           string             `json:"traceId"`
	SpanID            string             `json:"spanId"`
	TraceState        string             `json:"traceState"`
	ParentSpanID      string             `json:"parentSpanId"`
	Flags             uint32             `json:"flags"`
	Name              string             `json:"name"`
	Kind              int                `json:"kind"`
	StartTimeUnixNano string             `json:"startTimeUnixNano"`
	EndTimeUnixNano   string             `json:"endTimeUnixNano"`
	Attributes        []JaegerV3KeyValue `json:"attributes"`
	Status            JaegerV3Status     `json:"status"`
}

// JaegerV3Status is the OTLP Status message.
type JaegerV3Status struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// JaegerV3KeyValue is the OTLP KeyValue message.
type JaegerV3KeyValue struct {
	Key   string           `json:"key"`
	Value JaegerV3AnyValue `json:"value"`
}

// JaegerV3AnyValue is the OTLP AnyValue message.
//
// Only the string case is covered, since every attribute is stored as a string.
type JaegerV3AnyValue struct {
	StringValue string `json:"stringValue"`
}

// NewJaegerAPIV3ServicesResponse is a test helper function that creates a new
// instance of JaegerAPIV3ServicesResponse by unmarshalling a json string.
func NewJaegerAPIV3ServicesResponse(t *testing.T, s string) *JaegerAPIV3ServicesResponse {
	t.Helper()

	res := &JaegerAPIV3ServicesResponse{}
	if err := json.Unmarshal([]byte(s), res); err != nil {
		t.Fatalf("could not unmarshal query response data=\n%s\n: %v", s, err)
	}
	return res
}

// NewJaegerAPIV3OperationsResponse is a test helper function that creates a new
// instance of JaegerAPIV3OperationsResponse by unmarshalling a json string.
func NewJaegerAPIV3OperationsResponse(t *testing.T, s string) *JaegerAPIV3OperationsResponse {
	t.Helper()

	res := &JaegerAPIV3OperationsResponse{}
	if err := json.Unmarshal([]byte(s), res); err != nil {
		t.Fatalf("could not unmarshal query response data=\n%s\n: %v", s, err)
	}
	return res
}

// NewJaegerAPIV3TracesResponse is a test helper function that creates a new
// instance of JaegerAPIV3TracesResponse by unmarshalling a json string.
func NewJaegerAPIV3TracesResponse(t *testing.T, s string) *JaegerAPIV3TracesResponse {
	t.Helper()

	res := &JaegerAPIV3TracesResponse{}
	if err := json.Unmarshal([]byte(s), res); err != nil {
		t.Fatalf("could not unmarshal query response data=\n%s\n: %v", s, err)
	}
	return res
}

// JaegerAPIV3TraceSummariesResponse is an in-memory representation of the
// /select/jaeger/api/v3/trace-summaries response.
type JaegerAPIV3TraceSummariesResponse struct {
	Summaries []JaegerV3TraceSummary `json:"summaries"`
}

// JaegerV3TraceSummary is a single entry of the /api/v3/trace-summaries response.
type JaegerV3TraceSummary struct {
	TraceID              string                   `json:"traceId"`
	RootServiceName      string                   `json:"rootServiceName"`
	RootOperationName    string                   `json:"rootOperationName"`
	MinStartTimeUnixNano string                   `json:"minStartTimeUnixNano"`
	MaxEndTimeUnixNano   string                   `json:"maxEndTimeUnixNano"`
	SpanCount            int                      `json:"spanCount"`
	ErrorSpanCount       int                      `json:"errorSpanCount"`
	OrphanSpanCount      int                      `json:"orphanSpanCount"`
	Services             []JaegerV3ServiceSummary `json:"services"`
}

// JaegerV3ServiceSummary holds the span counts of a single service within a trace.
type JaegerV3ServiceSummary struct {
	Name           string `json:"name"`
	SpanCount      int    `json:"spanCount"`
	ErrorSpanCount int    `json:"errorSpanCount"`
}

// NewJaegerAPIV3TraceSummariesResponse is a test helper function that creates a new
// instance of JaegerAPIV3TraceSummariesResponse by unmarshalling a json string.
func NewJaegerAPIV3TraceSummariesResponse(t *testing.T, s string) *JaegerAPIV3TraceSummariesResponse {
	t.Helper()

	res := &JaegerAPIV3TraceSummariesResponse{}
	if err := json.Unmarshal([]byte(s), res); err != nil {
		t.Fatalf("could not unmarshal query response data=\n%s\n: %v", s, err)
	}
	return res
}
