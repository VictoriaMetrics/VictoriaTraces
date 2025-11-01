package apptest

import (
	"encoding/json"
	"net/url"
	"strconv"
	"testing"

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
}

// OTLPTracesWriter contains methods for writing OTLP trace data.
type OTLPTracesWriter interface {
	OTLPHTTPExportTraces(t *testing.T, request *otelpb.ExportTraceServiceRequest, opts QueryOpts)
	OTLPgRPCExportTraces(t *testing.T, request *otelpb.ExportTraceServiceRequest, opts QueryOpts)
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
