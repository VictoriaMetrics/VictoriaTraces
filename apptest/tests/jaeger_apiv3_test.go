package tests

import (
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// TestSingleJaegerAPIV3Query tests the queries of the `/select/jaeger/api/v3/*` APIs for vt-single.
//
// Jaeger UI 2.15 and newer calls these paths instead of the v1 ones, and it expects OTLP/JSON
// instead of the Jaeger JSON model.
// See https://github.com/VictoriaMetrics/VictoriaTraces/issues/141
func TestSingleJaegerAPIV3Query(t *testing.T) {
	os.RemoveAll(t.Name())

	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartDefaultVtsingle()

	testJaegerAPIV3Query(tc, sut)
}

func testJaegerAPIV3Query(tc *at.TestCase, sut at.VictoriaTracesWriteQuerier) {
	t := tc.T()

	serviceName := "testJaegerV3Service"
	spanName := "testJaegerV3Span"
	traceID := "bda5886e99fffef35a847cb2d493fde1"
	spanID := "0123456789abcdef"
	attrValue := "testValue"
	attr := []*otelpb.KeyValue{
		{
			Key:   "testAttr",
			Value: &otelpb.AnyValue{StringValue: &attrValue},
		},
	}
	spanTime := time.Now()

	req := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{
				Resource: otelpb.Resource{
					Attributes: []*otelpb.KeyValue{
						{
							Key:   "service.name",
							Value: &otelpb.AnyValue{StringValue: &serviceName},
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
								SpanID:            spanID,
								Name:              spanName,
								Kind:              otelpb.SpanKind(2),
								StartTimeUnixNano: uint64(spanTime.UnixNano()),
								EndTimeUnixNano:   uint64(spanTime.UnixNano()),
								Attributes:        attr,
								Status: otelpb.Status{
									Message: "success",
									Code:    0,
								},
							},
						},
					},
				},
			},
		},
	}

	sut.OTLPHTTPExportTraces(t, req, at.QueryOpts{})
	sut.ForceFlush(t)
	time.Sleep(2 * time.Second) // index will be created after -insert.traceMaxDuration (2s in integration test)

	// check services via /select/jaeger/api/v3/services
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/v3/services response",
		Got: func() any {
			return sut.JaegerAPIV3Services(t, at.QueryOpts{})
		},
		Want: &at.JaegerAPIV3ServicesResponse{
			Services: []string{serviceName},
		},
	})

	// check operations via /select/jaeger/api/v3/operations
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/v3/operations response",
		Got: func() any {
			return sut.JaegerAPIV3Operations(t, serviceName, at.QueryOpts{})
		},
		Want: &at.JaegerAPIV3OperationsResponse{
			// span kind is not stored per operation, so the default is reported, as Jaeger does.
			Operations: []at.JaegerV3Operation{{Name: spanName, SpanKind: "internal"}},
		},
	})

	wantTraces := &at.JaegerAPIV3TracesResponse{
		Result: at.JaegerV3TracesData{
			ResourceSpans: []at.JaegerV3ResourceSpans{
				{
					Resource: at.JaegerV3Resource{
						Attributes: []at.JaegerV3KeyValue{
							{Key: "service.name", Value: at.JaegerV3AnyValue{StringValue: serviceName}},
						},
					},
					ScopeSpans: []at.JaegerV3ScopeSpans{
						{
							Scope: at.JaegerV3Scope{
								Name:       "testInstrumentation",
								Version:    "1.0",
								Attributes: []at.JaegerV3KeyValue{},
							},
							Spans: []at.JaegerV3Span{
								{
									TraceID: traceID,
									SpanID:  spanID,
									Name:    spanName,
									Kind:    2,
									// OTLP/JSON encodes 64-bit integers as decimal strings.
									StartTimeUnixNano: strconv.FormatInt(spanTime.UnixNano(), 10),
									EndTimeUnixNano:   strconv.FormatInt(spanTime.UnixNano(), 10),
									Attributes: []at.JaegerV3KeyValue{
										{Key: "testAttr", Value: at.JaegerV3AnyValue{StringValue: attrValue}},
									},
									Status: at.JaegerV3Status{Message: "success"},
								},
							},
						},
					},
				},
			},
		},
	}

	// check a single trace via /select/jaeger/api/v3/traces/<trace_id>
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/v3/traces/<trace_id> response",
		Got: func() any {
			return sut.JaegerAPIV3Trace(t, traceID, at.QueryOpts{})
		},
		Want:    wantTraces,
		CmpOpts: []cmp.Option{cmpAttributesSorted()},
	})

	// check the trace search via /select/jaeger/api/v3/traces
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/v3/traces response",
		Got: func() any {
			return sut.JaegerAPIV3Traces(t, at.JaegerV3QueryParam{
				ServiceName:  serviceName,
				StartTimeMin: spanTime.Add(-time.Hour),
				StartTimeMax: spanTime.Add(time.Hour),
			}, at.QueryOpts{})
		},
		Want:    wantTraces,
		CmpOpts: []cmp.Option{cmpAttributesSorted()},
	})

	// a trace which doesn't exist must be reported as Jaeger does, with 404 and an error body.
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/v3/traces/<trace_id> response for a missing trace",
		Got: func() any {
			return sut.JaegerAPIV3Trace(t, "0123456789abcdef0123456789abcdef", at.QueryOpts{})
		},
		Want: &at.JaegerAPIV3TracesResponse{
			Error: &at.JaegerV3Error{HTTPCode: 404, Message: "No traces found"},
		},
	})
}

// cmpAttributesSorted compares attribute lists without regard to their order, since attributes
// are collected from a map when a span is read back from the storage.
func cmpAttributesSorted() cmp.Option {
	return cmp.Transformer("sortAttributes", func(kvs []at.JaegerV3KeyValue) []at.JaegerV3KeyValue {
		sorted := append([]at.JaegerV3KeyValue(nil), kvs...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })
		return sorted
	})
}

// TestSingleJaegerAPIV3TraceSummaries tests the `/select/jaeger/api/v3/trace-summaries` API,
// which Jaeger UI v2.19 and newer uses for the search screen.
func TestSingleJaegerAPIV3TraceSummaries(t *testing.T) {
	os.RemoveAll(t.Name())

	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartDefaultVtsingle()

	testJaegerAPIV3TraceSummaries(tc, sut)
}

func testJaegerAPIV3TraceSummaries(tc *at.TestCase, sut at.VictoriaTracesWriteQuerier) {
	t := tc.T()

	rootService := "testSummaryRootService"
	childService := "testSummaryChildService"
	traceID := "bda5886e99fffef35a847cb2d493fde2"
	rootSpanID := "0123456789abcde0"
	childSpanID := "0123456789abcde1"
	childSpanAttr := "redis"
	spanTime := time.Now()

	// a two-service trace where the child span reports an error, so every summary field
	// carries a value which a single-span trace would leave at zero.
	req := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			newSummaryResourceSpans(rootService, &otelpb.Span{
				TraceID:           traceID,
				SpanID:            rootSpanID,
				Name:              "rootOperation",
				StartTimeUnixNano: uint64(spanTime.UnixNano()),
				EndTimeUnixNano:   uint64(spanTime.Add(time.Second).UnixNano()),
			}),
			newSummaryResourceSpans(childService, &otelpb.Span{
				TraceID:           traceID,
				SpanID:            childSpanID,
				ParentSpanID:      rootSpanID,
				Name:              "childOperation",
				StartTimeUnixNano: uint64(spanTime.Add(100 * time.Millisecond).UnixNano()),
				EndTimeUnixNano:   uint64(spanTime.Add(500 * time.Millisecond).UnixNano()),
				Status:            otelpb.Status{Code: 2, Message: "boom"},
				Attributes: []*otelpb.KeyValue{
					{Key: "db.system", Value: &otelpb.AnyValue{StringValue: &childSpanAttr}},
				},
			}),
		},
	}

	sut.OTLPHTTPExportTraces(t, req, at.QueryOpts{})
	sut.ForceFlush(t)
	time.Sleep(2 * time.Second) // index will be created after -insert.traceMaxDuration (2s in integration test)

	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/v3/trace-summaries response",
		Got: func() any {
			return sut.JaegerAPIV3TraceSummaries(t, at.JaegerV3QueryParam{
				ServiceName:  rootService,
				StartTimeMin: spanTime.Add(-time.Hour),
				StartTimeMax: spanTime.Add(time.Hour),
			}, at.QueryOpts{})
		},
		Want: &at.JaegerAPIV3TraceSummariesResponse{
			Summaries: []at.JaegerV3TraceSummary{
				{
					TraceID:              traceID,
					RootServiceName:      rootService,
					RootOperationName:    "rootOperation",
					MinStartTimeUnixNano: strconv.FormatInt(spanTime.UnixNano(), 10),
					MaxEndTimeUnixNano:   strconv.FormatInt(spanTime.Add(time.Second).UnixNano(), 10),
					SpanCount:            2,
					ErrorSpanCount:       1,
					Services: []at.JaegerV3ServiceSummary{
						{Name: childService, SpanCount: 1, ErrorSpanCount: 1},
						{Name: rootService, SpanCount: 1},
					},
				},
			},
		},
	})

	// an absent service name means any service, so the same trace must still be found.
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/v3/trace-summaries response without a service name",
		Got: func() any {
			res := sut.JaegerAPIV3TraceSummaries(t, at.JaegerV3QueryParam{
				StartTimeMin: spanTime.Add(-time.Hour),
				StartTimeMax: spanTime.Add(time.Hour),
			}, at.QueryOpts{})
			// other tests may share the storage, so only the trace of this test is checked.
			for _, s := range res.Summaries {
				if s.TraceID == traceID {
					return s.SpanCount
				}
			}
			return 0
		},
		Want: 2,
	})

	// the Tags box of Jaeger UI arrives as query.attributes. A filter which matches a span of
	// the trace returns the whole trace, since a summary covers every span of it.
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/v3/trace-summaries response for a matching attribute",
		Got: func() any {
			res := sut.JaegerAPIV3TraceSummaries(t, at.JaegerV3QueryParam{
				Attributes:   map[string]string{"db.system": childSpanAttr},
				StartTimeMin: spanTime.Add(-time.Hour),
				StartTimeMax: spanTime.Add(time.Hour),
			}, at.QueryOpts{})
			for _, s := range res.Summaries {
				if s.TraceID == traceID {
					return s.SpanCount
				}
			}
			return 0
		},
		Want: 2,
	})

	// the duration filters narrow by the span duration. The root span runs for a second,
	// so a range around that keeps the trace.
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/v3/trace-summaries response for a duration range",
		Got: func() any {
			res := sut.JaegerAPIV3TraceSummaries(t, at.JaegerV3QueryParam{
				DurationMin:  500 * time.Millisecond,
				DurationMax:  2 * time.Second,
				StartTimeMin: spanTime.Add(-time.Hour),
				StartTimeMax: spanTime.Add(time.Hour),
			}, at.QueryOpts{})
			for _, s := range res.Summaries {
				if s.TraceID == traceID {
					return s.SpanCount
				}
			}
			return 0
		},
		Want: 2,
	})

	// a floor above the duration of every span must drop the trace.
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/v3/trace-summaries response for a too high minimum duration",
		Got: func() any {
			res := sut.JaegerAPIV3TraceSummaries(t, at.JaegerV3QueryParam{
				DurationMin:  time.Hour,
				StartTimeMin: spanTime.Add(-time.Hour),
				StartTimeMax: spanTime.Add(time.Hour),
			}, at.QueryOpts{})
			return len(res.Summaries)
		},
		Want: 0,
	})

	// a filter which matches nothing must return no summaries at all.
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/v3/trace-summaries response for a non-matching attribute",
		Got: func() any {
			res := sut.JaegerAPIV3TraceSummaries(t, at.JaegerV3QueryParam{
				Attributes:   map[string]string{"db.system": "no-such-value"},
				StartTimeMin: spanTime.Add(-time.Hour),
				StartTimeMax: spanTime.Add(time.Hour),
			}, at.QueryOpts{})
			return len(res.Summaries)
		},
		Want: 0,
	})
}

// newSummaryResourceSpans wraps a span into a ResourceSpans of the given service.
func newSummaryResourceSpans(serviceName string, span *otelpb.Span) *otelpb.ResourceSpans {
	return &otelpb.ResourceSpans{
		Resource: otelpb.Resource{
			Attributes: []*otelpb.KeyValue{
				{Key: "service.name", Value: &otelpb.AnyValue{StringValue: &serviceName}},
			},
		},
		ScopeSpans: []*otelpb.ScopeSpans{
			{Spans: []*otelpb.Span{span}},
		},
	}
}
