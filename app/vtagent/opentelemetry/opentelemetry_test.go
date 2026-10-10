package opentelemetry

import (
	"testing"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

func TestCountSpansSuccess(t *testing.T) {
	f := func(req *otelpb.ExportTraceServiceRequest, spansCountExpected int) {
		t.Helper()

		n, err := countSpans(req.MarshalProtobuf(nil))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if n != spansCountExpected {
			t.Fatalf("unexpected number of spans; got %d; want %d", n, spansCountExpected)
		}
	}

	serviceName := "svc"
	newSpans := func(n int) []*otelpb.Span {
		spans := make([]*otelpb.Span, n)
		for i := range spans {
			spans[i] = &otelpb.Span{Name: "op", TraceID: "5b8efff798038103d269b633813fc60c", SpanID: "eee19b7ec3c1b174"}
		}
		return spans
	}

	// empty request
	f(&otelpb.ExportTraceServiceRequest{}, 0)

	// resource spans without spans
	f(&otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{{}, {SchemaURL: "foo"}},
	}, 0)

	// multiple resource spans and scope spans
	f(&otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{
				Resource: otelpb.Resource{
					Attributes: []*otelpb.KeyValue{
						{Key: "service.name", Value: &otelpb.AnyValue{StringValue: &serviceName}},
					},
				},
				ScopeSpans: []*otelpb.ScopeSpans{
					{Scope: otelpb.InstrumentationScope{Name: "scope1"}, Spans: newSpans(3)},
					{Scope: otelpb.InstrumentationScope{Name: "scope2"}, Spans: newSpans(1)},
				},
			},
			{
				ScopeSpans: []*otelpb.ScopeSpans{
					{Spans: newSpans(5)},
				},
			},
		},
	}, 9)
}

func TestCountSpansFailure(t *testing.T) {
	f := func(data []byte) {
		t.Helper()

		if _, err := countSpans(data); err == nil {
			t.Fatalf("expecting non-nil error")
		}
	}

	f([]byte("invalid protobuf"))

	// truncated request
	req := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{ScopeSpans: []*otelpb.ScopeSpans{{Spans: []*otelpb.Span{{Name: "op"}}}}},
		},
	}
	data := req.MarshalProtobuf(nil)
	f(data[:len(data)-1])

	// malformed scope_spans inside a well-formed resource_spans
	f([]byte{
		0x0a, 0x03, // resource_spans, length 3
		0x12, 0x01, // scope_spans, length 1
		0xff, // invalid field header
	})
}
