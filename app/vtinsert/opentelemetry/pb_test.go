package opentelemetry

import (
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
	"github.com/VictoriaMetrics/easyproto"
)

func ptr(v uint64) *uint64 {
	return &v
}

func marshalSpanRequest(startTime, endTime *uint64) []byte {
	var mp easyproto.MarshalerPool
	m := mp.Get()
	mm := m.MessageMarshaler()

	rs := mm.AppendMessage(1) // ResourceSpans
	ss := rs.AppendMessage(2) // ScopeSpans
	s := ss.AppendMessage(2)  // Span

	traceID, _ := hex.DecodeString("0123456789abcdef0123456789abcdef")
	spanID, _ := hex.DecodeString("0123456789abcdef")
	s.AppendBytes(1, traceID)
	s.AppendBytes(2, spanID)
	s.AppendString(5, "test-span")

	if startTime != nil {
		s.AppendFixed64(7, *startTime)
	}
	if endTime != nil {
		s.AppendFixed64(8, *endTime)
	}

	data := m.Marshal(nil)
	mp.Put(m)
	return data
}

func TestDecodeExportTraceServiceRequest_Timestamp(t *testing.T) {
	baseTime := uint64(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC).UnixNano())
	fifteenMinutesNano := uint64(15 * time.Minute)

	tests := []struct {
		name           string
		startTime      *uint64
		endTime        *uint64
		wantTimestamp  int64
		wantDuration   string
		wantStartField string
		wantEndField   string
	}{
		{
			name:           "Normal span",
			startTime:      ptr(baseTime),
			endTime:        ptr(baseTime + fifteenMinutesNano),
			wantTimestamp:  int64(baseTime + fifteenMinutesNano),
			wantDuration:   strconv.FormatUint(fifteenMinutesNano, 10),
			wantStartField: strconv.FormatUint(baseTime, 10),
			wantEndField:   strconv.FormatUint(baseTime+fifteenMinutesNano, 10),
		},
		{
			name:           "Zero-duration span",
			startTime:      ptr(baseTime),
			endTime:        ptr(baseTime),
			wantTimestamp:  int64(baseTime),
			wantDuration:   "0",
			wantStartField: strconv.FormatUint(baseTime, 10),
			wantEndField:   strconv.FormatUint(baseTime, 10),
		},
		{
			name:           "Missing end (omitted field)",
			startTime:      ptr(baseTime),
			endTime:        nil,
			wantTimestamp:  0,
			wantDuration:   "",
			wantStartField: strconv.FormatUint(baseTime, 10),
			wantEndField:   "",
		},
		{
			name:           "Missing end (explicit zero)",
			startTime:      ptr(baseTime),
			endTime:        ptr(0),
			wantTimestamp:  0,
			wantDuration:   "",
			wantStartField: strconv.FormatUint(baseTime, 10),
			wantEndField:   "",
		},
		{
			name:           "Missing start (omitted field)",
			startTime:      nil,
			endTime:        ptr(baseTime + fifteenMinutesNano),
			wantTimestamp:  int64(baseTime + fifteenMinutesNano),
			wantDuration:   "",
			wantStartField: "",
			wantEndField:   strconv.FormatUint(baseTime+fifteenMinutesNano, 10),
		},
		{
			name:           "Missing start (explicit zero)",
			startTime:      ptr(0),
			endTime:        ptr(baseTime + fifteenMinutesNano),
			wantTimestamp:  int64(baseTime + fifteenMinutesNano),
			wantDuration:   "",
			wantStartField: "",
			wantEndField:   strconv.FormatUint(baseTime+fifteenMinutesNano, 10),
		},
		{
			name:           "Both missing (omitted fields)",
			startTime:      nil,
			endTime:        nil,
			wantTimestamp:  0,
			wantDuration:   "",
			wantStartField: "",
			wantEndField:   "",
		},
		{
			name:           "Both missing (explicit zeros)",
			startTime:      ptr(0),
			endTime:        ptr(0),
			wantTimestamp:  0,
			wantDuration:   "",
			wantStartField: "",
			wantEndField:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := marshalSpanRequest(tt.startTime, tt.endTime)

			var (
				calledCount  int
				gotTimestamp int64
				gotFields    []logstorage.Field
			)

			err := decodeExportTraceServiceRequest(data, func(timestamp int64, fields []logstorage.Field) {
				calledCount++
				gotTimestamp = timestamp
				gotFields = append([]logstorage.Field(nil), fields...)
			})
			if err != nil {
				t.Fatalf("unexpected decode error: %v", err)
			}
			if calledCount != 1 {
				t.Fatalf("expected pushSpans called once, got %d", calledCount)
			}

			if gotTimestamp != tt.wantTimestamp {
				t.Errorf("got timestamp %d, want %d", gotTimestamp, tt.wantTimestamp)
			}

			getField := func(name string) string {
				for _, f := range gotFields {
					if f.Name == name {
						return f.Value
					}
				}
				return ""
			}

			if got := getField(otelpb.DurationField); got != tt.wantDuration {
				t.Errorf("got duration %q, want %q", got, tt.wantDuration)
			}
			if got := getField(otelpb.StartTimeUnixNanoField); got != tt.wantStartField {
				t.Errorf("got start_time_unix_nano %q, want %q", got, tt.wantStartField)
			}
			if got := getField(otelpb.EndTimeUnixNanoField); got != tt.wantEndField {
				t.Errorf("got end_time_unix_nano %q, want %q", got, tt.wantEndField)
			}
		})
	}
}
