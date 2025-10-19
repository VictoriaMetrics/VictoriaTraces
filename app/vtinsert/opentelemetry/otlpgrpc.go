package opentelemetry

import (
	"encoding/binary"
	"fmt"
	"net/http"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/bytesutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/protoparser/protoparserutil"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/insertutil"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

const OLTPExportTracesGrpcPath = "/opentelemetry.proto.collector.trace.v1.TraceService/Export"

// https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
const (
	GrpcOk                 = "0"
	GrpcCancelled          = "1"
	GrpcUnknown            = "2"
	GrpcInvalidArgument    = "3"
	GrpcDeadlineExceeded   = "4"
	GrpcNotFound           = "5"
	GrpcAlreadyExist       = "6"
	GrpcPermissionDenied   = "7"
	GrpcResourceExhausted  = "8"
	GrpcFailedPrecondition = "9"
	GrpcAbort              = "10"
	GrpcOutOfRange         = "11"
	GrpcUnimplemented      = "12"
	GrpcInternal           = "13"
	GrpcUnavailable        = "14"
	GrpcDataLoss           = "15"
	GrpcUnauthenticated    = "16"
)

var (
	compressedBytes     bytesutil.ByteBufferPool
	responseBodyBytes   bytesutil.ByteBufferPool
	responsePrefixBytes bytesutil.ByteBufferPool
	responseBytes       bytesutil.ByteBufferPool
)

func GrpcExportHandler(r *http.Request, w http.ResponseWriter) {
	if r.URL.Path != OLTPExportTracesGrpcPath {
		WriteErrorGrpcResponse(w, GrpcUnimplemented, fmt.Sprintf("grpc method not found: %s", r.URL.Path))
		return
	}
	cp, err := insertutil.GetCommonParams(r)
	if err != nil {
		WriteErrorGrpcResponse(w, GrpcInternal, fmt.Sprintf("cannot parse common params from request: %s", err))
		return
	}
	// stream fields must contain the service name and span name.
	// by using arguments and headers, users can also add other fields as stream fields
	// for potentially better efficiency.
	cp.StreamFields = append(mandatoryStreamFields, cp.StreamFields...)

	if err = insertutil.CanWriteData(); err != nil {
		WriteErrorGrpcResponse(w, GrpcInternal, err.Error())
		return
	}

	bb := compressedBytes.Get()
	defer compressedBytes.Put(bb)

	_, err = bb.ReadFrom(r.Body)
	if err != nil {
		WriteErrorGrpcResponse(w, GrpcInternal, fmt.Sprintf("cannot read request body: %s", err))
		return
	}

	err = checkProtobufRequest(bb.B)
	if err != nil {
		WriteErrorGrpcResponse(w, GrpcInternal, err.Error())
		return
	}
	bb.B = bb.B[5:]

	encoding := r.Header.Get("grpc-encoding")
	err = protoparserutil.ReadUncompressedData(bb.NewReader(), encoding, maxRequestSize, func(data []byte) error {
		var (
			req         otelpb.ExportTraceServiceRequest
			callbackErr error
		)
		lmp := cp.NewLogMessageProcessor("opentelemetry_traces", false)
		if callbackErr = req.UnmarshalProtobuf(data); callbackErr != nil {
			return fmt.Errorf("cannot unmarshal request from %d protobuf bytes: %w", len(data), callbackErr)
		}
		callbackErr = pushExportTraceServiceRequest(&req, lmp)
		lmp.MustClose()
		return callbackErr
	})
	if err != nil {
		WriteErrorGrpcResponse(w, GrpcInternal, fmt.Sprintf("cannot read OpenTelemetry protocol data: %s", err))
		return
	}

	writeExportTracesGrpcResponse(w, 0, "")
	return
}

// +------------+---------------------------------------------+
// |   1 byte   |                 4 bytes                     |
// +------------+---------------------------------------------+
// | Compressed |               Message Length                |
// |   Flag     |                 (uint32)                    |
// +------------+---------------------------------------------+
// |                                                          |
// |                   Message Data                           |
// |                 (variable length)                        |
// |                                                          |
// +----------------------------------------------------------+
// See https://grpc.github.io/grpc/core/md_doc__p_r_o_t_o_c_o_l-_h_t_t_p2.html
func checkProtobufRequest(req []byte) error {
	n := len(req)
	if n < 5 {
		return fmt.Errorf("invalid grpc header length: %d", n)
	}

	grpcHeader := req[:5]
	if isCompress := grpcHeader[0]; isCompress != 0 && isCompress != 1 {
		return fmt.Errorf("grpc compression not supporte")
	}
	messageLength := binary.BigEndian.Uint32(grpcHeader[1:5])
	if n != 5+int(messageLength) {
		return fmt.Errorf("invalid message length: %d", messageLength)
	}
	return nil
}

func WriteErrorGrpcResponse(w http.ResponseWriter, grpcErrorCode, grpcErrorMessage string) {
	w.Header().Set("Content-Type", "application/grpc+proto")
	w.Header().Set("Trailer", "grpc-status, grpc-message")
	w.Header().Set("Grpc-Status", grpcErrorCode)
	w.Header().Set("Grpc-Message", grpcErrorMessage)
}

// https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md#message-encoding
func writeExportTracesGrpcResponse(w http.ResponseWriter, rejectedSpans int64, errorMessage string) {
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
	rb.Write(rpb.B)
	rb.Write(b)
	w.Header().Set("Content-Type", "application/grpc+proto")
	w.Header().Set("Trailer", "grpc-status, grpc-message")
	w.Header().Set("Grpc-Status", GrpcOk)

	writtenLen, err := w.Write(rb.B) // this will write both header and body since w.WriteHeader is not called.
	if err != nil {
		logger.Errorf("error writing response body: %s", err)
		return
	}
	if writtenLen != rb.Len() {
		logger.Errorf("unexpected write of %d bytes in replying OLTP export grpc request, expected:%d", writtenLen, rb.Len())
		return
	}
}
