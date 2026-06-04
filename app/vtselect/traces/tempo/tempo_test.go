package tempo

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/tracecommon"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// spanRow builds a *tracecommon.Row from the given field name/value pairs.
func spanRow(fields map[string]string) *tracecommon.Row {
	r := &tracecommon.Row{}
	for name, value := range fields {
		r.Fields = append(r.Fields, logstorage.Field{Name: name, Value: value})
	}
	return r
}

// TestSummarySearchTracesResult verifies that the search response is populated
// with the spans that satisfied the filter (not the trace root span), and that
// the matched count is independent of whether the root span is present.
func TestSummarySearchTracesResult(t *testing.T) {
	const (
		traceID     = "0123456789abcdef"
		rootSpanID  = "1111111111111111"
		matchSpanID = "2222222222222222"
	)

	// A root span that does NOT satisfy the filter.
	rootRow := spanRow(map[string]string{
		otelpb.TraceIDField:            traceID,
		otelpb.SpanIDField:             rootSpanID,
		otelpb.ParentSpanIDField:       "",
		otelpb.NameField:               "root-span",
		otelpb.ResourceAttrServiceName: "service-a",
		otelpb.StartTimeUnixNanoField:  "1000",
		otelpb.EndTimeUnixNanoField:    "5000",
	})
	// A child span that satisfies the filter. The matchedMarkerField is set by
	// query 2's 'format if' pipe on spans that match the filter.
	matchRow := spanRow(map[string]string{
		otelpb.TraceIDField:                      traceID,
		otelpb.SpanIDField:                       matchSpanID,
		otelpb.ParentSpanIDField:                 rootSpanID,
		otelpb.NameField:                         "match-span",
		otelpb.ResourceAttrServiceName:           "service-b",
		otelpb.SpanAttrPrefixField + "test_attr": "test_value",
		otelpb.StartTimeUnixNanoField:            "2000",
		otelpb.EndTimeUnixNanoField:              "4500",
		matchedMarkerField:                       matchedMarker,
	})

	referenced := []string{otelpb.NameField, "span.test_attr"}

	// f asserts the returned matched span has the expected id/name and that the
	// matched count equals wantCount for the given rows.
	f := func(name string, rows []*tracecommon.Row, wantSpanID string, wantCount int, wantRootService string) {
		t.Helper()
		res, err := summarySearchTracesResult(rows, referenced, 3)
		if err != nil {
			t.Fatalf("%s: unexpected error: %s", name, err)
		}
		if len(res) != 1 {
			t.Fatalf("%s: got %d summaries; want 1", name, len(res))
		}
		s := res[0]
		if s.matchedCount != wantCount {
			t.Fatalf("%s: matchedCount=%d; want %d", name, s.matchedCount, wantCount)
		}
		if len(s.matchedSpans) != 1 {
			t.Fatalf("%s: len(matchedSpans)=%d; want 1", name, len(s.matchedSpans))
		}
		ms := s.matchedSpans[0]
		if ms.spanID != wantSpanID {
			t.Fatalf("%s: matched spanID=%q; want %q (root substituted?)", name, ms.spanID, wantSpanID)
		}
		if ms.name != "match-span" {
			t.Fatalf("%s: matched span name=%q; want %q", name, ms.name, "match-span")
		}
		if s.rootServiceName != wantRootService {
			t.Fatalf("%s: rootServiceName=%q; want %q", name, s.rootServiceName, wantRootService)
		}
	}

	// Decisive case: the root span exists and does not match; the matched child
	// span must be returned, and trace metadata still comes from the root span.
	f("root present but non-matching", []*tracecommon.Row{rootRow, matchRow}, matchSpanID, 1, "service-a")

	// Root span not yet ingested: the matched child span and its count must still
	// be reported, independent of root-span presence.
	f("root span absent", []*tracecommon.Row{matchRow}, matchSpanID, 1, "<root span not yet received>")
}

// TestTraceByIDV1JSON verifies the bare Trace JSON shape of the Tempo
// /api/traces/<trace_id> (v1) API: resource spans nested under "batches", base64
// IDs, enum-name kind, string-encoded nanos, omitted root parentSpanId.
func TestTraceByIDV1JSON(t *testing.T) {
	response := TraceByIDV1JSON(generateResourceSpans())
	expect := `{
  "batches": [
    {
      "resource": {
        "attributes": [
          {
            "key": "service.name",
            "value": {
              "stringValue": "svc-a"
            }
          }
        ]
      },
      "scopeSpans": [
        {
          "scope": {
            "name": "scope",
            "version": "1.0",
            "attributes": []
          },
          "spans": [
            {
              "traceId": "Cgs=",
              "spanId": "DA0=",
              "name": "op",
              "kind": "SPAN_KIND_SERVER",
              "startTimeUnixNano": "100",
              "endTimeUnixNano": "200",
              "attributes": [
                {
                  "key": "http.method",
                  "value": {
                    "stringValue": "GET"
                  }
                }
              ],
              "status": {
                "message": "boom",
                "code": "STATUS_CODE_ERROR"
              }
            }
          ]
        }
      ]
    }
  ]
}`

	// compact the response and compare
	compactResponse, compactExpect := new(bytes.Buffer), new(bytes.Buffer)
	err1 := json.Compact(compactResponse, []byte(response))
	err2 := json.Compact(compactExpect, []byte(expect))
	if err1 != nil || err2 != nil {
		t.Fatalf("got error when json.Compact: err of compacting response: %v, err of compacting expect: %v", err1, err2)
	}

	if compactExpect.String() != compactResponse.String() {
		t.Fatalf("got %q; want %q", compactExpect.String(), compactResponse.String())
	}
}

// TestTraceByIDV2JSON verifies the TraceByIDResponse wrapper of the Tempo
// /api/v2/traces/<trace_id> API: resource spans under trace.resourceSpans plus metrics.
func TestTraceByIDV2JSON(t *testing.T) {
	response := TraceByIDV2JSON(generateResourceSpans())
	expect := `{
  "trace": {
    "resourceSpans": [
      {
        "resource": {
          "attributes": [
            {
              "key": "service.name",
              "value": {
                "stringValue": "svc-a"
              }
            }
          ]
        },
        "scopeSpans": [
          {
            "scope": {
              "name": "scope",
              "version": "1.0",
              "attributes": []
            },
            "spans": [
              {
                "traceId": "Cgs=",
                "spanId": "DA0=",
                "name": "op",
                "kind": "SPAN_KIND_SERVER",
                "startTimeUnixNano": "100",
                "endTimeUnixNano": "200",
                "attributes": [
                  {
                    "key": "http.method",
                    "value": {
                      "stringValue": "GET"
                    }
                  }
                ],
                "status": {
                  "message": "boom",
                  "code": "STATUS_CODE_ERROR"
                }
              }
            ]
          }
        ]
      }
    ]
  },
  "metrics": {
    "inspectedBytes": "0"
  }
}`

	// compact the response and compare
	compactResponse, compactExpect := new(bytes.Buffer), new(bytes.Buffer)
	err1 := json.Compact(compactResponse, []byte(response))
	err2 := json.Compact(compactExpect, []byte(expect))
	if err1 != nil || err2 != nil {
		t.Fatalf("got error when json.Compact: err of compacting response: %v, err of compacting expect: %v", err1, err2)
	}

	if compactExpect.String() != compactResponse.String() {
		t.Fatalf("got %q; want %q", compactExpect.String(), compactResponse.String())
	}
}

// generateResourceSpans returns []*otelpb.ResourceSpans for testing.
func generateResourceSpans() []*otelpb.ResourceSpans {
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
