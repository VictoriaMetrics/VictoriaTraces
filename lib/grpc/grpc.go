package grpc

import (
	"encoding/binary"
	"fmt"
	"net/http"
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
	if n < 5 {
		return fmt.Errorf("invalid gRPC header length: %d", n)
	}

	grpcHeader := req[:5]
	if isCompress := grpcHeader[0]; isCompress != 0 && isCompress != 1 {
		return fmt.Errorf("invalid gRPC compressed flag %d", isCompress)
	}
	messageLength := binary.BigEndian.Uint32(grpcHeader[1:5])
	if n != 5+int(messageLength) {
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
