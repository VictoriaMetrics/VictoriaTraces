package opentelemetry

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/insertutil"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

type fakeLogRowsStorage struct {
	rowsAdded int
}

func (f *fakeLogRowsStorage) MustAddRows(lr *logstorage.LogRows) {
	f.rowsAdded += lr.RowsCount()
}

func (f *fakeLogRowsStorage) CanWriteData() error {
	return nil
}

func (f *fakeLogRowsStorage) IsLocalStorage() bool {
	return false
}

func buildTestGRPCFrame(payload []byte, compressed bool) []byte {
	frame := make([]byte, 5+len(payload))
	if compressed {
		frame[0] = 1
	}
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	copy(frame[5:], payload)
	return frame
}

func gzipCompressTest(t *testing.T, src []byte) []byte {
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

func sampleOTLPRequestBytes() []byte {
	serviceName := "repro-service"
	req := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{
				Resource: otelpb.Resource{
					Attributes: []*otelpb.KeyValue{
						{Key: "service.name", Value: &otelpb.AnyValue{StringValue: &serviceName}},
					},
				},
				ScopeSpans: []*otelpb.ScopeSpans{
					{
						Spans: []*otelpb.Span{
							{
								Name:    "GET /api/users",
								TraceID: "5b8efff798038103d269b633813fc60c",
								SpanID:  "eee19b7ec3c1b174",
							},
						},
					},
				},
			},
		},
	}
	return req.MarshalProtobuf(nil)
}

func TestOTLPGRPCRequestHandler_CompressionFlag(t *testing.T) {
	reqPB := sampleOTLPRequestBytes()

	t.Run("gzip_header_with_uncompressed_frame", func(t *testing.T) {
		storage := &fakeLogRowsStorage{}
		insertutil.SetLogRowsStorage(storage)
		defer insertutil.SetLogRowsStorage(nil)

		body := buildTestGRPCFrame(reqPB, false)
		r := httptest.NewRequest(http.MethodPost, otlpExportTracesPath, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/grpc")
		r.Header.Set("grpc-encoding", "gzip")
		w := httptest.NewRecorder()

		OTLPGRPCRequestHandler(r, w)

		if got := w.Header().Get("grpc-status"); got != "0" {
			t.Fatalf("unexpected grpc-status: got %q (message=%q), want %q", got, w.Header().Get("grpc-message"), "0")
		}
		if storage.rowsAdded != 1 {
			t.Fatalf("unexpected rowsAdded: got %d, want 1", storage.rowsAdded)
		}
	})

	t.Run("gzip_header_with_compressed_frame", func(t *testing.T) {
		storage := &fakeLogRowsStorage{}
		insertutil.SetLogRowsStorage(storage)
		defer insertutil.SetLogRowsStorage(nil)

		body := buildTestGRPCFrame(gzipCompressTest(t, reqPB), true)
		r := httptest.NewRequest(http.MethodPost, otlpExportTracesPath, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/grpc")
		r.Header.Set("grpc-encoding", "gzip")
		w := httptest.NewRecorder()

		OTLPGRPCRequestHandler(r, w)

		if got := w.Header().Get("grpc-status"); got != "0" {
			t.Fatalf("unexpected grpc-status: got %q (message=%q), want %q", got, w.Header().Get("grpc-message"), "0")
		}
		if storage.rowsAdded != 1 {
			t.Fatalf("unexpected rowsAdded: got %d, want 1", storage.rowsAdded)
		}
	})

	t.Run("compressed_frame_with_identity_encoding", func(t *testing.T) {
		storage := &fakeLogRowsStorage{}
		insertutil.SetLogRowsStorage(storage)
		defer insertutil.SetLogRowsStorage(nil)

		body := buildTestGRPCFrame(gzipCompressTest(t, reqPB), true)
		r := httptest.NewRequest(http.MethodPost, otlpExportTracesPath, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/grpc")
		r.Header.Set("grpc-encoding", "identity")
		w := httptest.NewRecorder()

		OTLPGRPCRequestHandler(r, w)

		if got := w.Header().Get("grpc-status"); got != "13" {
			t.Fatalf("unexpected grpc-status: got %q, want %q", got, "13")
		}
		if msg := w.Header().Get("grpc-message"); !strings.Contains(msg, "grpc-encoding") {
			t.Fatalf("unexpected grpc-message: got %q, want mention of grpc-encoding", msg)
		}
		if storage.rowsAdded != 0 {
			t.Fatalf("unexpected rowsAdded: got %d, want 0", storage.rowsAdded)
		}
	})
}
