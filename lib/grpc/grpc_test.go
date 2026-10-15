package grpc

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestWriteErrorGrpcResponse to verify the gRPC headers in error response.
func TestWriteErrorGrpcResponse(t *testing.T) {
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
