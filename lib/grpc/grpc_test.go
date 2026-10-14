package grpc

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
)

func TestWriteErrorGrpcResponse(t *testing.T) {
	f := func(requestPath, remoteAddr, xForwardedFor, grpcErrorCode, grpcErrorMessage, wantLoggedRemoteAddr string) {
		t.Helper()

		var logBuf bytes.Buffer
		logger.SetOutputForTests(&logBuf)
		defer logger.ResetOutputForTest()

		r := httptest.NewRequest(http.MethodPost, requestPath, nil)
		r.Header.Set("Content-Type", "application/grpc")
		if xForwardedFor != "" {
			r.Header.Set("X-Forwarded-For", xForwardedFor)
		}
		r.RemoteAddr = remoteAddr
		w := httptest.NewRecorder()

		Errorf(w, r, grpcErrorCode, grpcErrorMessage)

		// the error must be still returned to the client in gRPC format
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
		if got := h.Get("grpc-message"); got != grpcErrorMessage {
			t.Fatalf("unexpected grpc-message header; got %q; want %q", got, grpcErrorMessage)
		}

		// the error must be logged exactly once with the client address, the request URI and the error message
		logLines := strings.Split(strings.TrimSuffix(logBuf.String(), "\n"), "\n")
		if len(logLines) != 1 {
			t.Fatalf("unexpected number of log lines; got %d; want 1; log:\n%s", len(logLines), logBuf.String())
		}
		logLine := logLines[0]
		for _, s := range []string{
			"\twarn\t",
			// the location in the log must point to the caller of WriteErrorGrpcResponse, e.g. to this file, but not to grpc.go
			"grpc_test.go:",
			"remoteAddr: " + wantLoggedRemoteAddr + ";",
			"requestURI: " + requestPath + ";",
			grpcErrorMessage,
		} {
			if !strings.Contains(logLine, s) {
				t.Fatalf("log line must contain %q; got:\n%s", s, logLine)
			}
		}
		if strings.Contains(logLine, "grpc.go:") {
			t.Fatalf("log line must not point to grpc.go; got:\n%s", logLine)
		}
	}

	const exportPath = "/opentelemetry.proto.collector.trace.v1.TraceService/Export"

	// internal error on the export method
	f(exportPath, "192.0.2.10:5555", "", StatusCodeInternal,
		"cannot read OpenTelemetry protocol data: cannot decode LogsData request from 7 bytes: unknown wireType=7",
		`"192.0.2.10:5555"`)

	// unknown method
	f("/foo/Bar", "192.0.2.10:5555", "", StatusCodeUnimplemented,
		"gRPC method not found: /foo/Bar",
		`"192.0.2.10:5555"`)

	// the client is behind a proxy
	f(exportPath, "192.0.2.10:5555", "10.1.1.1", StatusCodeUnavailable,
		"requests to grpc export are disabled with -insert.disable command-line flag",
		`"192.0.2.10:5555, X-Forwarded-For: 10.1.1.1"`)
}
