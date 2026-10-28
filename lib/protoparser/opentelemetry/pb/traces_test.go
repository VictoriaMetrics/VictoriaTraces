package pb

import (
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

		if err := reqInJSON.UnmarshalJSONCustom(jsonBytes); err != nil {
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
		if err := req.UnmarshalJSONCustom([]byte(jsonString)); err != nil {
			t.Fatalf("unexpected error when unmarshaling %s: %s", jsonString, err)
		}
		if err := reqExpected.UnmarshalJSONCustom([]byte(jsonStringExpected)); err != nil {
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
		if err := req.UnmarshalJSONCustom([]byte(jsonString)); err != nil {
			t.Fatalf("unexpected error when unmarshaling %s: %s", jsonString, err)
		}
		if err := reqExpected.UnmarshalJSONCustom([]byte(jsonStringExpected)); err != nil {
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
		if err := req.UnmarshalJSONCustom([]byte(jsonString)); err == nil {
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
