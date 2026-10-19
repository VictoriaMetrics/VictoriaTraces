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
	jsonPath := "./lib/protoparser/opentelemetry/pb/testdata/json"
	protobufPath := "./lib/protoparser/opentelemetry/pb/testdata/protobuf"
	jsonTestData, _ := os.ReadDir(jsonPath)
	protobufTestData, _ := os.ReadDir(protobufPath)

	// verify test data
	if len(jsonTestData) != len(protobufTestData) {
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

		jsonBytes, _ := os.ReadFile(path.Join(jsonPath, jsonTestData[i].Name()))
		protobufBytes, _ := os.ReadFile(path.Join(protobufPath, protobufTestData[i].Name()))

		_ = reqInJSON.UnmarshalJSONCustom(jsonBytes)
		_ = reqInProtobuf.UnmarshalProtobuf(protobufBytes)

		// compare unmarshal result
		if !reflect.DeepEqual(reqInJSON, reqInProtobuf) {
			t.Fatalf("unmarshal result mismatch")
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
