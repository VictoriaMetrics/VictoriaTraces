package tests

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/fs"
	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// TestSingleTempoAPISearchSpanEvents checks that the Tempo API finds a span by its
// events and links.
//
// Each event and link is stored under its own numbered field, so the assertions
// deliberately cover both ends: most match the second event or link, and a pair match
// the first, which proves the index wildcard spans every index rather than one of them.
func TestSingleTempoAPISearchSpanEvents(t *testing.T) {
	fs.MustRemoveDir(t.Name())

	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartDefaultVtsingle()

	traceID := "aa5886e99fffef35a847cb2d493fde01"
	// the linked ids differ from the span's own, so an assertion on link:traceID cannot
	// pass by accidentally reading the span's trace_id field.
	linkedTraceID0 := "cc5886e99fffef35a847cb2d493fde03"
	linkedTraceID := "bb5886e99fffef35a847cb2d493fde02"
	spanID := "12345678"
	linkedSpanID := "87654321"
	spanTime := time.Now()

	stringValue := func(s string) *otelpb.AnyValue {
		return &otelpb.AnyValue{StringValue: &s}
	}

	sut.OTLPHTTPExportTraces(t, &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{
				Resource: otelpb.Resource{
					Attributes: []*otelpb.KeyValue{
						{Key: "service.name", Value: stringValue("spanEventService")},
					},
				},
				ScopeSpans: []*otelpb.ScopeSpans{
					{
						Spans: []*otelpb.Span{
							{
								TraceID:           traceID,
								SpanID:            spanID,
								Name:              "spanEventOperation",
								Kind:              otelpb.SpanKind(1),
								StartTimeUnixNano: uint64(spanTime.UnixNano()),
								EndTimeUnixNano:   uint64(spanTime.UnixNano()),
								Events: []*otelpb.SpanEvent{
									{
										TimeUnixNano: uint64(spanTime.UnixNano()),
										Name:         "first event",
									},
									{
										TimeUnixNano: uint64(spanTime.UnixNano()),
										Name:         "exception",
										Attributes: []*otelpb.KeyValue{
											{Key: "exception.type", Value: stringValue("OutOfMemoryError")},
										},
									},
								},
								Links: []*otelpb.SpanLink{
									{
										TraceID: linkedTraceID0,
										SpanID:  spanID,
									},
									{
										TraceID: linkedTraceID,
										SpanID:  linkedSpanID,
										Attributes: []*otelpb.KeyValue{
											{Key: "link.kind", Value: stringValue("follows-from")},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, at.QueryOpts{})
	sut.ForceFlush(t)
	time.Sleep(2 * time.Second) // index will be created after -insert.traceMaxDuration (2s in integration test)

	f := func(traceQL string, wantTraceIDs []string) {
		t.Helper()

		tc.Assert(&at.AssertOptions{
			Msg: "unexpected /select/tempo/api/search response for " + traceQL,
			Got: func() any {
				return sut.TempoAPISearch(t, traceQL, at.QueryOpts{}).TraceIDs()
			},
			Want:    wantTraceIDs,
			CmpOpts: []cmp.Option{cmpopts.EquateEmpty()},
		})
	}

	// the span itself is reachable, so a miss below is about the event, not the setup.
	f(`{name="spanEventOperation"}`, []string{traceID})

	// event index 1
	f(`{event:name="exception"}`, []string{traceID})
	f(`{event.exception.type="OutOfMemoryError"}`, []string{traceID})
	f(`{event.exception.type=~"OutOfMemory.*"}`, []string{traceID})

	// event index 0, so the wildcard must cover more than the last event
	f(`{event:name="first event"}`, []string{traceID})

	// link index 1
	f(`{link.link.kind="follows-from"}`, []string{traceID})
	f(`{link:traceID="`+linkedTraceID+`"}`, []string{traceID})
	f(`{link:spanID="`+linkedSpanID+`"}`, []string{traceID})

	// link index 0
	f(`{link:traceID="`+linkedTraceID0+`"}`, []string{traceID})

	f(`{event:name="no such event"}`, nil)
	f(`{event.exception.type="NoSuchError"}`, nil)

	// Each condition is matched on its own rather than pinned to one event, so this
	// matches even though the `first event` entry carries no attributes. Documented in
	// docs/victoriatraces/querying/README.md; pinned here so a change is deliberate.
	f(`{event:name="first event" && event.exception.type="OutOfMemoryError"}`, []string{traceID})

	// `= nil` reads across every event, so it is not the negation of `!= nil` on one
	// event: it holds only when NO event carries the attribute. The span below has an
	// event without `exception.type`, and `= nil` still does not match.
	f(`{event.absent.attr = nil}`, []string{traceID})
	f(`{event.exception.type = nil}`, nil)
	f(`{event.exception.type != nil}`, []string{traceID})
	f(`{event.absent.attr != nil}`, nil)
}
