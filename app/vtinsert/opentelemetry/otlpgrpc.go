package opentelemetry

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/bytesutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/protoparser/protoparserutil"
	"github.com/VictoriaMetrics/metrics"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/insertutil"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/grpc"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

const otlpExportTracesPath = "/opentelemetry.proto.collector.trace.v1.TraceService/Export"

var (
	compressedBytes     bytesutil.ByteBufferPool
	responseBodyBytes   bytesutil.ByteBufferPool
	responsePrefixBytes bytesutil.ByteBufferPool
	responseBytes       bytesutil.ByteBufferPool
)

var (
	requestsGRPCTotal = metrics.NewCounter(`vt_http_requests_total{path="/opentelemetry.proto.collector.trace.v1.TraceService/Export",format="protobuf"}`)
	errorsGRPCTotal   = metrics.NewCounter(`vt_http_errors_total{path="/opentelemetry.proto.collector.trace.v1.TraceService/Export",format="protobuf"}`)

	requestGRPCDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/opentelemetry.proto.collector.trace.v1.TraceService/Export",format="protobuf"}`)
)

// OTLPGRPCRequestHandler is the router of gRPC requests.
func OTLPGRPCRequestHandler(r *http.Request, w http.ResponseWriter) bool {
	switch r.URL.Path {
	case otlpExportTracesPath:
		otlpExportTracesHandler(r, w)
	default:
		grpc.WriteErrorGrpcResponse(w, grpc.StatusCodeUnimplemented, fmt.Sprintf("gRPC method not found: %s", r.URL.Path))
	}
	return true
}

// otlpExportTracesHandler handles OTLP export traces requests.
func otlpExportTracesHandler(r *http.Request, w http.ResponseWriter) {
	startTime := time.Now()
	requestsGRPCTotal.Inc()

	if err := insertutil.CanWriteData(); err != nil {
		grpc.WriteErrorGrpcResponse(w, grpc.StatusCodeInternal, err.Error())
		return
	}

	// prepare ingestion
	cp, err := insertutil.GetCommonParams(r)
	if err != nil {
		grpc.WriteErrorGrpcResponse(w, grpc.StatusCodeInternal, fmt.Sprintf("cannot parse common params from request: %s", err))
		return
	}

	// read, check and extract the real message from request body.
	bb := compressedBytes.Get()
	defer compressedBytes.Put(bb)

	_, err = bb.ReadFrom(r.Body)
	if err != nil {
		errorsGRPCTotal.Inc()
		grpc.WriteErrorGrpcResponse(w, grpc.StatusCodeInternal, fmt.Sprintf("cannot read request body: %s", err))
		return
	}

	err = grpc.CheckDataFrame(bb.B)
	if err != nil {
		errorsGRPCTotal.Inc()
		grpc.WriteErrorGrpcResponse(w, grpc.StatusCodeInternal, err.Error())
		return
	}

	bb.B = bb.B[5:]

	// stream fields must contain the service name and span name.
	// by using arguments and headers, users can also add other fields as stream fields
	// for potentially better efficiency.
	cp.StreamFields = append(mandatoryStreamFields, cp.StreamFields...)

	encoding := r.Header.Get("grpc-encoding")
	err = protoparserutil.ReadUncompressedData(bb.NewReader(), encoding, maxRequestSize, func(data []byte) error {
		var (
			req         otelpb.ExportTraceServiceRequest
			callbackErr error
		)
		lmp := cp.NewLogMessageProcessor("opentelemetry_traces_otlpgrpc", false)
		if callbackErr = req.UnmarshalProtobuf(data); callbackErr != nil {
			return fmt.Errorf("cannot unmarshal request from %d protobuf bytes: %w", len(data), callbackErr)
		}
		callbackErr = pushExportTraceServiceRequest(&req, lmp)
		lmp.MustClose()
		return callbackErr
	})
	if err != nil {
		errorsGRPCTotal.Inc()
		grpc.WriteErrorGrpcResponse(w, grpc.StatusCodeInternal, fmt.Sprintf("cannot read OpenTelemetry protocol data: %s", err))
		return
	}

	writeExportTraceServiceResponse(w, 0, "")

	// update requestGRPCDuration only for successfully parsed requests
	// There is no need in updating requestGRPCDuration for request errors,
	// since their timings are usually much smaller than the timing for successful request parsing.
	requestGRPCDuration.UpdateDuration(startTime)
}

// writeExportTraceServiceResponse write response in gRPC protocol over HTTP.
func writeExportTraceServiceResponse(w http.ResponseWriter, rejectedSpans int64, errorMessage string) {
	rbb := responseBodyBytes.Get()
	defer responseBodyBytes.Put(rbb)

	b := rbb.B

	// The server MUST leave the partial_success field unset in case of a successful response.
	// https://opentelemetry.io/docs/specs/otlp/#full-success
	resp := &otelpb.ExportTraceServiceResponse{}
	if rejectedSpans != 0 || errorMessage == "" {
		resp.ExportTracePartialSuccess = &otelpb.ExportTracePartialSuccess{
			RejectedSpans: rejectedSpans,
			ErrorMessage:  errorMessage,
		}
	}
	b = resp.MarshalProtobuf(b)

	rb := responseBytes.Get()
	defer responseBytes.Put(rb)

	rpb := responsePrefixBytes.Get()
	defer responsePrefixBytes.Put(rpb)

	// 5 bytes prefix: 1 byte compress flag + 4 bytes body length
	rpb.B = bytesutil.ResizeNoCopyNoOverallocate(rpb.B, 5)
	binary.BigEndian.PutUint32(rpb.B[1:5], uint32(len(b)))

	// append prefix(5 bytes) and body to response bytes.
	_, _ = rb.Write(rpb.B)
	_, _ = rb.Write(b)

	w.Header().Set("content-type", "application/grpc+proto")
	w.Header().Set("trailer", "grpc-status, grpc-message")
	w.Header().Set("grpc-status", grpc.StatusCodeOk)

	writtenLen, err := w.Write(rb.B) // this will write both header and body since w.WriteHeader is not called.
	if err != nil {
		logger.Errorf("error writing response body: %s", err)
		return
	}
	if writtenLen != rb.Len() {
		logger.Errorf("unexpected write of %d bytes in replying OLTP export gRPC request, expected:%d", writtenLen, rb.Len())
		return
	}
}
