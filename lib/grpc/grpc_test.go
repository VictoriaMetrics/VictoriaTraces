package grpc

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/bytesutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/flagutil"
)

// TestGRPCErrorf verifies the gRPC headers in the error response.
func TestGRPCErrorf(t *testing.T) {
	f := func(grpcErrorCode string) {
		t.Helper()

		r := httptest.NewRequest(http.MethodPost, "/opentelemetry.proto.collector.trace.v1.TraceService/Export", nil)
		r.Header.Set("Content-Type", "application/grpc")
		w := httptest.NewRecorder()

		Errorf(w, r, grpcErrorCode, "test error")

		h := w.Header()
		if got := h.Get("content-type"); got != "application/grpc+proto" {
			t.Fatalf("unexpected content-type header; got %q; want %q", got, "application/grpc+proto")
		}
		if got := h.Get("trailer"); got != "grpc-status, grpc-message" {
			t.Fatalf("unexpected trailer header; got %q; want %q", got, "grpc-status, grpc-message")
		}
		if got := h.Get("grpc-status"); got != grpcErrorCode {
			t.Fatalf("unexpected grpc-status header; got %q; want %q", got, grpcErrorCode)
		}
	}

	f(StatusCodeInternal)
}

func newDataFrame(message string) []byte {
	frame := make([]byte, dataFrameHeaderLen, dataFrameHeaderLen+len(message))
	binary.BigEndian.PutUint32(frame[1:], uint32(len(message)))
	return append(frame, message...)
}

func TestReadDataFrame_Success(t *testing.T) {
	f := func(body []byte, maxMessageSize int64, messageExpected string) {
		t.Helper()
		var bb bytesutil.ByteBuffer
		message, err := ReadDataFrame(&bb, bytes.NewReader(body), &flagutil.Bytes{N: maxMessageSize, Name: "maxRequestSize"})
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if string(message) != messageExpected {
			t.Fatalf("unexpected message; got %q; want %q", message, messageExpected)
		}
		if string(bb.B) != string(body) {
			t.Fatalf("unexpected bb.B; got %q; want %q", bb.B, body)
		}
	}

	f(newDataFrame(""), 0, "")
	f(newDataFrame("foo"), 10, "foo")

	// message size equals to the limit
	f(newDataFrame("foobar"), 6, "foobar")
}

func TestReadDataFrame_Failure(t *testing.T) {
	f := func(body []byte, maxMessageSize int64, errExpected string) {
		t.Helper()
		var bb bytesutil.ByteBuffer
		_, err := ReadDataFrame(&bb, bytes.NewReader(body), &flagutil.Bytes{N: maxMessageSize, Name: "maxRequestSize"})
		if err == nil {
			t.Fatalf("expecting non-nil error")
		}
		if !strings.Contains(err.Error(), errExpected) {
			t.Fatalf("unexpected error; got %q; want it to contain %q", err, errExpected)
		}
	}

	// message size exceeds the limit
	f(newDataFrame("foobar"), 5, "too big gRPC message size exceeding -maxRequestSize=5 bytes")

	// the body exceeds the limit, while Message-Length claims a small message
	f(append(newDataFrame("foo"), strings.Repeat("x", 100)...), 10, "too big gRPC message size")

	// invalid DATA frame
	f(nil, 10, "invalid gRPC header length: 0")
	f(append(newDataFrame("foo"), 'x'), 10, "invalid gRPC message length")
	f([]byte{2, 0, 0, 0, 0}, 10, "invalid gRPC compressed flag 2")
}
