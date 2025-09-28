package tests

import (
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/query"
	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// TestSingleServiceGraphGenerationJaegerQuery test service graph data generation
// and query of `/select/jaeger/api/dependencies` API for vt-single.
func TestSingleServiceGraphGenerationJaegerQuery(t *testing.T) {
	os.RemoveAll(t.Name())

	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartVtsingle("vtsingle", []string{
		"-storageDataPath=" + tc.Dir() + "/vtsingle",
		"-retentionPeriod=100y",
		"-servicegraph.enable=true",
		"-servicegraph.taskInterval=1s",
	})

	testServiceGraphGenerationJaegerQuery(tc, sut)
}

func testServiceGraphGenerationJaegerQuery(tc *at.TestCase, sut at.VictoriaTracesWriteQuerier) {
	t := tc.T()

	parentServiceName, childServiceName := prepareTraceParentAndChildSpanData(tc, sut)

	// wait for service graph data to be generated
	tc.Assert(&at.AssertOptions{
		Msg: "service graph data not generated",
		Got: func() any {
			return getServiceGraphRowsInsertedTotal(t, sut) >= 1
		},
		Want:    true,
		Retries: 10,
		Period:  time.Second,
	})
	sut.ForceFlush(t)

	// verify the service graph relations via /select/jaeger/api/dependencies
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/dependencies response",
		Got: func() any {
			return sut.JaegerAPIDependencies(t, at.JaegerDependenciesParam{
				ServiceGraphQueryParameters: query.ServiceGraphQueryParameters{
					EndTs:    time.Now(),
					Lookback: time.Minute,
				},
			}, at.QueryOpts{})
		},
		Want: &at.JaegerAPIDependenciesResponse{
			Data: []at.DependenciesResponseData{
				{
					Parent:    parentServiceName,
					Child:     childServiceName,
					CallCount: 1,
				},
			},
		},
		CmpOpts: []cmp.Option{
			cmpopts.IgnoreFields(at.JaegerAPIDependenciesResponse{}, "Errors", "Limit", "Offset", "Total"),
			cmpopts.IgnoreFields(at.DependenciesResponseData{}, "CallCount"),
		},
	})
}

func prepareTraceParentAndChildSpanData(tc *at.TestCase, sut at.VictoriaTracesWriteQuerier) (string, string) {
	t := tc.T()

	// important data required by verification.
	parentServiceName := "testServiceGraphServiceNameParent"
	childServiceName := "testServiceGraphServiceNameChild"

	// prepare test data for ingestion and assertion.
	parentSpanID := "987654321"
	childSpanID := "9876543210"

	spanName := "testKeyIngestQuerySpan"
	traceID := "123456789"
	testTagValue := "testValue"
	testTag := []*otelpb.KeyValue{
		{
			Key: "testTag",
			Value: &otelpb.AnyValue{
				StringValue: &testTagValue,
			},
		},
	}
	spanTime := time.Now()

	parentSpanReq := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{
				Resource: otelpb.Resource{
					Attributes: []*otelpb.KeyValue{
						{
							Key: "service.name",
							Value: &otelpb.AnyValue{
								StringValue: &parentServiceName,
							},
						},
					},
				},
				ScopeSpans: []*otelpb.ScopeSpans{
					{
						Scope: otelpb.InstrumentationScope{
							Name:    "testInstrumentation",
							Version: "1.0",
						},
						Spans: []*otelpb.Span{
							{
								TraceID:           traceID,
								SpanID:            parentSpanID,
								TraceState:        "trace_state",
								ParentSpanID:      "", // root span
								Flags:             1,
								Name:              spanName,
								Kind:              otelpb.SpanKind(3), // parent span must be 3 or 4, 3 means client
								StartTimeUnixNano: uint64(spanTime.UnixNano()),
								EndTimeUnixNano:   uint64(spanTime.UnixNano()),
								Attributes:        testTag,
								Events:            []*otelpb.SpanEvent{},
								Links:             []*otelpb.SpanLink{},
								Status:            otelpb.Status{},
							},
						},
					},
				},
			},
		},
	}

	childSpanReq := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{
				Resource: otelpb.Resource{
					Attributes: []*otelpb.KeyValue{
						{
							Key: "service.name",
							Value: &otelpb.AnyValue{
								StringValue: &childServiceName,
							},
						},
					},
				},
				ScopeSpans: []*otelpb.ScopeSpans{
					{
						Scope: otelpb.InstrumentationScope{
							Name:    "testInstrumentation",
							Version: "1.0",
						},
						Spans: []*otelpb.Span{
							{
								TraceID:           traceID,
								SpanID:            childSpanID,
								TraceState:        "trace_state",
								ParentSpanID:      parentSpanID,
								Flags:             1,
								Name:              spanName,
								Kind:              otelpb.SpanKind(2), // child span must be 2 or 5, 2 means server
								StartTimeUnixNano: uint64(spanTime.UnixNano()),
								EndTimeUnixNano:   uint64(spanTime.UnixNano()),
								Attributes:        testTag,
								Events:            []*otelpb.SpanEvent{},
								Links:             []*otelpb.SpanLink{},
								Status:            otelpb.Status{},
							},
						},
					},
				},
			},
		},
	}

	// ingest data via /insert/opentelemetry/v1/traces
	sut.OTLPExportTraces(t, parentSpanReq, at.QueryOpts{})
	sut.OTLPExportTraces(t, childSpanReq, at.QueryOpts{})
	return parentServiceName, childServiceName
}

func getServiceGraphRowsInsertedTotal(t *testing.T, sut at.VictoriaTracesWriteQuerier) int {
	t.Helper()

	selector := `vt_rows_ingested_total{type="internalinsert_servicegraph"}`
	switch tt := sut.(type) {
	case *at.Vtsingle:
		return tt.GetIntMetric(t, selector)
	default:
		t.Fatalf("unexpected type: got %T, want *Vtsingle", sut)
	}
	return 0
}
