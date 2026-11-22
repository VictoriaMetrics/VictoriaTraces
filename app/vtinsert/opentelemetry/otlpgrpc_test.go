package opentelemetry

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/writeconcurrencylimiter"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/insertutil"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/grpc"
)

type stubLogRowsStorage struct {
	canWriteErr error
}

func (s *stubLogRowsStorage) MustAddRows(*logstorage.LogRows) {}

func (s *stubLogRowsStorage) CanWriteData() error {
	return s.canWriteErr
}

func (s *stubLogRowsStorage) IsLocalStorage() bool {
	return true
}

func newGRPCFrame(payload []byte) []byte {
	frame := make([]byte, 5+len(payload))
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	copy(frame[5:], payload)
	return frame
}

func TestOTLPGRPCExportStatusCodes(t *testing.T) {
	t.Run("read-only storage returns UNAVAILABLE", func(t *testing.T) {
		insertutil.SetLogRowsStorage(&stubLogRowsStorage{
			canWriteErr: &httpserver.ErrorWithStatusCode{
				Err:        errors.New("storage is in read-only mode"),
				StatusCode: http.StatusTooManyRequests,
			},
		})
		t.Cleanup(func() { insertutil.SetLogRowsStorage(nil) })

		r := httptest.NewRequest(http.MethodPost, otlpExportTracesPath, bytes.NewReader(newGRPCFrame([]byte{0x0a, 0x00})))
		w := httptest.NewRecorder()
		OTLPGRPCRequestHandler(r, w)

		if got := w.Header().Get("grpc-status"); got != grpc.StatusCodeUnavailable {
			t.Fatalf("grpc-status = %q, want %q", got, grpc.StatusCodeUnavailable)
		}
	})

	t.Run("busy write concurrency limit returns UNAVAILABLE", func(t *testing.T) {
		insertutil.SetLogRowsStorage(&stubLogRowsStorage{})
		t.Cleanup(func() { insertutil.SetLogRowsStorage(nil) })

		oldMaxConcurrentInserts := flag.Lookup("maxConcurrentInserts").Value.String()
		oldMaxQueueDuration := flag.Lookup("insert.maxQueueDuration").Value.String()
		t.Cleanup(func() {
			if err := flag.Set("maxConcurrentInserts", oldMaxConcurrentInserts); err != nil {
				t.Errorf("failed restoring maxConcurrentInserts: %v", err)
			}
			if err := flag.Set("insert.maxQueueDuration", oldMaxQueueDuration); err != nil {
				t.Errorf("failed restoring insert.maxQueueDuration: %v", err)
			}
		})
		if err := flag.Set("maxConcurrentInserts", "1"); err != nil {
			t.Fatal(err)
		}
		if err := flag.Set("insert.maxQueueDuration", "10ms"); err != nil {
			t.Fatal(err)
		}
		if err := writeconcurrencylimiter.IncConcurrency(); err != nil {
			t.Fatalf("IncConcurrency setup failed: %v", err)
		}
		defer writeconcurrencylimiter.DecConcurrency()

		r := httptest.NewRequest(http.MethodPost, otlpExportTracesPath, bytes.NewReader(newGRPCFrame([]byte{0x0a, 0x00})))
		w := httptest.NewRecorder()
		OTLPGRPCRequestHandler(r, w)

		if got := w.Header().Get("grpc-status"); got != grpc.StatusCodeUnavailable {
			t.Fatalf("grpc-status = %q, want %q", got, grpc.StatusCodeUnavailable)
		}
	})

	t.Run("invalid protobuf returns INTERNAL", func(t *testing.T) {
		insertutil.SetLogRowsStorage(&stubLogRowsStorage{})
		t.Cleanup(func() { insertutil.SetLogRowsStorage(nil) })

		r := httptest.NewRequest(http.MethodPost, otlpExportTracesPath, bytes.NewReader(newGRPCFrame([]byte{0xff, 0xff})))
		w := httptest.NewRecorder()
		OTLPGRPCRequestHandler(r, w)

		if got := w.Header().Get("grpc-status"); got != grpc.StatusCodeInternal {
			t.Fatalf("grpc-status = %q, want %q", got, grpc.StatusCodeInternal)
		}
	})
}
