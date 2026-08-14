package tests

import (
	"os"
	"strings"
	"testing"
	"time"

	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// TestSingleExtraFiltersJaegerAndTempo checks that the Jaeger and Tempo APIs apply
// the extra_filters query arg.
//
// vmauth injects this arg to keep a user inside its own data, so an API which drops it
// shows that user the data of others.
// See https://github.com/VictoriaMetrics/VictoriaTraces/issues/178
func TestSingleExtraFiltersJaegerAndTempo(t *testing.T) {
	os.RemoveAll(t.Name())

	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartVtsingle("vtsingle", []string{
		"-storageDataPath=" + tc.Dir() + "/vtsingle",
		"-retentionPeriod=100y",
	})

	spanTime := time.Now()
	req := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			newExtraFiltersResourceSpans("alpha", "0123456789abcde1", "aa5886e99fffef35a847cb2d493fde10", spanTime),
			newExtraFiltersResourceSpans("beta", "0123456789abcde2", "aa5886e99fffef35a847cb2d493fde20", spanTime),
		},
	}
	sut.OTLPHTTPExportTraces(t, req, at.QueryOpts{})
	sut.ForceFlush(t)
	time.Sleep(2 * time.Second) // index will be created after -insert.traceMaxDuration (2s in integration test)

	onlyAlpha := at.QueryOpts{ExtraFilters: []string{`{"resource_attr:service.name":"alpha"}`}}

	// without the filter both services are visible.
	tc.Assert(&at.AssertOptions{
		Msg: "unexpected /select/jaeger/api/services response without extra_filters",
		Got: func() any {
			return sut.JaegerAPIServices(t, at.QueryOpts{}).Data
		},
		Want:    []string{"alpha", "beta"},
		Retries: 10,
		Period:  time.Second,
	})

	// with the filter only the allowed service is visible.
	tc.Assert(&at.AssertOptions{
		Msg: "extra_filters is ignored by /select/jaeger/api/services",
		Got: func() any {
			return sut.JaegerAPIServices(t, onlyAlpha).Data
		},
		Want:    []string{"alpha"},
		Retries: 10,
		Period:  time.Second,
	})

	// the operations of a service outside the filter must not be returned.
	tc.Assert(&at.AssertOptions{
		Msg: "extra_filters is ignored by /select/jaeger/api/services/<name>/operations",
		Got: func() any {
			return len(sut.JaegerAPIOperations(t, "beta", onlyAlpha).Data)
		},
		Want:    0,
		Retries: 10,
		Period:  time.Second,
	})

	// the Tempo API shares the same query params, so it must filter too.
	tc.Assert(&at.AssertOptions{
		Msg: "extra_filters is ignored by the Tempo tag values API",
		Got: func() any {
			res := sut.TempoAPITagValues(t, "resource.service.name", onlyAlpha)
			return []bool{strings.Contains(res, `"alpha"`), strings.Contains(res, `"beta"`)}
		},
		Want:    []bool{true, false},
		Retries: 10,
		Period:  time.Second,
	})
}

func newExtraFiltersResourceSpans(serviceName, spanID, traceID string, spanTime time.Time) *otelpb.ResourceSpans {
	return &otelpb.ResourceSpans{
		Resource: otelpb.Resource{
			Attributes: []*otelpb.KeyValue{
				{
					Key:   "service.name",
					Value: &otelpb.AnyValue{StringValue: &serviceName},
				},
			},
		},
		ScopeSpans: []*otelpb.ScopeSpans{
			{
				Spans: []*otelpb.Span{
					{
						TraceID:           traceID,
						SpanID:            spanID,
						Name:              "op-" + serviceName,
						StartTimeUnixNano: uint64(spanTime.UnixNano()),
						EndTimeUnixNano:   uint64(spanTime.Add(time.Second).UnixNano()),
					},
				},
			},
		},
	}
}
