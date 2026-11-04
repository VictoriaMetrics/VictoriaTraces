package tests

import (
	"encoding/hex"
	"fmt"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/fs"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/query"
	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"testing"
	"time"
)

func TestVtagentRemoteWriteSingleTenant(t *testing.T) {
	fs.MustRemoveDir(t.Name())
	tc := at.NewTestCase(t)
	defer tc.Stop()

	// test data ingestion into
	sut1 := tc.MustStartVtsingle("vtsingle-1", []string{
		"-storageDataPath=" + tc.Dir() + "/vtsingle-1",
		"-retentionPeriod=2y",
	})
	sut2 := tc.MustStartVtsingle("vtsingle-2", []string{
		"-storageDataPath=" + tc.Dir() + "/vtsingle-2",
		"-retentionPeriod=2y",
	})

	remoteWriteURLs := []string{
		fmt.Sprintf("http://%s/insert/opentelemetry/v1/traces", sut1.HTTPAddr()),
		fmt.Sprintf("http://%s/insert/opentelemetry/v1/traces", sut2.HTTPAddr()),
	}

	// use a single queue and a flush interval long enough for vtagent to batch all the incoming requests into a single block.
	vtagent := tc.MustStartVtagent("vtagent-1", remoteWriteURLs, []string{
		"-remoteWrite.queues=1",
		"-remoteWrite.flushInterval=2s",
	})
	testOTLPIngestion(tc, vtagent, []at.JaegerQuerier{sut1, sut2})
}

func testOTLPIngestion(tc *at.TestCase, vtagent *at.Vtagent, vtsingles []at.JaegerQuerier) {
	t := tc.T()

	const requestsCount = 5
	reqs, qFunc := getVtagentReqsAndVtsingleQueryFunc(tc, vtsingles, requestsCount)

	// ingest data via /insert/opentelemetry/v1/traces
	for _, req := range reqs {
		vtagent.OTLPHTTPExportTraces(t, req, at.QueryOpts{})
	}
	// data will be flushed to remote storages after -remoteWrite.flushInterval (2s),
	// and index will be created after -insert.indexFlushInterval (2s in integration test).
	time.Sleep(5 * time.Second)

	// verify that all the requests are batched into a single block per remote storage
	remoteStoragesCount := len(vtsingles)
	if n := vtagent.GetIntMetric(t, "vtagent_remotewrite_block_size_spans_count"); n != remoteStoragesCount {
		t.Fatalf("unexpected number of blocks sent to remote storages; got %d; want %d", n, remoteStoragesCount)
	}
	if n := vtagent.GetIntMetric(t, "vtagent_remotewrite_block_size_spans_sum"); n != requestsCount*remoteStoragesCount {
		t.Fatalf("unexpected number of spans sent to remote storages; got %d; want %d", n, requestsCount*remoteStoragesCount)
	}

	// verify
	qFunc()
}

// getVtagentReqsAndVtsingleQueryFunc creates test data, returns it as requestsCount `*otelpb.ExportTraceServiceRequest`
// with a single trace per request, and a func for verifying the data at queriers.
func getVtagentReqsAndVtsingleQueryFunc(tc *at.TestCase, queriers []at.JaegerQuerier, requestsCount int) ([]*otelpb.ExportTraceServiceRequest, func()) {
	t := tc.T()

	// prepare test data
	serviceName := "testKeyIngestQueryService"
	spanName := "testKeyIngestQuerySpan"
	spanID := "987654321"
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

	newRequest := func(traceID string) *otelpb.ExportTraceServiceRequest {
		return &otelpb.ExportTraceServiceRequest{
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
	}

	newTraceData := func(traceID string) at.TracesResponseData {
		return at.TracesResponseData{
			Processes: map[string]at.Process{"p1": {ServiceName: "testKeyIngestQueryService", Tags: []at.Tag{}}},
			Spans: []at.Span{
				{
					Duration: 0,
					TraceID:  traceID,
					SpanID:   hex.EncodeToString([]byte(spanID)),
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
							TraceID: traceID,
							SpanID:  hex.EncodeToString([]byte(spanID)),
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
			TraceID: traceID,
		}
	}

	var reqs []*otelpb.ExportTraceServiceRequest
	var traceIDs []string
	var expectTraceData []at.TracesResponseData
	for i := range requestsCount {
		traceID := fmt.Sprintf("bda5886e99fffef35a847cb2d493fd%02x", i)
		traceIDs = append(traceIDs, traceID)
		reqs = append(reqs, newRequest(traceID))
		expectTraceData = append(expectTraceData, newTraceData(traceID))
	}
	sortTraces := cmpopts.SortSlices(func(a, b at.TracesResponseData) bool {
		return a.TraceID < b.TraceID
	})

	assertFunc := func() {
		for _, querier := range queriers {
			// check services via /select/jaeger/api/services
			tc.Assert(&at.AssertOptions{
				Msg: "unexpected /select/jaeger/api/services response",
				Got: func() any {
					return querier.JaegerAPIServices(t, at.QueryOpts{})
				},
				Want: &at.JaegerAPIServicesResponse{
					Data: []string{serviceName},
				},
				CmpOpts: []cmp.Option{
					cmpopts.IgnoreFields(at.JaegerAPIServicesResponse{}, "Errors", "Limit", "Offset", "Total"),
				},
			})

			// check span name via /select/jaeger/api/services/*/operations
			tc.Assert(&at.AssertOptions{
				Msg: "unexpected /select/jaeger/api/services/*/operations response",
				Got: func() any {
					return querier.JaegerAPIOperations(t, serviceName, at.QueryOpts{})
				},
				Want: &at.JaegerAPIOperationsResponse{
					Data: []string{spanName},
				},
				CmpOpts: []cmp.Option{
					cmpopts.IgnoreFields(at.JaegerAPIOperationsResponse{}, "Errors", "Limit", "Offset", "Total"),
				},
			})

			// check traces data via /select/jaeger/api/traces
			tc.Assert(&at.AssertOptions{
				Msg: "unexpected /select/jaeger/api/traces response",
				Got: func() any {
					return querier.JaegerAPITraces(t, at.JaegerQueryParam{
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
					sortTraces,
				},
			})

			// check single trace data via /select/jaeger/api/traces/<trace_id>
			for i, traceID := range traceIDs {
				tc.Assert(&at.AssertOptions{
					Msg: "unexpected /select/jaeger/api/traces/<trace_id> response",
					Got: func() any {
						return querier.JaegerAPITrace(t, traceID, at.QueryOpts{})
					},
					Want: &at.JaegerAPITraceResponse{
						Data: expectTraceData[i : i+1],
					},
					CmpOpts: []cmp.Option{
						cmpopts.IgnoreFields(at.JaegerAPITraceResponse{}, "Errors", "Limit", "Offset", "Total"),
					},
				})
			}

			// check traces data via /select/jaeger/api/traces with regex tag filter: NOT match, empty result
			tc.Assert(&at.AssertOptions{
				Msg: "unexpected /select/jaeger/api/traces response",
				Got: func() any {
					return querier.JaegerAPITraces(t, at.JaegerQueryParam{
						TraceQueryParam: query.TraceQueryParam{
							ServiceName:  serviceName,
							StartTimeMin: spanTime.Add(-10 * time.Minute),
							StartTimeMax: spanTime.Add(10 * time.Minute),
							Attributes: map[string]string{
								"testTag": "~INVALID.*",
							},
						},
					}, at.QueryOpts{})
				},
				Want: &at.JaegerAPITracesResponse{
					Data: []at.TracesResponseData{},
				},
				CmpOpts: []cmp.Option{
					cmpopts.IgnoreFields(at.JaegerAPITracesResponse{}, "Errors", "Limit", "Offset", "Total"),
				},
			})

			// check traces data via /select/jaeger/api/traces with regex tag filter: match
			tc.Assert(&at.AssertOptions{
				Msg: "unexpected /select/jaeger/api/traces response",
				Got: func() any {
					return querier.JaegerAPITraces(t, at.JaegerQueryParam{
						TraceQueryParam: query.TraceQueryParam{
							ServiceName:  serviceName,
							StartTimeMin: spanTime.Add(-10 * time.Minute),
							StartTimeMax: spanTime.Add(10 * time.Minute),
							Attributes: map[string]string{
								"testTag": "~test.*",
							},
						},
					}, at.QueryOpts{})
				},
				Want: &at.JaegerAPITracesResponse{
					Data: expectTraceData,
				},
				CmpOpts: []cmp.Option{
					cmpopts.IgnoreFields(at.JaegerAPITracesResponse{}, "Errors", "Limit", "Offset", "Total"),
					sortTraces,
				},
			})
		}
	}

	return reqs, assertFunc
}
