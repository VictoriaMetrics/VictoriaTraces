package pb

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"reflect"
	"testing"
)

// TestUnmarshalExportTraceServiceRequest load same request samples in both JSON and Protobuf type
// and verify if the unmarshal result match.
func TestUnmarshalExportTraceServiceRequest(t *testing.T) {
	// load test data
	jsonPath := "./testdata/json"
	protobufPath := "./testdata/protobuf"
	jsonTestData, err := os.ReadDir(jsonPath)
	if err != nil {
		t.Fatalf("cannot read JSON test data: %s", err)
	}
	protobufTestData, err := os.ReadDir(protobufPath)
	if err != nil {
		t.Fatalf("cannot read protobuf test data: %s", err)
	}

	// verify test data
	if len(jsonTestData) == 0 || len(jsonTestData) != len(protobufTestData) {
		t.Fatalf("incorrect test request sample count, please verify.")
	}

	for i := range jsonTestData {
		if jsonTestData[i].Name() != protobufTestData[i].Name() {
			t.Fatalf("incorrect test request sample, please verify.")
		}
	}

	// do unmarshal
	for i := range jsonTestData {
		var reqInJSON ExportTraceServiceRequest
		var reqInProtobuf ExportTraceServiceRequest

		jsonBytes, err := os.ReadFile(path.Join(jsonPath, jsonTestData[i].Name()))
		if err != nil {
			t.Fatalf("cannot read %s: %s", jsonTestData[i].Name(), err)
		}
		protobufBytes, err := os.ReadFile(path.Join(protobufPath, protobufTestData[i].Name()))
		if err != nil {
			t.Fatalf("cannot read %s: %s", protobufTestData[i].Name(), err)
		}

		if err := json.Unmarshal(jsonBytes, &reqInJSON); err != nil {
			t.Fatalf("cannot unmarshal JSON %s: %s", jsonTestData[i].Name(), err)
		}
		if err := reqInProtobuf.UnmarshalProtobuf(protobufBytes); err != nil {
			t.Fatalf("cannot unmarshal protobuf %s: %s", protobufTestData[i].Name(), err)
		}

		// compare unmarshal result
		if !reflect.DeepEqual(reqInJSON, reqInProtobuf) {
			t.Fatalf("unmarshal result mismatch for %s", jsonTestData[i].Name())
		}
	}
}

func TestUnmarshalExportTraceServiceRequestNullItems(t *testing.T) {
	f := func(jsonString, jsonStringExpected string) {
		t.Helper()
		var req, reqExpected ExportTraceServiceRequest
		if err := json.Unmarshal([]byte(jsonString), &req); err != nil {
			t.Fatalf("unexpected error when unmarshaling %s: %s", jsonString, err)
		}
		if err := json.Unmarshal([]byte(jsonStringExpected), &reqExpected); err != nil {
			t.Fatalf("unexpected error when unmarshaling %s: %s", jsonStringExpected, err)
		}
		// null items must be treated as absent items, so the marshaled results must be identical.
		data := req.MarshalProtobuf(nil)
		dataExpected := reqExpected.MarshalProtobuf(nil)
		if string(data) != string(dataExpected) {
			t.Fatalf("unexpected result for %s; got %X; want %X", jsonString, data, dataExpected)
		}
	}

	f(`{"resourceSpans":[null]}`, `{}`)

	// null items in resource and scope
	f(`{
		"resourceSpans": [{
			"resource": {
				"attributes": [
					null,
					{"key": "a", "value": {"arrayValue": {"values": [null]}}},
					{"key": "b", "value": {"keyValueList": {"values": [null]}}}
				]
			},
			"scopeSpans": [
				null,
				{"scope": {"attributes": [null]}}
			]
		}]
	}`, `{
		"resourceSpans": [{
			"resource": {
				"attributes": [
					{"key": "a", "value": {"arrayValue": {}}},
					{"key": "b", "value": {"keyValueList": {}}}
				]
			},
			"scopeSpans": [
				{}
			]
		}]
	}`)

	// null items in spans, span attributes, links and events, while other items are kept
	f(`{
		"resourceSpans": [
			null,
			{
				"scopeSpans": [{
					"spans": [
						null,
						{
							"name": "foo",
							"traceId": "AB",
							"attributes": [null],
							"links": [null, {"attributes": [null]}],
							"events": [null, {"attributes": [null]}]
						},
						null
					]
				}]
			},
			null
		]
	}`, `{
		"resourceSpans": [{
			"scopeSpans": [{
				"spans": [{
					"name": "foo",
					"traceId": "ab",
					"links": [{}],
					"events": [{}]
				}]
			}]
		}]
	}`)
}

func TestUnmarshalExportTraceServiceRequestEquivalentForms(t *testing.T) {
	f := func(jsonString, jsonStringExpected string) {
		t.Helper()
		var req, reqExpected ExportTraceServiceRequest
		if err := json.Unmarshal([]byte(jsonString), &req); err != nil {
			t.Fatalf("unexpected error when unmarshaling %s: %s", jsonString, err)
		}
		if err := json.Unmarshal([]byte(jsonStringExpected), &reqExpected); err != nil {
			t.Fatalf("unexpected error when unmarshaling %s: %s", jsonStringExpected, err)
		}
		// equivalent forms must be decoded to the same request, so the marshaled results must be identical.
		data := req.MarshalProtobuf(nil)
		dataExpected := reqExpected.MarshalProtobuf(nil)
		if string(data) != string(dataExpected) {
			t.Fatalf("unexpected result for %s; got %X; want %X", jsonString, data, dataExpected)
		}
	}

	// timestamps as numbers are equivalent to timestamps as decimal strings
	f(`{
		"resourceSpans": [{
			"scopeSpans": [{
				"spans": [{
					"name": "foo",
					"startTimeUnixNano": 1757320933956646600,
					"endTimeUnixNano": 18446744073709551615,
					"events": [{"name": "bar", "timeUnixNano": 1757320933956646601}]
				}]
			}]
		}]
	}`, `{
		"resourceSpans": [{
			"scopeSpans": [{
				"spans": [{
					"name": "foo",
					"startTimeUnixNano": "1757320933956646600",
					"endTimeUnixNano": "18446744073709551615",
					"events": [{"name": "bar", "timeUnixNano": "1757320933956646601"}]
				}]
			}]
		}]
	}`)

	// null timestamps are equivalent to absent timestamps
	f(`{
		"resourceSpans": [{
			"scopeSpans": [{
				"spans": [{
					"name": "foo",
					"startTimeUnixNano": null,
					"endTimeUnixNano": null,
					"events": [{"name": "bar", "timeUnixNano": null}]
				}]
			}]
		}]
	}`, `{
		"resourceSpans": [{
			"scopeSpans": [{
				"spans": [{
					"name": "foo",
					"events": [{"name": "bar"}]
				}]
			}]
		}]
	}`)

	// zero timestamps are equivalent to absent timestamps
	f(`{
		"resourceSpans": [{
			"scopeSpans": [{
				"spans": [{
					"name": "foo",
					"startTimeUnixNano": "0",
					"endTimeUnixNano": 0,
					"events": [{"name": "bar", "timeUnixNano": "0"}]
				}]
			}]
		}]
	}`, `{
		"resourceSpans": [{
			"scopeSpans": [{
				"spans": [{
					"name": "foo",
					"events": [{"name": "bar"}]
				}]
			}]
		}]
	}`)

	// uppercase hex IDs are equivalent to lowercase hex IDs
	f(`{
		"resourceSpans": [{
			"scopeSpans": [{
				"spans": [{
					"name": "foo",
					"traceId": "5B8EFFF798038103D269B633813FC60C",
					"spanId": "EEE19B7EC3C1B174",
					"parentSpanId": "EEE19B7EC3C1B173",
					"links": [{"traceId": "5B8EFFF798038103D269B633813FC60D", "spanId": "EEE19B7EC3C1B175"}]
				}]
			}]
		}]
	}`, `{
		"resourceSpans": [{
			"scopeSpans": [{
				"spans": [{
					"name": "foo",
					"traceId": "5b8efff798038103d269b633813fc60c",
					"spanId": "eee19b7ec3c1b174",
					"parentSpanId": "eee19b7ec3c1b173",
					"links": [{"traceId": "5b8efff798038103d269b633813fc60d", "spanId": "eee19b7ec3c1b175"}]
				}]
			}]
		}]
	}`)
}

func TestUnmarshalExportTraceServiceRequestInvalidTimestamps(t *testing.T) {
	f := func(jsonString string) {
		t.Helper()
		var req ExportTraceServiceRequest
		if err := json.Unmarshal([]byte(jsonString), &req); err == nil {
			t.Fatalf("expecting non-nil error when unmarshaling %s", jsonString)
		}
	}

	f(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"startTimeUnixNano":-1}]}]}]}`)
	f(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"startTimeUnixNano":"-1"}]}]}]}`)
	f(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"startTimeUnixNano":1.5}]}]}]}`)
	f(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"startTimeUnixNano":""}]}]}]}`)
	f(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"startTimeUnixNano":"abc"}]}]}]}`)
	f(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"startTimeUnixNano":true}]}]}]}`)
	f(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"startTimeUnixNano":{}}]}]}]}`)
	f(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"startTimeUnixNano":[]}]}]}]}`)
	f(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"startTimeUnixNano":18446744073709551616}]}]}]}`)
	f(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"endTimeUnixNano":"18446744073709551616"}]}]}]}`)
	f(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"events":[{"timeUnixNano":"abc"}]}]}]}]}`)
}

// TestMarshalExportTraceServiceRequestJSONRoundTrip verifies that JSON produced by json.Marshal
// is decoded to the same request as the original protobuf sample.
func TestMarshalExportTraceServiceRequestJSONRoundTrip(t *testing.T) {
	jsonPath := "./testdata/json"
	protobufPath := "./testdata/protobuf"
	jsonTestData, err := os.ReadDir(jsonPath)
	if err != nil {
		t.Fatalf("cannot read JSON test data: %s", err)
	}

	for _, de := range jsonTestData {
		var req, reqRoundTrip, reqInProtobuf ExportTraceServiceRequest

		jsonBytes, err := os.ReadFile(path.Join(jsonPath, de.Name()))
		if err != nil {
			t.Fatalf("cannot read %s: %s", de.Name(), err)
		}
		protobufBytes, err := os.ReadFile(path.Join(protobufPath, de.Name()))
		if err != nil {
			t.Fatalf("cannot read %s: %s", de.Name(), err)
		}

		if err := json.Unmarshal(jsonBytes, &req); err != nil {
			t.Fatalf("cannot unmarshal JSON %s: %s", de.Name(), err)
		}
		data, err := json.Marshal(&req)
		if err != nil {
			t.Fatalf("cannot marshal JSON %s: %s", de.Name(), err)
		}
		if err := json.Unmarshal(data, &reqRoundTrip); err != nil {
			t.Fatalf("cannot unmarshal marshaled JSON %s: %s", de.Name(), err)
		}
		if err := reqInProtobuf.UnmarshalProtobuf(protobufBytes); err != nil {
			t.Fatalf("cannot unmarshal protobuf %s: %s", de.Name(), err)
		}

		if !reflect.DeepEqual(reqRoundTrip, reqInProtobuf) {
			t.Fatalf("round-trip result mismatch for %s\nmarshaled JSON: %s", de.Name(), data)
		}
	}
}

// TestMarshalJSON verifies that JSON in the canonical form is marshaled back unchanged after unmarshaling.
//
// The canonical form is the one produced by json.Marshal: all the fields are present in the struct field order,
// 64-bit integers are decimal strings, IDs are lowercase hex and AnyValue contains only the set field.
func TestMarshalJSON(t *testing.T) {
	f := func(dst any, jsonString string) {
		t.Helper()
		if err := json.Unmarshal([]byte(jsonString), dst); err != nil {
			t.Fatalf("unexpected error when unmarshaling %s: %s", jsonString, err)
		}
		data, err := json.Marshal(dst)
		if err != nil {
			t.Fatalf("unexpected error when marshaling %s: %s", jsonString, err)
		}
		// jsonString may be formatted for readability, so compact it before the comparison
		var bb bytes.Buffer
		if err := json.Compact(&bb, []byte(jsonString)); err != nil {
			t.Fatalf("cannot compact %s: %s", jsonString, err)
		}
		if string(data) != bb.String() {
			t.Fatalf("unexpected marshaled JSON\ngot\n%s\nwant\n%s", data, bb.String())
		}
	}

	// AnyValue with every kind of value
	f(&AnyValue{}, `{}`)
	f(&AnyValue{}, `{"stringValue":"x"}`)
	f(&AnyValue{}, `{"stringValue":""}`)
	f(&AnyValue{}, `{"boolValue":true}`)
	f(&AnyValue{}, `{"boolValue":false}`)
	f(&AnyValue{}, `{"intValue":"-9223372036854775808"}`)
	f(&AnyValue{}, `{"intValue":"0"}`)
	f(&AnyValue{}, `{"doubleValue":1.5}`)
	f(&AnyValue{}, `{"bytesValue":"AQI="}`)
	f(&AnyValue{}, `{"arrayValue":{"values":[{"intValue":"1"},{"stringValue":"a"}]}}`)
	f(&AnyValue{}, `{"keyValueList":{"values":[{"key":"k","value":{"boolValue":false}}]}}`)

	// request with all the fields set
	f(&ExportTraceServiceRequest{}, `{
		"resourceSpans": [{
			"resource": {"attributes": [{"key": "service.name", "value": {"stringValue": "svc"}}]},
			"scopeSpans": [{
				"scope": {"name": "lib", "version": "1.0", "attributes": [{"key": "a", "value": {"intValue": "1"}}], "droppedAttributesCount": 1},
				"spans": [{
					"traceId": "0123456789abcdef0123456789abcdef",
					"spanId": "0123456789abcdef",
					"traceState": "k=v",
					"parentSpanId": "fedcba9876543210",
					"flags": 1,
					"name": "GET /",
					"kind": 2,
					"attributes": [{"key": "b", "value": {"doubleValue": 1.5}}],
					"droppedAttributesCount": 2,
					"events": [{
						"name": "event",
						"attributes": [{"key": "c", "value": {"bytesValue": "AQI="}}],
						"droppedAttributesCount": 3,
						"timeUnixNano": "1712345678500000000"
					}],
					"droppedEventsCount": 4,
					"links": [{
						"traceId": "fedcba9876543210fedcba9876543210",
						"spanId": "fedcba9876543210",
						"traceState": "k=v",
						"attributes": [{"key": "d", "value": {"boolValue": true}}],
						"droppedAttributesCount": 5,
						"flags": 1
					}],
					"droppedLinksCount": 6,
					"status": {"message": "error", "code": 2},
					"startTimeUnixNano": "1712345678000000000",
					"endTimeUnixNano": "18446744073709551615"
				}],
				"schemaUrl": "https://opentelemetry.io/schemas/1.0.0"
			}],
			"schemaUrl": "https://opentelemetry.io/schemas/1.0.0"
		}]
	}`)
}

// TestMarshalJSONNormalized verifies the cases where the marshaled JSON differs from the unmarshaled one.
//
// OTLP/JSON accepts multiple forms of the same value, while json.Marshal always produces the canonical form.
// See TestMarshalJSON for the canonical form.
func TestMarshalJSONNormalized(t *testing.T) {
	f := func(dst any, jsonString, jsonStringExpected string) {
		t.Helper()
		if err := json.Unmarshal([]byte(jsonString), dst); err != nil {
			t.Fatalf("unexpected error when unmarshaling %s: %s", jsonString, err)
		}
		data, err := json.Marshal(dst)
		if err != nil {
			t.Fatalf("unexpected error when marshaling %s: %s", jsonString, err)
		}
		// jsonStringExpected may be formatted for readability, so compact it before the comparison
		var bb bytes.Buffer
		if err := json.Compact(&bb, []byte(jsonStringExpected)); err != nil {
			t.Fatalf("cannot compact %s: %s", jsonStringExpected, err)
		}
		if string(data) != bb.String() {
			t.Fatalf("unexpected marshaled JSON\ngot\n%s\nwant\n%s", data, bb.String())
		}
	}

	// int64 JSON numbers are encoded as decimal strings
	f(&AnyValue{}, `{"intValue":-9223372036854775808}`, `{"intValue":"-9223372036854775808"}`)

	// null values in AnyValue are treated as absent
	f(&AnyValue{}, `{"stringValue":null,"intValue":null}`, `{}`)

	// fixed64 JSON numbers are encoded as decimal strings
	f(&SpanEvent{}, `{"timeUnixNano":3}`, `{"name":"","attributes":null,"droppedAttributesCount":0,"timeUnixNano":"3"}`)

	// null and absent timestamps are encoded as "0"
	f(&SpanEvent{}, `{"timeUnixNano":null}`, `{"name":"","attributes":null,"droppedAttributesCount":0,"timeUnixNano":"0"}`)
	f(&SpanEvent{}, `{}`, `{"name":"","attributes":null,"droppedAttributesCount":0,"timeUnixNano":"0"}`)

	// IDs are encoded as lowercase hex
	f(&SpanLink{}, `{"traceId":"ABCDEF","spanId":"0A"}`, `{"traceId":"abcdef","spanId":"0a","traceState":"","attributes":null,"droppedAttributesCount":0,"flags":0}`)

	// keys are matched case-insensitively and encoded in camelCase
	f(&SpanLink{}, `{"TRACEID":"ab","SpanID":"cd"}`, `{"traceId":"ab","spanId":"cd","traceState":"","attributes":null,"droppedAttributesCount":0,"flags":0}`)

	// absent fields are encoded with zero values, IDs are lowercased and timestamps are encoded as decimal strings
	f(&ExportTraceServiceRequest{}, `{"resourceSpans":[{"scopeSpans":[{"spans":[{
		"traceId":"ABCDEF","spanId":"0A","parentSpanID":"0B",
		"startTimeUnixNano":1,"endTimeUnixNano":null
	}]}]}]}`, `{"resourceSpans":[{"resource":{"attributes":null},"scopeSpans":[{"scope":{"name":"","version":"","attributes":null,"droppedAttributesCount":0},"spans":[{
		"traceId":"abcdef","spanId":"0a","traceState":"","parentSpanId":"0b","flags":0,"name":"","kind":0,
		"attributes":null,"droppedAttributesCount":0,"events":null,"droppedEventsCount":0,"links":null,"droppedLinksCount":0,
		"status":{"message":"","code":0},"startTimeUnixNano":"1","endTimeUnixNano":"0"
	}],"schemaUrl":""}],"schemaUrl":""}]}`)
}
