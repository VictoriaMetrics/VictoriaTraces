package tempo

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/tracecommon"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

func TestSummarySearchTracesResult(t *testing.T) {
	tests := []struct {
		name              string
		rows              []*tracecommon.Row
		wantTraceIDs      []string
		wantStartNano     int64
		wantEndNano       int64
		wantServiceName   string
		wantTraceName     string
		wantErr           bool
	}{
		{
			name: "single span with all fields",
			rows: []*tracecommon.Row{
				{
					Timestamp: 1000,
					Fields: []logstorage.Field{
						{Name: otelpb.TraceIDField, Value: "abc123"},
						{Name: otelpb.ResourceAttrServiceName, Value: "my-service"},
						{Name: otelpb.NameField, Value: "GET /api"},
						{Name: otelpb.ParentSpanIDField, Value: ""},
						{Name: otelpb.StartTimeUnixNanoField, Value: "1000000000000"},
						{Name: otelpb.EndTimeUnixNanoField, Value: "1000032000000"},
					},
				},
			},
			wantTraceIDs:    []string{"abc123"},
			wantStartNano:   1000000000000,
			wantEndNano:     1000032000000,
			wantServiceName: "my-service",
			wantTraceName:   "GET /api",
		},
		{
			name: "span missing start_time_unix_nano should not corrupt startTime",
			rows: []*tracecommon.Row{
				{
					Timestamp: 1000,
					Fields: []logstorage.Field{
						{Name: otelpb.TraceIDField, Value: "abc123"},
						{Name: otelpb.ResourceAttrServiceName, Value: "my-service"},
						{Name: otelpb.NameField, Value: "GET /api"},
						{Name: otelpb.ParentSpanIDField, Value: ""},
						{Name: otelpb.EndTimeUnixNanoField, Value: "1000032000000"},
					},
				},
				{
					Timestamp: 2000,
					Fields: []logstorage.Field{
						{Name: otelpb.TraceIDField, Value: "abc123"},
						{Name: otelpb.StartTimeUnixNanoField, Value: "1000000000000"},
						{Name: otelpb.EndTimeUnixNanoField, Value: "1000020000000"},
						{Name: otelpb.ParentSpanIDField, Value: "abc123span1"},
					},
				},
			},
			wantTraceIDs:    []string{"abc123"},
			wantStartNano:   1000000000000,
			wantEndNano:     1000032000000,
			wantServiceName: "my-service",
			wantTraceName:   "GET /api",
		},
		{
			name: "all spans missing start_time should produce 0 not MaxInt64",
			rows: []*tracecommon.Row{
				{
					Timestamp: 1000,
					Fields: []logstorage.Field{
						{Name: otelpb.TraceIDField, Value: "abc123"},
						{Name: otelpb.ResourceAttrServiceName, Value: "svc"},
						{Name: otelpb.NameField, Value: "op"},
						{Name: otelpb.ParentSpanIDField, Value: ""},
						{Name: otelpb.EndTimeUnixNanoField, Value: "5000000000000"},
					},
				},
			},
			wantTraceIDs:    []string{"abc123"},
			wantStartNano:   0,
			wantEndNano:     5000000000000,
			wantServiceName: "svc",
			wantTraceName:   "op",
		},
		{
			name: "multiple traces",
			rows: []*tracecommon.Row{
				{
					Timestamp: 1000,
					Fields: []logstorage.Field{
						{Name: otelpb.TraceIDField, Value: "trace1"},
						{Name: otelpb.ResourceAttrServiceName, Value: "svc-a"},
						{Name: otelpb.NameField, Value: "op-a"},
						{Name: otelpb.ParentSpanIDField, Value: ""},
						{Name: otelpb.StartTimeUnixNanoField, Value: "100"},
						{Name: otelpb.EndTimeUnixNanoField, Value: "200"},
					},
				},
				{
					Timestamp: 2000,
					Fields: []logstorage.Field{
						{Name: otelpb.TraceIDField, Value: "trace2"},
						{Name: otelpb.ResourceAttrServiceName, Value: "svc-b"},
						{Name: otelpb.NameField, Value: "op-b"},
						{Name: otelpb.ParentSpanIDField, Value: ""},
						{Name: otelpb.StartTimeUnixNanoField, Value: "300"},
						{Name: otelpb.EndTimeUnixNanoField, Value: "400"},
					},
				},
			},
			wantTraceIDs: []string{"trace1", "trace2"},
		},
		{
			name: "missing trace_id returns error",
			rows: []*tracecommon.Row{
				{
					Timestamp: 1000,
					Fields: []logstorage.Field{
						{Name: otelpb.StartTimeUnixNanoField, Value: "100"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "root span identified by empty parent_span_id",
			rows: []*tracecommon.Row{
				{
					Timestamp: 1000,
					Fields: []logstorage.Field{
						{Name: otelpb.TraceIDField, Value: "t1"},
						{Name: otelpb.ParentSpanIDField, Value: "parent1"},
						{Name: otelpb.ResourceAttrServiceName, Value: "child-svc"},
						{Name: otelpb.NameField, Value: "child-op"},
						{Name: otelpb.StartTimeUnixNanoField, Value: "200"},
						{Name: otelpb.EndTimeUnixNanoField, Value: "300"},
					},
				},
				{
					Timestamp: 2000,
					Fields: []logstorage.Field{
						{Name: otelpb.TraceIDField, Value: "t1"},
						{Name: otelpb.ParentSpanIDField, Value: ""},
						{Name: otelpb.ResourceAttrServiceName, Value: "root-svc"},
						{Name: otelpb.NameField, Value: "root-op"},
						{Name: otelpb.StartTimeUnixNanoField, Value: "100"},
						{Name: otelpb.EndTimeUnixNanoField, Value: "350"},
					},
				},
			},
			wantTraceIDs:    []string{"t1"},
			wantStartNano:   100,
			wantEndNano:     350,
			wantServiceName: "root-svc",
			wantTraceName:   "root-op",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := summarySearchTracesResult(context.Background(), tc.rows, 100)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tc.wantTraceIDs) > 0 {
				gotIDs := make(map[string]bool)
				for _, s := range result {
					gotIDs[s.traceID] = true
				}
				for _, wantID := range tc.wantTraceIDs {
					if !gotIDs[wantID] {
						t.Errorf("missing trace ID %q in result", wantID)
					}
				}
			}

			if len(tc.wantTraceIDs) == 1 && len(result) == 1 {
				s := result[0]
				if tc.wantStartNano != 0 || tc.wantEndNano != 0 {
					if s.startTimeUnixNano != tc.wantStartNano {
						t.Errorf("startTimeUnixNano = %d, want %d", s.startTimeUnixNano, tc.wantStartNano)
					}
					if s.endTimeUnixNano != tc.wantEndNano {
						t.Errorf("endTimeUnixNano = %d, want %d", s.endTimeUnixNano, tc.wantEndNano)
					}
				}
				if tc.wantServiceName != "" && s.rootServiceName != tc.wantServiceName {
					t.Errorf("rootServiceName = %q, want %q", s.rootServiceName, tc.wantServiceName)
				}
				if tc.wantTraceName != "" && s.rootTraceName != tc.wantTraceName {
					t.Errorf("rootTraceName = %q, want %q", s.rootTraceName, tc.wantTraceName)
				}
			}

			// Verify no summary has startTimeUnixNano == math.MaxInt64 (sentinel leak)
			for _, s := range result {
				if s.startTimeUnixNano == math.MaxInt64 {
					t.Errorf("trace %q has sentinel startTimeUnixNano (math.MaxInt64), should be 0", s.traceID)
				}
			}
		})
	}
}

func TestSummarySearchDurationMsFitsUint32(t *testing.T) {
	// Reproduce the original bug: a span without start_time_unix_nano
	// caused durationMs to be ~1.78 trillion, overflowing uint32.
	rows := []*tracecommon.Row{
		{
			Timestamp: 1000,
			Fields: []logstorage.Field{
				{Name: otelpb.TraceIDField, Value: "abc"},
				{Name: otelpb.ParentSpanIDField, Value: ""},
				{Name: otelpb.ResourceAttrServiceName, Value: "svc"},
				{Name: otelpb.NameField, Value: "op"},
				// No start_time_unix_nano — this is the bug trigger
				{Name: otelpb.EndTimeUnixNanoField, Value: "1778797098296000000"},
			},
		},
		{
			Timestamp: 2000,
			Fields: []logstorage.Field{
				{Name: otelpb.TraceIDField, Value: "abc"},
				{Name: otelpb.ParentSpanIDField, Value: "span1"},
				{Name: otelpb.StartTimeUnixNanoField, Value: "1778797098262000000"},
				{Name: otelpb.EndTimeUnixNanoField, Value: "1778797098294000000"},
			},
		},
	}

	result, err := summarySearchTracesResult(context.Background(), rows, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(result))
	}

	s := result[0]
	durationMs := (s.endTimeUnixNano - s.startTimeUnixNano) / 1e6

	// durationMs must fit in uint32 (max ~4.2 billion) for Tempo API compat
	if durationMs > math.MaxUint32 {
		t.Errorf("durationMs = %d, exceeds uint32 max (%d); startTimeUnixNano=%d, endTimeUnixNano=%d",
			durationMs, uint32(math.MaxUint32), s.startTimeUnixNano, s.endTimeUnixNano)
	}

	if s.startTimeUnixNano == 0 {
		t.Errorf("startTimeUnixNano = 0, should be 1778797098262000000 (from the span that has it)")
	}
}

func TestSearchResponseJSON(t *testing.T) {
	summaries := []traceSummary{
		{
			traceID:           "abc123",
			rootServiceName:   "my-service",
			rootTraceName:     "GET /api",
			startTimeUnixNano: 1684778327699392724,
			endTimeUnixNano:   1684778327756392724,
		},
	}

	json := SearchResponse(summaries)

	// startTimeUnixNano must be a quoted string per Tempo API spec
	if !strings.Contains(json, `"startTimeUnixNano":"1684778327699392724"`) {
		t.Errorf("startTimeUnixNano should be a quoted string, got: %s", json)
	}

	// durationMs should be a bare integer
	if !strings.Contains(json, `"durationMs":57`) {
		t.Errorf("durationMs should be bare integer 57, got: %s", json)
	}

	// Basic structure checks
	if !strings.Contains(json, `"traceID":"abc123"`) {
		t.Errorf("missing traceID in response: %s", json)
	}
	if !strings.Contains(json, `"rootServiceName":"my-service"`) {
		t.Errorf("missing rootServiceName in response: %s", json)
	}
}
