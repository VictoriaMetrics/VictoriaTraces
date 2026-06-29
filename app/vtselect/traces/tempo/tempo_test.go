package tempo

import (
	"bytes"
	"context"
	"encoding/json"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
	"slices"
	"testing"
)

// Intrinsic span fields have no attribute prefix, so they are not discovered by
// the field_names scan. They must be advertised explicitly under the "intrinsic"
// scope of /api/v2/search/tags, otherwise they never appear in the Grafana
// Traces Drilldown attribute breakdown.
func TestSearchTagsIntrinsicScope(t *testing.T) {
	res, err := searchTags(context.Background(), nil, "{}", "intrinsic", 0, 0, 100)
	if err != nil {
		t.Fatalf("searchTags(scope=intrinsic): %s", err)
	}
	for _, want := range []string{"name", "kind", "status", "duration"} {
		if !slices.Contains(res.intrinsicTagList, want) {
			t.Fatalf("intrinsic scope missing %q; got %v", want, res.intrinsicTagList)
		}
	}
	// scope=intrinsic must not leak into other scopes.
	if len(res.spanTagList) != 0 || len(res.resourceTagList) != 0 {
		t.Fatalf("intrinsic scope must not populate span/resource lists; got span=%v resource=%v",
			res.spanTagList, res.resourceTagList)
	}
}

// The /api/v2/search/tags response must expose an "intrinsic" scope carrying the
// intrinsic tag names, matching Tempo's response shape.
func TestSearchTagsResponseIncludesIntrinsicScope(t *testing.T) {
	out := SearchTagsResponse(nil, nil, nil, nil, nil, []string{"name", "kind", "status", "duration"})

	var parsed struct {
		Scopes []struct {
			Name string   `json:"name"`
			Tags []string `json:"tags"`
		} `json:"scopes"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("response is not valid JSON: %s\n%s", err, out)
	}

	var intrinsic []string
	found := false
	for _, sc := range parsed.Scopes {
		if sc.Name == "intrinsic" {
			found = true
			intrinsic = sc.Tags
		}
	}
	if !found {
		t.Fatalf("response missing intrinsic scope: %s", out)
	}
	for _, want := range []string{"name", "kind", "status", "duration"} {
		if !slices.Contains(intrinsic, want) {
			t.Fatalf("intrinsic scope missing %q; got %v", want, intrinsic)
		}
	}
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
