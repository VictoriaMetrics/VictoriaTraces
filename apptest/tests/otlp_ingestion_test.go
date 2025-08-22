package tests

import (
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/query"
	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

type JaegerIngestionParam struct {
	serviceName       string
	spanName          string
	traceID           string
	spanID            string
	format            string
	expectServiceList *[]string
}

// TestSingleOTLPIngestionJaegerQuery test data ingestion of `/insert/opentelemetry/v1/traces` API
// and queries of various `/select/jaeger/api/*` APIs for vl-single.
func TestSingleOTLPIngestionJaegerQuery(t *testing.T) {
	os.RemoveAll(t.Name())

	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartDefaultVtsingle()
	var expectServiceList []string

	pbParam := JaegerIngestionParam{
		serviceName:       "testPbKeyIngestQueryService",
		spanName:          "testPbKeyIngestQuerySpan",
		traceID:           "123456789",
		spanID:            "987654321",
		format:            "protobuf",
		expectServiceList: &expectServiceList,
	}
	expectServiceList = append(expectServiceList, pbParam.serviceName)
	testOTLPIngestionJaegerQuery(tc, sut, pbParam)

	jsonParam := JaegerIngestionParam{
		serviceName:       "testJsonKeyIngestQueryService",
		spanName:          "testJsonKeyIngestQuerySpan",
		traceID:           "11223344A",
		spanID:            "22334455B",
		format:            "json",
		expectServiceList: &expectServiceList,
	}
	expectServiceList = append(expectServiceList, jsonParam.serviceName)
	testOTLPIngestionJaegerQuery(tc, sut, jsonParam)
}

func testOTLPIngestionJaegerQuery(tc *at.TestCase, sut at.VictoriaTracesWriteQuerier, param JaegerIngestionParam) {
	t := tc.T()

	// prepare test data for ingestion and assertion.
	serviceName := param.serviceName
	spanName := param.spanName
	traceID := param.traceID
	spanID := param.spanID
	testTagValue := "testValue"
	testTag := []*otelpb.KeyValue{
		{
			Key: "testTag",
			Value: &otelpb.AnyValue{
				StringValue: &testTagValue,
			},
		},
	}
	assertTag := []at.Tag{
		{
			Key:   "testTag",
			Type:  "string",
			Value: "testValue",
		},
	}
	spanTime := time.Now()

	req := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{
				Resource: otelpb.Resource{
					Attributes: []*otelpb.KeyValue{
						{
							Key: "service.name",
							Value: &otelpb.AnyValue{
								StringValue: &serviceName,
							},
						},
					},
				},
				ScopeSpans: []*otelpb.ScopeSpans{
					{
						Scope: otelpb.InstrumentationScope{
							Name:                   "testInstrumentation",
							Version:                "1.0",
							Attributes:             testTag,
							DroppedAttributesCount: 999,
						},
						Spans: []*otelpb.Span{
							{
								TraceID:           traceID,
								SpanID:            spanID,
								TraceState:        "trace_state",
								ParentSpanID:      spanID,
								Flags:             1,
								Name:              spanName,
								Kind:              otelpb.SpanKind(1),
								StartTimeUnixNano: uint64(spanTime.UnixNano()),
								EndTimeUnixNano:   uint64(spanTime.UnixNano()),
								Attributes:        testTag,
								Events: []*otelpb.SpanEvent{
									{
										TimeUnixNano: uint64(spanTime.UnixNano()),
										Name:         "test event",
										Attributes:   testTag,
									},
								},
								Links: []*otelpb.SpanLink{
									{
										TraceID:    traceID,
										SpanID:     spanID,
										TraceState: "trace_state",
										Attributes: testTag,
										Flags:      1,
									},
								},
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

	// ingest data via /insert/opentelemetry/v1/traces
	sut.OTLPExportTraces(t, req, at.QueryOpts{Format: param.format})
	sut.ForceFlush(t)

	// check services via /select/jaeger/api/services
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/services response",
		Got: func() any {
			return sut.JaegerAPIServices(t, at.QueryOpts{})
		},
		Want: &at.JaegerAPIServicesResponse{
			Data: *param.expectServiceList,
		},
		CmpOpts: []cmp.Option{
			cmpopts.IgnoreFields(at.JaegerAPIServicesResponse{}, "Errors", "Limit", "Offset", "Total"),
			cmpopts.SortSlices(func(a, b string) bool {
				return a < b
			}),
		},
	})

	// check span name via /select/jaeger/api/services/*/operations
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/services/*/operations response",
		Got: func() any {
			return sut.JaegerAPIOperations(t, serviceName, at.QueryOpts{})
		},
		Want: &at.JaegerAPIOperationsResponse{
			Data: []string{spanName},
		},
		CmpOpts: []cmp.Option{
			cmpopts.IgnoreFields(at.JaegerAPIOperationsResponse{}, "Errors", "Limit", "Offset", "Total"),
		},
	})

	// https://opentelemetry.io/docs/specs/otlp/#json-protobuf-encoding
	var expectTraceID, expectSpanID string
	if param.format == "protobuf" {
		expectTraceID = hex.EncodeToString([]byte(traceID))
		expectSpanID = hex.EncodeToString([]byte(spanID))
	} else if param.format == "json" {
		expectTraceID = strings.ToLower(traceID)
		expectSpanID = strings.ToLower(spanID)
	}

	expectTraceData := []at.TracesResponseData{
		{
			Processes: map[string]at.Process{"p1": {ServiceName: param.serviceName, Tags: []at.Tag{}}},
			Spans: []at.Span{
				{
					Duration: 0,
					TraceID:  expectTraceID,
					SpanID:   expectSpanID,
					Logs: []at.Log{
						{
							Timestamp: spanTime.UnixMicro(),
							Fields: append(assertTag, at.Tag{
								Key:   "event",
								Type:  "string",
								Value: "test event",
							}),
						},
					},
					OperationName: spanName,
					ProcessID:     "p1",
					References: []at.Reference{
						{
							TraceID: expectTraceID,
							SpanID:  expectSpanID,
							RefType: "FOLLOWS_FROM",
						},
					},
					StartTime: spanTime.UnixMicro(),
					Tags: []at.Tag{
						{Key: "span.kind", Type: "string", Value: "internal"},
						{Key: "scope_attr:testTag", Type: "string", Value: "testValue"},
						{Key: "otel.scope.name", Type: "string", Value: "testInstrumentation"},
						{Key: "otel.scope.version", Type: "string", Value: "1.0"},
						{Key: "testTag", Type: "string", Value: "testValue"},
						{Key: "error", Type: "string", Value: "unset"},
						{Key: "otel.status_description", Type: "string", Value: "success"},
						{Key: "w3c.tracestate", Type: "string", Value: "trace_state"},
					},
				},
			},
			TraceID: expectTraceID,
		},
	}

	// check traces data via /select/jaeger/api/traces
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/traces response",
		Got: func() any {
			return sut.JaegerAPITraces(t, at.JaegerQueryParam{
				TraceQueryParam: query.TraceQueryParam{
					ServiceName:  serviceName,
					StartTimeMin: spanTime.Add(-10 * time.Minute),
					StartTimeMax: spanTime.Add(10 * time.Minute),
				},
			}, at.QueryOpts{})
		},
		Want: &at.JaegerAPITracesResponse{
			Data: expectTraceData,
		},
		CmpOpts: []cmp.Option{
			cmpopts.IgnoreFields(at.JaegerAPITracesResponse{}, "Errors", "Limit", "Offset", "Total"),
		},
	})
	// check single trace data via /select/jaeger/api/traces/<trace_id>
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/traces/<trace_id> response",
		Got: func() any {
			return sut.JaegerAPITrace(t, expectTraceID, at.QueryOpts{})
		},
		Want: &at.JaegerAPITraceResponse{
			Data: expectTraceData,
		},
		CmpOpts: []cmp.Option{
			cmpopts.IgnoreFields(at.JaegerAPITraceResponse{}, "Errors", "Limit", "Offset", "Total"),
		},
	})
}
