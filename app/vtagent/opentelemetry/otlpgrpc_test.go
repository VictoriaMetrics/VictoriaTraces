package opentelemetry

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

func buildGRPCFrame(payload []byte, compressed bool) []byte {
	frame := make([]byte, 5+len(payload))
	if compressed {
		frame[0] = 1
	}
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	copy(frame[5:], payload)
	return frame
}

func gzipCompress(t *testing.T, src []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(src); err != nil {
		t.Fatalf("gzip write failed: %s", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close failed: %s", err)
	}
	return buf.Bytes()
}

func TestOTLPGRPCRequestHandler_CompressionFlag(t *testing.T) {
	reqPB := (&otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{{}},
	}).MarshalProtobuf(nil)

	t.Run("gzip_header_with_uncompressed_frame", func(t *testing.T) {
		body := buildGRPCFrame(reqPB, false)
		r := httptest.NewRequest(http.MethodPost, otlpExportTracesPath, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/grpc")
		r.Header.Set("grpc-encoding", "gzip")
		w := httptest.NewRecorder()

		OTLPGRPCRequestHandler(w, r)

		if got := w.Header().Get("grpc-status"); got != "0" {
			t.Fatalf("unexpected grpc-status: got %q (message=%q), want %q", got, w.Header().Get("grpc-message"), "0")
		}
	})

	t.Run("gzip_header_with_compressed_frame", func(t *testing.T) {
		body := buildGRPCFrame(gzipCompress(t, reqPB), true)
		r := httptest.NewRequest(http.MethodPost, otlpExportTracesPath, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/grpc")
		r.Header.Set("grpc-encoding", "gzip")
		w := httptest.NewRecorder()

		OTLPGRPCRequestHandler(w, r)

		if got := w.Header().Get("grpc-status"); got != "0" {
			t.Fatalf("unexpected grpc-status: got %q (message=%q), want %q", got, w.Header().Get("grpc-message"), "0")
		}
	})

	t.Run("compressed_frame_without_encoding_header", func(t *testing.T) {
		body := buildGRPCFrame(gzipCompress(t, reqPB), true)
		r := httptest.NewRequest(http.MethodPost, otlpExportTracesPath, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/grpc")
		w := httptest.NewRecorder()

		OTLPGRPCRequestHandler(w, r)

		if got := w.Header().Get("grpc-status"); got != "13" {
			t.Fatalf("unexpected grpc-status: got %q, want %q", got, "13")
		}
		if msg := w.Header().Get("grpc-message"); !strings.Contains(msg, "grpc-encoding") {
			t.Fatalf("unexpected grpc-message: got %q, want mention of grpc-encoding", msg)
		}
	})
}
