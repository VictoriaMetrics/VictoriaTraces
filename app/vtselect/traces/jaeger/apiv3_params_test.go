package jaeger

import (
	"reflect"
	"testing"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// TestToStorageAttributeFilter pins the tag conversion, which is shared by the v1 `tags` param
// and the v3 `query.attributes` param. A change here changes both API versions.
func TestToStorageAttributeFilter(t *testing.T) {
	f := func(attributes, expected map[string]string) {
		t.Helper()
		result := toStorageAttributeFilter(attributes)
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("unexpected filter;\ngot\n%v\nwant\n%v", result, expected)
		}
	}

	f(nil, map[string]string{})

	// a plain tag becomes a span attribute.
	f(map[string]string{"foo": "bar"}, map[string]string{
		otelpb.SpanAttrPrefixField + "foo": "bar",
	})

	// resource and scope attributes are already stored under their own prefix.
	f(map[string]string{
		otelpb.ResourceAttrPrefix + "os.type":         "linux",
		otelpb.InstrumentationScopeAttrPrefix + "foo": "bar",
	}, map[string]string{
		otelpb.ResourceAttrPrefix + "os.type":         "linux",
		otelpb.InstrumentationScopeAttrPrefix + "foo": "bar",
	})

	// `error` and `span.kind` map both the name and the value.
	f(map[string]string{"error": "true"}, map[string]string{
		otelpb.StatusCodeField: "2",
	})
	f(map[string]string{"span.kind": "client"}, map[string]string{
		otelpb.KindField: "3",
	})

	// a name-only mapping keeps the value as is.
	f(map[string]string{"otel.status_description": "boom"}, map[string]string{
		otelpb.StatusMessageField: "boom",
	})
}
