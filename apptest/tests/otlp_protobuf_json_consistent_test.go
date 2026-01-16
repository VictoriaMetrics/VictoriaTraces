package tests

import (
	"encoding/json"
	"os"
	"path"
	"sort"
	"strconv"
	"testing"
	"time"

	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
)

// TestSingleOTLPIngestionProtobufJSONConsistency test protobuf and JSON data ingestion of `/insert/opentelemetry/v1/traces` API,
// to verify that they can parse and ingest data in the exact same way.
func TestSingleOTLPIngestionProtobufJSONConsistency(t *testing.T) {
	os.RemoveAll(t.Name())

	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut1 := tc.MustStartVtsingle("vtsingle-1", []string{
		"-storageDataPath=" + tc.Dir() + "/vtsingle-1",
		"-retentionPeriod=100y",
	})
	sut2 := tc.MustStartVtsingle("vtsingle-2", []string{
		"-storageDataPath=" + tc.Dir() + "/vtsingle-2",
		"-retentionPeriod=100y",
	})

	doOTLPHTTPProtobufIngestion(tc, sut1)
	doOTLPHTTPJSONIngestion(tc, sut2)

	testProtobufJSONConsistency(tc, sut1, sut2)
}

func doOTLPHTTPProtobufIngestion(tc *at.TestCase, sut at.VictoriaTracesWriteQuerier) {
	t := tc.T()

	// read test data
	protobufPath := "./testdata/protobuf"
	protobufTestData, _ := os.ReadDir(protobufPath)

	for i := range protobufTestData {
		protobufBytes, _ := os.ReadFile(path.Join(protobufPath, protobufTestData[i].Name()))
		// ingest data via /insert/opentelemetry/v1/traces
		sut.OTLPHTTPExportRawTraces(t, protobufBytes, at.QueryOpts{
			HTTPHeaders: map[string]string{
				"Content-Type": "application/x-protobuf",
			},
		})
	}

	sut.ForceFlush(t)
}

func doOTLPHTTPJSONIngestion(tc *at.TestCase, sut at.VictoriaTracesWriteQuerier) {
	t := tc.T()

	// read test data
	jsonPath := "./testdata/json"
	jsonTestData, _ := os.ReadDir(jsonPath)

	for i := range jsonTestData {
		jsonBytes, _ := os.ReadFile(path.Join(jsonPath, jsonTestData[i].Name()))
		// ingest data via /insert/opentelemetry/v1/traces
		sut.OTLPHTTPExportRawTraces(t, jsonBytes, at.QueryOpts{
			HTTPHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		})
	}

	sut.ForceFlush(t)
}

func testProtobufJSONConsistency(tc *at.TestCase, sut1, sut2 at.VictoriaTracesWriteQuerier) {
	t := tc.T()

	query := "*"
	opts := at.QueryOpts{
		Start: strconv.FormatInt(time.Now().Add(-20*365*24*time.Hour).UnixNano(), 10),
		End:   strconv.FormatInt(time.Now().UnixNano(), 10),
		Limit: "10000",
	}
	response1 := sut1.LogsQLQuery(t, query, opts)
	response2 := sut2.LogsQLQuery(t, query, opts)
	sort.Slice(response1.LogLines, func(i, j int) bool {
		m1, m2 := make(map[string]string), make(map[string]string)

		_ = json.Unmarshal([]byte(response1.LogLines[i]), &m1)
		_ = json.Unmarshal([]byte(response1.LogLines[j]), &m2)

		return m1["span_id"]+m1["trace_id_idx"] < m2["span_id"]+m2["trace_id_idx"]
	})
	sort.Slice(response2.LogLines, func(i, j int) bool {
		m1, m2 := make(map[string]string), make(map[string]string)

		_ = json.Unmarshal([]byte(response2.LogLines[i]), &m1)
		_ = json.Unmarshal([]byte(response2.LogLines[j]), &m2)

		return m1["span_id"]+m1["trace_id_idx"] < m2["span_id"]+m2["trace_id_idx"]
	})

	// TODO: Fix redundant fields that introduced by JSON unmarshaling and default value of the OTLP struct.
	//
	// in previous version, fields that actually not exist in the protobuf/JSON request data could be added to the log fields,
	// because some fields of the ExportTraceServiceRequest struct is not pointer, and carry default value, such as 0.
	//
	// while pull request #93 will skip the intermediate step ([]byte to `struct`) and turn []byte into fields directly,
	// it fix this issue for protobuf data ingestion. the issue remain the same for JSON, tho.
	//
	// the expect result of the ingestion will be: fields should not be ingested if they're not exist in protobuf/JSON request
	// body. it could be fixed by follow-up commits, as JSON is not the primary ingestion content type.
	//
	// the following assertion will skip such fields (which are found manually). it can be used as a reference of:
	// - "which field needs pointer and omitempty"
	ignoreFields := map[string]bool{
		"_time":                    true,
		"dropped_links_count":      true,
		"dropped_attributes_count": true,
		"dropped_events_count":     true,
		"status_code":              true,
		"flags":                    true,
	}
	ignoreFieldPrefixes := map[string]bool{
		"event:event_dropped_attributes_count": true,
	}
	for i := range response1.LogLines {
		// let's try not to complicate the test and simply unmarshal them as key-value, and compare.
		m1, m2 := make(map[string]string), make(map[string]string)
		_ = json.Unmarshal([]byte(response1.LogLines[i]), &m1)
		_ = json.Unmarshal([]byte(response2.LogLines[i]), &m2)
		for k, v := range m2 {
			if m1[k] != v {
				if ignoreFields[k] || ignoreFieldPrefixes[k[:len(k)-2]] {
					continue
				}
				t.Fatalf("Mismatched values for key %s: got %s, expected %s", k, m1[k], v)
			}
		}
		for k, v := range m1 {
			if m2[k] != v {
				if ignoreFields[k] || ignoreFieldPrefixes[k[:len(k)-2]] {
					continue
				}
				t.Fatalf("Mismatched values for key %s: got %s, expected %s", k, m2[k], v)
			}
		}
	}
}
