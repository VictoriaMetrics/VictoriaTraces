package opentelemetry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/protoparser/protoparserutil"
	"github.com/VictoriaMetrics/metrics"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/insertutil"
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

	cp, err := insertutil.GetCommonParams(r)
	if err != nil {
		httpserver.Errorf(w, r, "cannot parse common params from request: %s", err)
		return
	}
	// stream fields must contain the service name and span name.
	// by using arguments and headers, users can also add other fields as stream fields
	// for potentially better efficiency.
	cp.StreamFields = append(mandatoryStreamFields, cp.StreamFields...)

	if err = insertutil.CanWriteData(); err != nil {
		httpserver.Errorf(w, r, "%s", err)
		return
	}

	encoding := r.Header.Get("Content-Encoding")
	err = protoparserutil.ReadUncompressedData(r.Body, encoding, maxRequestSize, func(data []byte) error {
		var (
			req         otelpb.ExportTraceServiceRequest
			callbackErr error
		)
		lmp := cp.NewLogMessageProcessor("opentelemetry_traces_otlphttp_protobuf", false)
		if callbackErr = req.UnmarshalProtobuf(data); callbackErr != nil {
			errorsProtobufTotal.Inc()
			return fmt.Errorf("cannot unmarshal request from %d protobuf bytes: %w", len(data), callbackErr)
		}
		callbackErr = pushExportTraceServiceRequest(&req, lmp)
		lmp.MustClose()
		return callbackErr
	})
	if err != nil {
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

	cp, err := insertutil.GetCommonParams(r)
	if err != nil {
		httpserver.Errorf(w, r, "cannot parse common params from request: %s", err)
		return
	}
	// stream fields must contain the service name and span name.
	// by using arguments and headers, users can also add other fields as stream fields
	// for potentially better efficiency.
	cp.StreamFields = append(mandatoryStreamFields, cp.StreamFields...)

	if err = insertutil.CanWriteData(); err != nil {
		httpserver.Errorf(w, r, "%s", err)
		return
	}

	encoding := r.Header.Get("Content-Encoding")
	err = protoparserutil.ReadUncompressedData(r.Body, encoding, maxRequestSize, func(data []byte) error {
		var (
			req         otelpb.ExportTraceServiceRequest
			callbackErr error
		)
		lmp := cp.NewLogMessageProcessor("opentelemetry_traces_otlphttp_json", false)
		if callbackErr = req.UnmarshalJSONCustom(data); callbackErr != nil {
			errorsJSONTotal.Inc()
			return fmt.Errorf("cannot unmarshal request from %d protobuf bytes: %w", len(data), callbackErr)
		}
		callbackErr = pushExportTraceServiceRequest(&req, lmp)
		lmp.MustClose()
		return callbackErr
	})
	if err != nil {
		httpserver.Errorf(w, r, "cannot read OpenTelemetry protocol data: %s", err)
		return
	}
	// update requestJSONDuration only for successfully parsed requests
	// There is no need in updating requestJSONDuration for request errors,
	// since their timings are usually much smaller than the timing for successful request parsing.
	requestJSONDuration.UpdateDuration(startTime)
}
