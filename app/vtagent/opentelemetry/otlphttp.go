package opentelemetry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/bytesutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/protoparser/protoparserutil"
	"github.com/VictoriaMetrics/metrics"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

const (
	contentTypeProtobuf = "application/x-protobuf"
	contentTypeJSON     = "application/json"
)

var (
	requestsProtobufTotal = metrics.NewCounter(`vt_http_requests_total{path="/insert/opentelemetry/v1/traces",format="protobuf"}`)
	errorsProtobufTotal   = metrics.NewCounter(`vt_http_errors_total{path="/insert/opentelemetry/v1/traces",format="protobuf"}`)
	requestsJSONTotal     = metrics.NewCounter(`vt_http_requests_total{path="/insert/opentelemetry/v1/traces",format="json"}`)
	errorsJSONTotal       = metrics.NewCounter(`vt_http_errors_total{path="/insert/opentelemetry/v1/traces",format="json"}`)

	requestProtobufDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/insert/opentelemetry/v1/traces",format="protobuf"}`)
	requestJSONDuration     = metrics.NewSummary(`vt_http_request_duration_seconds{path="/insert/opentelemetry/v1/traces",format="json"}`)
)

var protobufBufPool bytesutil.ByteBufferPool

// RequestHandler processes Opentelemetry insert requests
func RequestHandler(path string, w http.ResponseWriter, r *http.Request) bool {
	switch path {
	// use the same path as opentelemetry collector
	// https://opentelemetry.io/docs/specs/otlp/#otlphttp-request
	case "/insert/opentelemetry/v1/traces":
		return handleTracesRequest(r, w)
	default:
		return false
	}
}

func handleTracesRequest(r *http.Request, w http.ResponseWriter) bool {
	switch contentType := r.Header.Get("Content-Type"); contentType {
	case contentTypeProtobuf:
		handleProtobufRequest(r, w)
	case contentTypeJSON:
		handleJSONRequest(r, w)
	default:
		httpserver.Errorf(w, r, "Content-Type %s isn't supported for opentelemetry format. Use protobuf or JSON encoding", contentType)
		return false
	}
	return true
}

func handleProtobufRequest(r *http.Request, w http.ResponseWriter) {
	startTime := time.Now()
	requestsProtobufTotal.Inc()

	encoding := r.Header.Get("Content-Encoding")
	err := protoparserutil.ReadUncompressedData(r.Body, encoding, maxRequestSize, pushExportTraceServiceRequest)
	if err != nil {
		errorsProtobufTotal.Inc()
		httpserver.Errorf(w, r, "cannot read OpenTelemetry protocol data: %s", err)
		return
	}
	// update requestProtobufDuration only for successfully parsed requests
	// There is no need in updating requestProtobufDuration for request errors,
	// since their timings are usually much smaller than the timing for successful request parsing.
	requestProtobufDuration.UpdateDuration(startTime)
}

func handleJSONRequest(r *http.Request, w http.ResponseWriter) {
	startTime := time.Now()
	requestsJSONTotal.Inc()

	encoding := r.Header.Get("Content-Encoding")
	err := protoparserutil.ReadUncompressedData(r.Body, encoding, maxRequestSize, func(data []byte) error {
		// JSON-encoded requests cannot be concatenated, so convert them to protobuf before buffering.
		var req otelpb.ExportTraceServiceRequest
		if err := req.UnmarshalJSONCustom(data); err != nil {
			return fmt.Errorf("cannot unmarshal request from %d json bytes: %w", len(data), err)
		}

		bb := protobufBufPool.Get()
		defer protobufBufPool.Put(bb)
		bb.B = req.MarshalProtobuf(bb.B[:0])

		return pushExportTraceServiceRequest(bb.B)
	})
	if err != nil {
		errorsJSONTotal.Inc()
		httpserver.Errorf(w, r, "cannot read OpenTelemetry protocol data: %s", err)
		return
	}
	// update requestJSONDuration only for successfully parsed requests
	// There is no need in updating requestJSONDuration for request errors,
	// since their timings are usually much smaller than the timing for successful request parsing.
	requestJSONDuration.UpdateDuration(startTime)
}
