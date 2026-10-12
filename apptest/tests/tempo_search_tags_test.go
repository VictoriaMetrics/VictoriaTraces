package tests

import (
	"testing"
	"time"

	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// TestSingleTempoSearchTagsScope verifies that Tempo /api/v2/search/tags returns
// the attribute names of the requested scope.
//
// See https://github.com/VictoriaMetrics/VictoriaTraces/issues/268
func TestSingleTempoSearchTagsScope(t *testing.T) {
	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartVtsingle("vtsingle", []string{
		"-storageDataPath=" + tc.Dir() + "/vtsingle",
		"-retentionPeriod=100y",
	})

	strValue := func(s string) *otelpb.AnyValue { return &otelpb.AnyValue{StringValue: &s} }
	spanTime := time.Now().Add(-time.Hour)
	req := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{
				Resource: otelpb.Resource{
					Attributes: []*otelpb.KeyValue{
						{Key: "service.name", Value: strValue("testSearchTagsService")},
						{Key: "istio.cluster_id", Value: strValue("c1")},
					},
				},
				ScopeSpans: []*otelpb.ScopeSpans{
					{
						Scope: otelpb.InstrumentationScope{
							Name:       "testScope",
							Attributes: []*otelpb.KeyValue{{Key: "scope.attr", Value: strValue("v")}},
						},
						Spans: []*otelpb.Span{
							{
								TraceID:           "1b8efff798038103d269b633813fc60c",
								SpanID:            "aee19b7ec3c1b174",
								Name:              "testSearchTagsSpan",
								StartTimeUnixNano: uint64(spanTime.UnixNano()),
								EndTimeUnixNano:   uint64(spanTime.Add(time.Millisecond).UnixNano()),
								Attributes:        []*otelpb.KeyValue{{Key: "http.method", Value: strValue("GET")}},
							},
						},
					},
				},
			},
		},
	}
	sut.OTLPHTTPExportTraces(t, req, at.QueryOpts{})
	sut.ForceFlush(t)

	startTime := spanTime.Add(-time.Hour)
	endTime := time.Now()

	f := func(scope string, want []string) {
		t.Helper()

		tc.Assert(&at.AssertOptions{
			Msg: "unexpected tags for scope=" + scope + " from /select/tempo/api/v2/search/tags",
			Got: func() any {
				return sut.TempoAPISearchTags(t, scope, startTime, endTime).TagsOf(scope)
			},
			Want: want,
		})
	}

	f("resource", []string{"istio.cluster_id", "service.name"})
	f("span", []string{"http.method"})
	f("instrumentation", []string{"scope.attr"})
}
