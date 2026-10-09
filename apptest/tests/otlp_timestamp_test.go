package tests

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/fs"
	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

func TestOTLPIngestionTimestampConsistency(t *testing.T) {
	fs.MustRemoveDir(t.Name())

	tc := at.NewTestCase(t)
	defer tc.Stop()

	dataPath := tc.Dir() + "/vtsingle"
	sut := tc.MustStartVtsingle("vtsingle", []string{
		"-storageDataPath=" + dataPath,
		"-retentionPeriod=100y",
		"-search.latencyOffset=0s",
		"-insert.indexFlushInterval=1s",
	})

	baseT := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	spanDuration := 15 * time.Minute
	startNano := uint64(baseT.UnixNano())
	endNano := uint64(baseT.Add(spanDuration).UnixNano())

	serviceName := "timestamp-consistency-svc"
	spanName := "test-span-15m"

	traceIDJSON := "11111111111111111111111111111111"
	spanIDJSON := "1111111111111111"

	traceIDPBHTTP := "22222222222222222222222222222222"
	spanIDPBHTTP := "2222222222222222"

	traceIDPBGRPC := "33333333333333333333333333333333"
	spanIDPBGRPC := "3333333333333333"

	// 1. Ingest JSON via HTTP
	jsonPayload := fmt.Sprintf(`{
		"resourceSpans": [{
			"resource": {
				"attributes": [
					{"key": "service.name", "value": {"stringValue": "%s"}}
				]
			},
			"scopeSpans": [{
				"scope": {"name": "test-scope"},
				"spans": [{
					"traceId": "%s",
					"spanId": "%s",
					"name": "%s",
					"kind": 1,
					"startTimeUnixNano": "%d",
					"endTimeUnixNano": "%d",
					"attributes": [
						{"key": "encoding", "value": {"stringValue": "json"}}
					]
				}]
			}]
		}]
	}`, serviceName, traceIDJSON, spanIDJSON, spanName, startNano, endNano)

	sut.OTLPHTTPExportRawTraces(t, []byte(jsonPayload), at.QueryOpts{
		HTTPHeaders: map[string]string{
			"Content-Type": "application/json",
		},
	})

	// 2. Ingest Protobuf via HTTP
	makePBReq := func(traceID, spanID, encoding string) *otelpb.ExportTraceServiceRequest {
		enc := encoding
		svc := serviceName
		return &otelpb.ExportTraceServiceRequest{
			ResourceSpans: []*otelpb.ResourceSpans{
				{
					Resource: otelpb.Resource{
						Attributes: []*otelpb.KeyValue{
							{Key: "service.name", Value: &otelpb.AnyValue{StringValue: &svc}},
						},
					},
					ScopeSpans: []*otelpb.ScopeSpans{
						{
							Scope: otelpb.InstrumentationScope{Name: "test-scope"},
							Spans: []*otelpb.Span{
								{
									TraceID:           traceID,
									SpanID:            spanID,
									Name:              spanName,
									Kind:              1,
									StartTimeUnixNano: startNano,
									EndTimeUnixNano:   endNano,
									Attributes: []*otelpb.KeyValue{
										{Key: "encoding", Value: &otelpb.AnyValue{StringValue: &enc}},
									},
								},
							},
						},
					},
				},
			},
		}
	}

	reqPBHTTP := makePBReq(traceIDPBHTTP, spanIDPBHTTP, "protobuf-http")
	sut.OTLPHTTPExportTraces(t, reqPBHTTP, at.QueryOpts{})

	// 3. Ingest Protobuf via gRPC
	reqPBGRPC := makePBReq(traceIDPBGRPC, spanIDPBGRPC, "protobuf-grpc")
	sut.OTLPgRPCExportTraces(t, reqPBGRPC, at.QueryOpts{})

	sut.ForceFlush(t)

	// Window 1: Covers start time, excludes end time: [T - 1m, T + 1m]
	w1Start := baseT.Add(-1 * time.Minute).UnixNano()
	w1End := baseT.Add(1 * time.Minute).UnixNano()
	resW1 := sut.LogsQLQuery(t, fmt.Sprintf(`{resource_attr:service.name=%q}`, serviceName), at.QueryOpts{
		Start: strconv.FormatInt(w1Start, 10),
		End:   strconv.FormatInt(w1End, 10),
	})
	if len(resW1.LogLines) != 0 {
		t.Fatalf("expected 0 spans in start window [T-1m, T+1m], got %d", len(resW1.LogLines))
	}

	// Window 2: Covers end time, excludes start time: [T + 14m, T + 16m]
	w2Start := baseT.Add(14 * time.Minute).UnixNano()
	w2End := baseT.Add(16 * time.Minute).UnixNano()
	resW2 := sut.LogsQLQuery(t, fmt.Sprintf(`{resource_attr:service.name=%q}`, serviceName), at.QueryOpts{
		Start: strconv.FormatInt(w2Start, 10),
		End:   strconv.FormatInt(w2End, 10),
	})
	if len(resW2.LogLines) != 3 {
		t.Fatalf("expected 3 spans in end window [T+14m, T+16m], got %d", len(resW2.LogLines))
	}

	expectedTimeStr := baseT.Add(spanDuration).UTC().Format(time.RFC3339Nano)
	for _, line := range resW2.LogLines {
		var row map[string]string
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("cannot unmarshal log line: %v", err)
		}
		if row["_time"] != expectedTimeStr {
			t.Errorf("span %s (%s) got _time=%s, want %s", row["trace_id"], row["span_attr:encoding"], row["_time"], expectedTimeStr)
		}
	}
}
