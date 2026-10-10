package remotewrite

import (
	"testing"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

func TestConcatenatedExportTraceServiceRequests(t *testing.T) {
	newRequest := func(serviceName string) *otelpb.ExportTraceServiceRequest {
		return &otelpb.ExportTraceServiceRequest{
			ResourceSpans: []*otelpb.ResourceSpans{
				{
					Resource: otelpb.Resource{
						Attributes: []*otelpb.KeyValue{
							{Key: "service.name", Value: &otelpb.AnyValue{StringValue: &serviceName}},
						},
					},
				},
			},
		}
	}

	var wr writeRequest
	_, _ = wr.pendingData.Write(newRequest("svc1").MarshalProtobuf(nil))
	_, _ = wr.pendingData.Write(newRequest("svc2").MarshalProtobuf(nil))

	var req otelpb.ExportTraceServiceRequest
	if err := req.UnmarshalProtobuf(wr.pendingData.B); err != nil {
		t.Fatalf("cannot unmarshal concatenated requests: %s", err)
	}
	if len(req.ResourceSpans) != 2 {
		t.Fatalf("unexpected number of resource spans; got %d; want 2", len(req.ResourceSpans))
	}
	for i, want := range []string{"svc1", "svc2"} {
		got := *req.ResourceSpans[i].Resource.Attributes[0].Value.StringValue
		if got != want {
			t.Fatalf("unexpected service.name at resource spans #%d; got %q; want %q", i, got, want)
		}
	}
}
