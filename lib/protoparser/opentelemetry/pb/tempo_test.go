package pb

import (
	"os"
	"path"
	"reflect"
	"testing"
)

// TestTempoTraceMarshalProtobuf verifies that the bare Trace message produced by the
// Tempo /api/traces/<trace_id> (v1) API round-trips back to the same resource spans.
//
// TempoTrace shares its wire shape with ExportTraceServiceRequest (field 1 = repeated
// ResourceSpans), so the v1 output can be re-read through ExportTraceServiceRequest.
func TestTempoTraceMarshalProtobuf(t *testing.T) {
	protobufPath := "./testdata/protobuf"
	entries, err := os.ReadDir(protobufPath)
	if err != nil {
		t.Fatalf("cannot read testdata: %s", err)
	}

	for _, e := range entries {
		src, err := os.ReadFile(path.Join(protobufPath, e.Name()))
		if err != nil {
			t.Fatalf("cannot read %s: %s", e.Name(), err)
		}

		var req ExportTraceServiceRequest
		if err := req.UnmarshalProtobuf(src); err != nil {
			t.Fatalf("cannot unmarshal %s: %s", e.Name(), err)
		}

		trace := TempoTrace{ResourceSpan: req.ResourceSpans}
		marshaled := trace.MarshalProtobuf(nil)

		var got ExportTraceServiceRequest
		if err := got.UnmarshalProtobuf(marshaled); err != nil {
			t.Fatalf("cannot unmarshal v1 trace bytes for %s: %s", e.Name(), err)
		}

		if !reflect.DeepEqual(got.ResourceSpans, req.ResourceSpans) {
			t.Fatalf("resource spans mismatch after v1 marshal round-trip for %s", e.Name())
		}
	}
}
