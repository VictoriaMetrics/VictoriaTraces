package pb

import (
	"encoding/json"
	"testing"
)

// emptySpanJSON returns the OTLP/JSON of a span which carries nothing but a name.
func emptySpanJSON(name string) string {
	return `{"traceId":"","spanId":"","traceState":"","parentSpanId":"","flags":0,"name":"` + name +
		`","kind":0,"startTimeUnixNano":"0","endTimeUnixNano":"0","attributes":[],` +
		`"droppedAttributesCount":0,"events":[],"droppedEventsCount":0,"links":[],` +
		`"droppedLinksCount":0,"status":{"message":"","code":0}}`
}

func TestWriteTracesDataJSON(t *testing.T) {
	f := func(rss []*ResourceSpans, expected string) {
		t.Helper()
		got := TracesDataJSON(rss)
		if got != expected {
			t.Fatalf("unexpected OTLP/JSON;\ngot\n%s\nwant\n%s", got, expected)
		}
		// the result must be valid JSON, since the template writes it by hand.
		if !json.Valid([]byte(got)) {
			t.Fatalf("invalid JSON: %s", got)
		}
	}

	f(nil, `{"resourceSpans":[]}`)

	// a KeyValue may carry no value, and the empty object must still be closed.
	f([]*ResourceSpans{
		{Resource: Resource{Attributes: []*KeyValue{{Key: "foo"}}}},
	}, `{"resourceSpans":[{"resource":{"attributes":[{"key":"foo","value":{}}]},`+
		`"scopeSpans":[],"schemaUrl":""}]}`)

	// more than one item per list, so that a missing separator is caught.
	f([]*ResourceSpans{
		{
			Resource: Resource{Attributes: []*KeyValue{
				{Key: "a", Value: &AnyValue{StringValue: strptr("1")}},
				{Key: "b", Value: &AnyValue{StringValue: strptr("2")}},
			}},
			ScopeSpans: []*ScopeSpans{
				{Spans: []*Span{{Name: "s1"}, {Name: "s2"}}},
				{Spans: []*Span{{Name: "s3"}}},
			},
		},
		{Resource: Resource{Attributes: []*KeyValue{}}},
	}, `{"resourceSpans":[{"resource":{"attributes":[{"key":"a","value":{"stringValue":"1"}},`+
		`{"key":"b","value":{"stringValue":"2"}}]},"scopeSpans":[`+
		`{"scope":{"name":"","version":"","attributes":[],"droppedAttributesCount":0},"spans":[`+
		emptySpanJSON("s1")+`,`+emptySpanJSON("s2")+`],"schemaUrl":""},`+
		`{"scope":{"name":"","version":"","attributes":[],"droppedAttributesCount":0},"spans":[`+
		emptySpanJSON("s3")+`],"schemaUrl":""}],"schemaUrl":""},`+
		`{"resource":{"attributes":[]},"scopeSpans":[],"schemaUrl":""}]}`)

	intValue := int64(42)
	f([]*ResourceSpans{
		{
			Resource: Resource{
				Attributes: []*KeyValue{
					{Key: "service.name", Value: &AnyValue{StringValue: strptr("foo")}},
				},
			},
			ScopeSpans: []*ScopeSpans{
				{
					Scope: InstrumentationScope{
						Name:    "bar",
						Version: "1.2.3",
					},
					Spans: []*Span{
						{
							TraceID:           "0123456789abcdef0123456789abcdef",
							SpanID:            "0123456789abcdef",
							ParentSpanID:      "fedcba9876543210",
							Name:              "baz",
							Kind:              2,
							StartTimeUnixNano: 1000,
							EndTimeUnixNano:   2000,
							Attributes: []*KeyValue{
								{Key: "http.status_code", Value: &AnyValue{IntValue: &intValue}},
							},
							Status: Status{Code: 2, Message: "error"},
						},
					},
				},
			},
		},
	}, `{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"foo"}}]},`+
		`"scopeSpans":[{"scope":{"name":"bar","version":"1.2.3","attributes":[],"droppedAttributesCount":0},`+
		`"spans":[{"traceId":"0123456789abcdef0123456789abcdef","spanId":"0123456789abcdef","traceState":"",`+
		`"parentSpanId":"fedcba9876543210","flags":0,"name":"baz","kind":2,"startTimeUnixNano":"1000",`+
		`"endTimeUnixNano":"2000","attributes":[{"key":"http.status_code","value":{"intValue":"42"}}],`+
		`"droppedAttributesCount":0,"events":[],"droppedEventsCount":0,"links":[],"droppedLinksCount":0,`+
		`"status":{"message":"error","code":2}}],"schemaUrl":""}],"schemaUrl":""}]}`)
}
