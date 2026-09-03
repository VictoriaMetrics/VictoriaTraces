package tests

import (
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// TestSingleDeleteRunTaskRequiresPost checks that /delete/run_task drops GET requests
// and still works over POST.
//
// A GET on this path lets a server-side request forgery destroy the stored spans
// with a plain URL fetch, because no request body is needed.
// See https://github.com/VictoriaMetrics/VictoriaTraces/issues/225
func TestSingleDeleteRunTaskRequiresPost(t *testing.T) {
	os.RemoveAll(t.Name())

	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartVtsingle("vtsingle", []string{
		"-storageDataPath=" + tc.Dir() + "/vtsingle",
		"-retentionPeriod=100y",
		"-delete.enable",
	})

	serviceName := "testDeleteMethodService"
	spanTime := time.Now()
	req := &otelpb.ExportTraceServiceRequest{
		ResourceSpans: []*otelpb.ResourceSpans{
			{
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
								TraceID:           "aa5886e99fffef35a847cb2d493fde20",
								SpanID:            "0123456789abcdef",
								Name:              "deleteMethodOperation",
								StartTimeUnixNano: uint64(spanTime.UnixNano()),
								EndTimeUnixNano:   uint64(spanTime.Add(time.Second).UnixNano()),
							},
						},
					},
				},
			},
		},
	}
	sut.OTLPHTTPExportTraces(t, req, at.QueryOpts{})
	sut.ForceFlush(t)
	time.Sleep(2 * time.Second) // index will be created after -insert.traceMaxDuration (2s in integration test)

	tc.Assert(&at.AssertOptions{
		Msg: "the span was not ingested, so the delete checks below would prove nothing",
		Got: func() any {
			return sut.JaegerAPIServices(t, at.QueryOpts{}).Data
		},
		Want:    []string{serviceName},
		Retries: 10,
		Period:  time.Second,
	})

	deleteURL := "http://" + sut.HTTPAddr() + "/delete/run_task"

	res, statusCode := tc.Client().Get(t, deleteURL+"?filter=*")
	if statusCode != 405 {
		t.Fatalf("unexpected status code for GET %s; got %d, want 405; response: %q", deleteURL, statusCode, res)
	}
	if !strings.Contains(res, "Only POST method is allowed") {
		t.Fatalf("unexpected response body for GET %s; got %q", deleteURL, res)
	}

	// The rejected GET must not have started a delete task.
	if got := sut.JaegerAPIServices(t, at.QueryOpts{}).Data; len(got) != 1 || got[0] != serviceName {
		t.Fatalf("the rejected GET removed data; services after the GET: %v", got)
	}

	res, statusCode = tc.Client().PostForm(t, deleteURL, url.Values{"filter": []string{"*"}})
	if statusCode != 200 {
		t.Fatalf("unexpected status code for POST %s; got %d, want 200; response: %q", deleteURL, statusCode, res)
	}
	if !strings.Contains(res, `"task_id"`) {
		t.Fatalf("missing task_id in response to POST %s; got %q", deleteURL, res)
	}

	// The delete task runs in the background, so wait for the spans to go away.
	tc.Assert(&at.AssertOptions{
		Msg: "the spans were not removed by the accepted POST",
		Got: func() any {
			return len(sut.JaegerAPIServices(t, at.QueryOpts{}).Data)
		},
		Want:    0,
		Retries: 30,
		Period:  time.Second,
	})
}
