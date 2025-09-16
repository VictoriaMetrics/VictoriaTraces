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
