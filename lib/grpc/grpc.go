package grpc

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/bytesutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/flagutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/ioutil"
)

// https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
const (
	StatusCodeOk                 = "0"
	StatusCodeCancelled          = "1"
	StatusCodeUnknown            = "2"
	StatusCodeInvalidArgument    = "3"
	StatusCodeDeadlineExceeded   = "4"
	StatusCodeNotFound           = "5"
	StatusCodeAlreadyExist       = "6"
	StatusCodePermissionDenied   = "7"
	StatusCodeResourceExhausted  = "8"
	StatusCodeFailedPrecondition = "9"
	StatusCodeAbort              = "10"
	StatusCodeOutOfRange         = "11"
	StatusCodeUnimplemented      = "12"
	StatusCodeInternal           = "13"
	StatusCodeUnavailable        = "14"
	StatusCodeDataLoss           = "15"
	StatusCodeUnauthenticated    = "16"
)

// dataFrameHeaderLen is the length of `Compressed-Flag` and `Message-Length` in DATA frame.
const dataFrameHeaderLen = 5

// ReadDataFrame reads DATA frame from r into bb and returns the message data without DATA frame header.
//
// The maxMessageSize limits the maximum message data size, which can be read from r.
func ReadDataFrame(bb *bytesutil.ByteBuffer, r io.Reader, maxMessageSize *flagutil.Bytes) ([]byte, error) {
	lr := ioutil.GetLimitedReader(r, dataFrameHeaderLen+maxMessageSize.N+1)
	defer ioutil.PutLimitedReader(lr)

	if _, err := bb.ReadFrom(lr); err != nil {
		return nil, fmt.Errorf("cannot read request body: %w", err)
	}
	if int64(len(bb.B)) > dataFrameHeaderLen+maxMessageSize.N {
		return nil, fmt.Errorf("too big gRPC message size exceeding -%s=%d bytes", maxMessageSize.Name, maxMessageSize.N)
	}
	if err := CheckDataFrame(bb.B); err != nil {
		return nil, err
	}
	return bb.B[dataFrameHeaderLen:], nil
}

// CheckDataFrame verify the `Compressed-Flag` and `Message-Length` in DATA frame.
// See https://grpc.github.io/grpc/core/md_doc__p_r_o_t_o_c_o_l-_h_t_t_p2.html
//
// The DATA frame looks like:
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
func CheckDataFrame(req []byte) error {
	n := len(req)
	if n < dataFrameHeaderLen {
		return fmt.Errorf("invalid gRPC header length: %d", n)
	}

	grpcHeader := req[:dataFrameHeaderLen]
	if isCompress := grpcHeader[0]; isCompress != 0 && isCompress != 1 {
		return fmt.Errorf("invalid gRPC compressed flag %d", isCompress)
	}
	messageLength := binary.BigEndian.Uint32(grpcHeader[1:5])
	if n != dataFrameHeaderLen+int(messageLength) {
		return fmt.Errorf("invalid gRPC message length: %d, actual length: %d", messageLength, n)
	}
	return nil
}

// WriteErrorGrpcResponse write error response in gRPC protocol over HTTP.
func WriteErrorGrpcResponse(w http.ResponseWriter, grpcErrorCode, grpcErrorMessage string) {
	w.Header().Set("content-type", "application/grpc+proto")
	w.Header().Set("trailer", "grpc-status, grpc-message")
	w.Header().Set("grpc-status", grpcErrorCode)
	w.Header().Set("grpc-message", grpcErrorMessage)
}
