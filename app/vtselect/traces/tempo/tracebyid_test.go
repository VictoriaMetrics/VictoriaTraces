package tempo

import (
	"encoding/json"
	"testing"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

func testResourceSpans() []*otelpb.ResourceSpans {
	svc := "svc-a"
	method := "GET"
	return []*otelpb.ResourceSpans{
		{
			Resource: otelpb.Resource{
				Attributes: []*otelpb.KeyValue{
					{Key: "service.name", Value: &otelpb.AnyValue{StringValue: &svc}},
				},
			},
			ScopeSpans: []*otelpb.ScopeSpans{
				{
					Scope: otelpb.InstrumentationScope{Name: "scope", Version: "1.0"},
					Spans: []*otelpb.Span{
						{
							TraceID:           "0a0b", // -> base64 "Cgs="
							SpanID:            "0c0d", // -> base64 "DA0="
							ParentSpanID:      "",     // root: omitted
							Name:              "op",
							Kind:              otelpb.SpanKind(2), // SPAN_KIND_SERVER
							StartTimeUnixNano: 100,
							EndTimeUnixNano:   200,
							Attributes: []*otelpb.KeyValue{
								{Key: "http.method", Value: &otelpb.AnyValue{StringValue: &method}},
							},
							Status: otelpb.Status{Code: otelpb.StatusCode(2), Message: "boom"},
						},
					},
				},
			},
		},
	}
}

// TestTraceByIDV1JSON verifies the bare Trace JSON shape of the Tempo
// /api/traces/<trace_id> (v1) API: resource spans nested under "batches", base64
// IDs, enum-name kind, string-encoded nanos, omitted root parentSpanId.
func TestTraceByIDV1JSON(t *testing.T) {
	out := TraceByIDV1JSON(testResourceSpans())
	if !json.Valid([]byte(out)) {
		t.Fatalf("v1 output is not valid JSON: %s", out)
	}

	var got struct {
		Batches []struct {
			Resource struct {
				Attributes []struct {
					Key   string `json:"key"`
					Value struct {
						StringValue string `json:"stringValue"`
					} `json:"value"`
				} `json:"attributes"`
			} `json:"resource"`
			ScopeSpans []struct {
				Spans []map[string]json.RawMessage `json:"spans"`
			} `json:"scopeSpans"`
		} `json:"batches"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("cannot unmarshal v1 output: %s\n%s", err, out)
	}

	if len(got.Batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(got.Batches))
	}
	if got.Batches[0].Resource.Attributes[0].Value.StringValue != "svc-a" {
		t.Fatalf("unexpected resource attribute: %+v", got.Batches[0].Resource.Attributes)
	}

	span := got.Batches[0].ScopeSpans[0].Spans[0]
	assertJSONField(t, span, "traceId", `"Cgs="`)
	assertJSONField(t, span, "spanId", `"DA0="`)
	assertJSONField(t, span, "kind", `"SPAN_KIND_SERVER"`)
	assertJSONField(t, span, "startTimeUnixNano", `"100"`)
	assertJSONField(t, span, "endTimeUnixNano", `"200"`)
	assertJSONField(t, span, "status", `{"message":"boom","code":"STATUS_CODE_ERROR"}`)
	if _, ok := span["parentSpanId"]; ok {
		t.Fatalf("root span must omit parentSpanId, got %s", span["parentSpanId"])
	}
}

// TestTraceByIDV2JSON verifies the TraceByIDResponse wrapper of the Tempo
// /api/v2/traces/<trace_id> API: resource spans under trace.resourceSpans plus metrics.
func TestTraceByIDV2JSON(t *testing.T) {
	out := TraceByIDV2JSON(testResourceSpans())
	if !json.Valid([]byte(out)) {
		t.Fatalf("v2 output is not valid JSON: %s", out)
	}

	var got struct {
		Trace struct {
			ResourceSpans []json.RawMessage `json:"resourceSpans"`
		} `json:"trace"`
		Metrics struct {
			InspectedBytes string `json:"inspectedBytes"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("cannot unmarshal v2 output: %s\n%s", err, out)
	}
	if len(got.Trace.ResourceSpans) != 1 {
		t.Fatalf("expected 1 resourceSpans entry, got %d", len(got.Trace.ResourceSpans))
	}
	if got.Metrics.InspectedBytes != "0" {
		t.Fatalf("expected metrics.inspectedBytes=\"0\", got %q", got.Metrics.InspectedBytes)
	}
}

func assertJSONField(t *testing.T, m map[string]json.RawMessage, key, want string) {
	t.Helper()
	got, ok := m[key]
	if !ok {
		t.Fatalf("missing field %q", key)
	}
	if string(got) != want {
		t.Fatalf("field %q: got %s, want %s", key, got, want)
	}
}
