package jaeger

import (
	"reflect"
	"testing"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/tracecommon"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// newSummaryRow builds a span row with the fields read by summarizeTraces.
func newSummaryRow(traceID, spanID, parentSpanID, name, serviceName, startTime, endTime, statusCode string) *tracecommon.Row {
	return &tracecommon.Row{
		Fields: []logstorage.Field{
			{Name: otelpb.TraceIDField, Value: traceID},
			{Name: otelpb.SpanIDField, Value: spanID},
			{Name: otelpb.ParentSpanIDField, Value: parentSpanID},
			{Name: otelpb.NameField, Value: name},
			{Name: otelpb.ResourceAttrServiceName, Value: serviceName},
			{Name: otelpb.StartTimeUnixNanoField, Value: startTime},
			{Name: otelpb.EndTimeUnixNanoField, Value: endTime},
			{Name: otelpb.StatusCodeField, Value: statusCode},
		},
	}
}

func TestSummarizeTracesSuccess(t *testing.T) {
	f := func(rows []*tracecommon.Row, expected []traceSummaryV3) {
		t.Helper()
		result, err := summarizeTraces(rows)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("unexpected summaries;\ngot\n%+v\nwant\n%+v", result, expected)
		}
	}

	f(nil, []traceSummaryV3{})

	// a single root span.
	f([]*tracecommon.Row{
		newSummaryRow("t1", "s1", "", "root", "foo", "100", "200", "0"),
	}, []traceSummaryV3{
		{
			traceID:              "t1",
			rootServiceName:      "foo",
			rootOperationName:    "root",
			minStartTimeUnixNano: 100,
			maxEndTimeUnixNano:   200,
			spanCount:            1,
			services:             []serviceSummaryV3{{name: "foo", spanCount: 1}},
		},
	})

	// two services, one of them reports an error. The time range spans both.
	f([]*tracecommon.Row{
		newSummaryRow("t1", "s1", "", "root", "foo", "100", "500", "0"),
		newSummaryRow("t1", "s2", "s1", "child", "bar", "150", "400", "2"),
	}, []traceSummaryV3{
		{
			traceID:              "t1",
			rootServiceName:      "foo",
			rootOperationName:    "root",
			minStartTimeUnixNano: 100,
			maxEndTimeUnixNano:   500,
			spanCount:            2,
			errorSpanCount:       1,
			services: []serviceSummaryV3{
				{name: "bar", spanCount: 1, errorSpanCount: 1},
				{name: "foo", spanCount: 1},
			},
		},
	})

	// a span whose parent is missing from the trace is an orphan.
	f([]*tracecommon.Row{
		newSummaryRow("t1", "s1", "", "root", "foo", "100", "500", "0"),
		newSummaryRow("t1", "s2", "missing", "orphan", "foo", "150", "400", "0"),
	}, []traceSummaryV3{
		{
			traceID:              "t1",
			rootServiceName:      "foo",
			rootOperationName:    "root",
			minStartTimeUnixNano: 100,
			maxEndTimeUnixNano:   500,
			spanCount:            2,
			orphanSpanCount:      1,
			services:             []serviceSummaryV3{{name: "foo", spanCount: 2}},
		},
	})

	// the earliest span without a parent wins, even when it isn't the first row.
	f([]*tracecommon.Row{
		newSummaryRow("t1", "s1", "", "later", "foo", "300", "500", "0"),
		newSummaryRow("t1", "s2", "", "earlier", "bar", "100", "400", "0"),
	}, []traceSummaryV3{
		{
			traceID:              "t1",
			rootServiceName:      "bar",
			rootOperationName:    "earlier",
			minStartTimeUnixNano: 100,
			maxEndTimeUnixNano:   500,
			spanCount:            2,
			services: []serviceSummaryV3{
				{name: "bar", spanCount: 1},
				{name: "foo", spanCount: 1},
			},
		},
	})

	// a trace without a root span reports empty root fields instead of failing.
	f([]*tracecommon.Row{
		newSummaryRow("t1", "s1", "missing", "child", "foo", "100", "200", "0"),
	}, []traceSummaryV3{
		{
			traceID:              "t1",
			minStartTimeUnixNano: 100,
			maxEndTimeUnixNano:   200,
			spanCount:            1,
			orphanSpanCount:      1,
			services:             []serviceSummaryV3{{name: "foo", spanCount: 1}},
		},
	})

	// multiple traces are returned newest first.
	f([]*tracecommon.Row{
		newSummaryRow("older", "s1", "", "root", "foo", "100", "200", "0"),
		newSummaryRow("newer", "s2", "", "root", "foo", "300", "400", "0"),
	}, []traceSummaryV3{
		{
			traceID:              "newer",
			rootServiceName:      "foo",
			rootOperationName:    "root",
			minStartTimeUnixNano: 300,
			maxEndTimeUnixNano:   400,
			spanCount:            1,
			services:             []serviceSummaryV3{{name: "foo", spanCount: 1}},
		},
		{
			traceID:              "older",
			rootServiceName:      "foo",
			rootOperationName:    "root",
			minStartTimeUnixNano: 100,
			maxEndTimeUnixNano:   200,
			spanCount:            1,
			services:             []serviceSummaryV3{{name: "foo", spanCount: 1}},
		},
	})
}

func TestSummarizeTracesFailure(t *testing.T) {
	f := func(rows []*tracecommon.Row) {
		t.Helper()
		if _, err := summarizeTraces(rows); err == nil {
			t.Fatalf("expecting non-nil error")
		}
	}

	// a row without the trace id.
	f([]*tracecommon.Row{
		{Fields: []logstorage.Field{{Name: otelpb.SpanIDField, Value: "s1"}}},
	})

	// a row with an unparsable start time.
	f([]*tracecommon.Row{
		newSummaryRow("t1", "s1", "", "root", "foo", "not-a-number", "200", "0"),
	})
}
