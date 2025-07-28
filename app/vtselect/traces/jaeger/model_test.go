package jaeger

import (
	"testing"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/google/go-cmp/cmp"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

func TestFieldsToSpan(t *testing.T) {
	f := func(input []logstorage.Field, want *span, errorMsg string) {
		t.Helper()

		var errMsgGot string
		got, err := fieldsToSpan(input)
		if err != nil {
			errMsgGot = err.Error()
		}
		if errMsgGot != errorMsg {
			t.Fatalf("fieldsToSpan() error = %v, want err: %v", err, errorMsg)
		}
		cmpOpts := cmp.AllowUnexported(span{}, process{}, spanRef{}, keyValue{}, log{})
		if !cmp.Equal(got, want, cmpOpts) {
			t.Fatalf("fieldsToSpan() diff = %v", cmp.Diff(got, want, cmpOpts))
		}
	}

	// case 1: empty
	f([]logstorage.Field{}, nil, "invalid fields: []")

	// case 2: without span_id
	fields := []logstorage.Field{
		{Name: otelpb.TraceIDField, Value: "1234567890"},
	}
	f(fields, nil, "invalid fields: [{trace_id 1234567890}]")

	// case 3: without trace_id
	fields = []logstorage.Field{
		{Name: otelpb.SpanIDField, Value: "12345"},
	}
	f(fields, nil, "invalid fields: [{span_id 12345}]")

	// case 4: with basic fields
	fields = []logstorage.Field{
		{Name: otelpb.TraceIDField, Value: "1234567890"},
		{Name: otelpb.SpanIDField, Value: "12345"},
	}
	sp := &span{
		traceID: "1234567890", spanID: "12345",
	}
	f(fields, sp, "")

	// case 5: with all fields
	// see: lib/protoparser/opentelemetry/pb/trace_fields.go
	fields = []logstorage.Field{
		{Name: otelpb.ResourceAttrServiceName, Value: "service_name_1"},
		{Name: otelpb.ResourceAttrPrefix + "resource_attr_1", Value: "resource_attr_1"},
		{Name: otelpb.ResourceAttrPrefix + "resource_attr_2", Value: "resource_attr_2"},

		{Name: otelpb.InstrumentationScopeName, Value: "scope_name_1"},
		{Name: otelpb.InstrumentationScopeVersion, Value: "scope_version_1"},
		{Name: otelpb.InstrumentationScopeAttrPrefix + "scope_attr_1", Value: "scope_attr_1"},
		{Name: otelpb.InstrumentationScopeAttrPrefix + "scope_attr_2", Value: "scope_attr_2"},

		{Name: otelpb.TraceIDField, Value: "1234567890"},
		{Name: otelpb.SpanIDField, Value: "12345"},
		{Name: otelpb.TraceStateField, Value: "trace_state_1"},
		{Name: otelpb.ParentSpanIDField, Value: "23456"},
		{Name: otelpb.FlagsField, Value: "0"},
		{Name: otelpb.NameField, Value: "span_name_1"},
		{Name: otelpb.KindField, Value: "1"},
		{Name: otelpb.StartTimeUnixNanoField, Value: "0"},
		{Name: otelpb.EndTimeUnixNanoField, Value: "123456789"},
		{Name: otelpb.SpanAttrPrefixField + "attr_1", Value: "attr_1"},
		{Name: otelpb.SpanAttrPrefixField + "attr_2", Value: "attr_2"},
		{Name: otelpb.DurationField, Value: "123456789"},

		{Name: otelpb.EventPrefix + otelpb.EventTimeUnixNanoField + ":0", Value: "0"},
		{Name: otelpb.EventPrefix + otelpb.EventNameField + ":0", Value: "event_0"},
		{Name: otelpb.EventPrefix + otelpb.EventAttrPrefix + "event_attr_1" + ":0", Value: "event_0_attr_1"},
		{Name: otelpb.EventPrefix + otelpb.EventAttrPrefix + "event_attr_2" + ":0", Value: "event_0_attr_2"},

		{Name: otelpb.EventPrefix + otelpb.EventTimeUnixNanoField + ":1", Value: "1"},
		{Name: otelpb.EventPrefix + otelpb.EventNameField + ":1", Value: "event_1"},
		{Name: otelpb.EventPrefix + otelpb.EventAttrPrefix + "event_attr_1" + ":1", Value: "event_1_attr_1"},
		{Name: otelpb.EventPrefix + otelpb.EventAttrPrefix + "event_attr_2" + ":1", Value: "event_1_attr_2"},

		{Name: otelpb.LinkPrefix + otelpb.LinkTraceIDField + ":0", Value: "1234567890"},
		{Name: otelpb.LinkPrefix + otelpb.LinkSpanIDField + ":0", Value: "23456"},
		{Name: otelpb.LinkPrefix + otelpb.LinkTraceStateField + ":0", Value: "link_0_trace_state_1"},
		{Name: otelpb.LinkPrefix + otelpb.LinkAttrPrefix + "link_attr_1" + ":0", Value: "link_0_trace_attr_1"},
		{Name: otelpb.LinkPrefix + otelpb.LinkAttrPrefix + "link_attr_2" + ":0", Value: "link_0_trace_attr_2"},
		{Name: otelpb.LinkPrefix + otelpb.LinkAttrPrefix + "opentracing.ref_type" + ":0", Value: "child_of"},
		{Name: otelpb.LinkPrefix + otelpb.LinkFlagsField + ":0", Value: "0"},
		{Name: otelpb.LinkPrefix + otelpb.LinkTraceIDField + ":1", Value: "99999999999"},
		{Name: otelpb.LinkPrefix + otelpb.LinkSpanIDField + ":1", Value: "98765"},
		{Name: otelpb.LinkPrefix + otelpb.LinkTraceStateField + ":1", Value: "link_1_trace_state_1"},
		{Name: otelpb.LinkPrefix + otelpb.LinkAttrPrefix + "link_attr_1" + ":1", Value: "link_1_trace_attr_1"},
		{Name: otelpb.LinkPrefix + otelpb.LinkAttrPrefix + "link_attr_2" + ":1", Value: "link_1_trace_attr_2"},
		{Name: otelpb.LinkPrefix + otelpb.LinkFlagsField + ":1", Value: "1"},

		{Name: otelpb.StatusMessageField, Value: "status_message_1"},
		{Name: otelpb.StatusCodeField, Value: "2"},
	}

	sp = &span{
		traceID:       "1234567890",
		spanID:        "12345",
		operationName: "span_name_1",
		references: []spanRef{
			{
				traceID: "1234567890",
				spanID:  "23456",
				refType: "CHILD_OF",
			},
			{
				traceID: "99999999999",
				spanID:  "98765",
				refType: "FOLLOWS_FROM",
			},
		},
		startTime: 0,
		duration:  123456,
		tags: []keyValue{
			{"otel.scope.name", "scope_name_1"},
			{"otel.scope.version", "scope_version_1"},
			{"scope_attr:scope_attr_1", "scope_attr_1"},
			{"scope_attr:scope_attr_2", "scope_attr_2"},
			{"w3c.tracestate", "trace_state_1"},
			{"span.kind", "internal"},
			{"attr_1", "attr_1"},
			{"attr_2", "attr_2"},
			{"otel.status_description", "status_message_1"},
			{"error", "true"},
		},
		logs: []log{
			{
				timestamp: 0,
				fields: []keyValue{
					{"event", "event_0"},
					{"event_attr_1", "event_0_attr_1"},
					{"event_attr_2", "event_0_attr_2"},
				},
			},
			{
				timestamp: 0,
				fields: []keyValue{
					{"event", "event_1"},
					{"event_attr_1", "event_1_attr_1"},
					{"event_attr_2", "event_1_attr_2"},
				},
			},
		},
		process: process{
			serviceName: "service_name_1",
			tags: []keyValue{
				{"resource_attr_1", "resource_attr_1"},
				{"resource_attr_2", "resource_attr_2"},
			},
		},
	}
	f(fields, sp, "")
}

func TestRemoveArrayIndex(t *testing.T) {
	f := func(input, wantName, wantIdx string) {
		t.Helper()
		if output, idx := extraAttributeNameAndIndex(input); output != wantName || idx != wantIdx {
			t.Fatalf("removeArrayIndex %q: got %q %q, want %q %q", input, output, idx, wantName, wantIdx)
		}
	}
	f("hello:1", "hello", "1")
	f("hello:my_index", "hello:my_index", "")
	f("hello", "hello", "")
	f("hello:my_index:99999", "hello:my_index", "99999")
	f("hello:my_index:", "hello:my_index:", "")
}
