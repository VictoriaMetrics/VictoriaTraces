package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/query"
	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// TestSingleSearchTracesNonUTCTimezone verifies that Jaeger and Tempo trace search return correct traces
// when VictoriaTraces runs in a time zone other than UTC.
//
// See https://github.com/VictoriaMetrics/VictoriaTraces/issues/260
func TestSingleSearchTracesNonUTCTimezone(t *testing.T) {
	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartVtsingle("vtsingle", []string{
		"-storageDataPath=" + tc.Dir() + "/vtsingle",
		"-retentionPeriod=100y",
	}, []string{"TZ=America/New_York"}...)

	serviceName := "testSearchTimezoneService"
	spanTime := time.Now().Add(-2 * time.Hour)
	req := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{
				Resource: otelpb.Resource{
					Attributes: []*otelpb.KeyValue{
						{Key: "service.name", Value: &otelpb.AnyValue{StringValue: &serviceName}},
					},
				},
				ScopeSpans: []*otelpb.ScopeSpans{
					{
						Spans: []*otelpb.Span{
							{
								TraceID:           "5b8efff798038103d269b633813fc60c",
								SpanID:            "eee19b7ec3c1b174",
								Name:              "testSearchTimezoneSpan",
								StartTimeUnixNano: uint64(spanTime.UnixNano()),
								EndTimeUnixNano:   uint64(spanTime.Add(time.Millisecond).UnixNano()),
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

	tc.Assert(&at.AssertOptions{
		Msg: "unexpected number of traces from /select/jaeger/api/traces",
		Got: func() any {
			res := sut.JaegerAPITraces(t, at.JaegerQueryParam{
				TraceQueryParam: query.TraceQueryParam{
					ServiceName:  serviceName,
					StartTimeMin: startTime,
					StartTimeMax: endTime,
				},
			}, at.QueryOpts{})
			return len(res.Data)
		},
		Want: 1,
	})

	tc.Assert(&at.AssertOptions{
		Msg: "unexpected number of traces from /select/tempo/api/search",
		Got: func() any {
			res := sut.TempoAPISearch(t, fmt.Sprintf(`{resource.service.name=%q}`, serviceName), startTime, endTime)
			return len(res.Traces)
		},
		Want: 1,
	})
}
