package tests

import (
	"testing"
	"time"

	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// TestSingleTempoSearchTagsByScope verifies that /select/tempo/api/v2/search/tags
// returns the attribute names of the requested scope.
func TestSingleTempoSearchTagsByScope(t *testing.T) {
	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartVtsingle("vtsingle", []string{
		"-storageDataPath=" + tc.Dir() + "/vtsingle",
		"-retentionPeriod=100y",
	})

	serviceName := "testSearchTagsService"
	clusterName := "testCluster"
	libLang := "go"
	httpMethod := "GET"
	spanTime := time.Now().Add(-time.Hour)
	req := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{
				Resource: otelpb.Resource{
					Attributes: []*otelpb.KeyValue{
						{Key: "service.name", Value: &otelpb.AnyValue{StringValue: &serviceName}},
						{Key: "k8s.cluster.name", Value: &otelpb.AnyValue{StringValue: &clusterName}},
					},
				},
				ScopeSpans: []*otelpb.ScopeSpans{
					{
						Scope: otelpb.InstrumentationScope{
							Name: "testScope",
							Attributes: []*otelpb.KeyValue{
								{Key: "library.language", Value: &otelpb.AnyValue{StringValue: &libLang}},
							},
						},
						Spans: []*otelpb.Span{
							{
								TraceID:           "7c2e8d1a4b9f0e3d5a6b7c8d9e0f1a2b",
								SpanID:            "1a2b3c4d5e6f7a8b",
								Name:              "testSearchTagsSpan",
								StartTimeUnixNano: uint64(spanTime.UnixNano()),
								EndTimeUnixNano:   uint64(spanTime.Add(time.Millisecond).UnixNano()),
								Attributes: []*otelpb.KeyValue{
									{Key: "http.method", Value: &otelpb.AnyValue{StringValue: &httpMethod}},
								},
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

	resourceTags := []string{"k8s.cluster.name", "service.name"}
	spanTags := []string{"http.method"}
	scopeTags := []string{"library.language"}

	f := func(scope string, wantResource, wantSpan, wantScope []string) {
		t.Helper()
		tc.Assert(&at.AssertOptions{
			Msg: "unexpected tags for scope=" + scope + " from /select/tempo/api/v2/search/tags",
			Got: func() any {
				res := sut.TempoAPISearchTags(t, scope, startTime, endTime)
				return [][]string{res.Tags("resource"), res.Tags("span"), res.Tags("instrumentation")}
			},
			Want: [][]string{wantResource, wantSpan, wantScope},
		})
	}

	f("resource", resourceTags, []string{}, []string{})
	f("span", []string{}, spanTags, []string{})
	f("instrumentation", []string{}, []string{}, scopeTags)
	f("all", resourceTags, spanTags, scopeTags)
}
