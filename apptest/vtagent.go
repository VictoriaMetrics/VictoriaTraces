package apptest

import (
	"bytes"
	"encoding/binary"
	"fmt"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Vtagent holds the state of a vtagent app and provides vtagent-specific functions
// Vtsingle holds the state of a Vtsingle app and provides Vtsingle-specific
// functions.
type Vtagent struct {
	*app
	*ServesMetrics

	remoteStoragesCount int
	httpListenAddr      string

	otlpTracesURL     string
	otlpGRPCTracesURL string
}

// StartVtagent starts an instance of Vtagent with the given flags.
// It also sets the default flags and populates the app instance state with
// values extracted from the application log (such as httpListenAddr).
func StartVtagent(instance string, remoteWriteURLs, flags []string, cli *Client) (*Vtagent, error) {
	app, stderrExtracts, err := startApp(instance, "../../bin/vtagent-race", flags, &appOptions{
		defaultFlags: map[string]string{
			"-httpListenAddr":            "127.0.0.1:0",
			"-otlpGRPCListenAddr":        "127.0.0.1:0",
			"-otlpGRPC.tls":              "false",
			"-remoteWrite.url":           strings.Join(remoteWriteURLs, ","),
			"-remoteWrite.tmpDataPath":   fmt.Sprintf("%s/%s-%d", os.TempDir(), instance, time.Now().UnixNano()),
			"-remoteWrite.flushInterval": "10ms",
			"-remoteWrite.showURL":       "true",
		},
		extractREs: []*regexp.Regexp{
			httpListenAddrRE,
			gRPCListenAddrRE,
		},
	})
	if err != nil {
		return nil, err
	}

	return &Vtagent{
		app: app,
		ServesMetrics: &ServesMetrics{
			metricsURL: fmt.Sprintf("http://%s/metrics", stderrExtracts[1]),
			cli:        cli,
		},

		remoteStoragesCount: len(remoteWriteURLs),
		httpListenAddr:      stderrExtracts[0],
		otlpTracesURL:       fmt.Sprintf("http://%s/insert/opentelemetry/v1/traces", stderrExtracts[0]),
		otlpGRPCTracesURL:   fmt.Sprintf("http://%s/opentelemetry.proto.collector.trace.v1.TraceService/Export", stderrExtracts[1]),
	}, nil
}

// OTLPHTTPExportTraces is a test helper function that exports OTLP trace data
// by sending an HTTP POST request to /insert/opentelemetry/v1/traces
// Vtagent endpoint.
func (app *Vtagent) OTLPHTTPExportTraces(t *testing.T, request *otelpb.ExportTraceServiceRequest, opts QueryOpts) {
	t.Helper()

	pbData := request.MarshalProtobuf(nil)
	app.OTLPHTTPExportRawTraces(t, pbData, opts)
}

// OTLPgRPCExportTraces is a test helper function that exports OTLP trace data
// by sending an `Export` gRPC call to a TraceService provider (Vtagent).
func (app *Vtagent) OTLPgRPCExportTraces(t *testing.T, request *otelpb.ExportTraceServiceRequest, _ QueryOpts) {
	t.Helper()

	pbData := request.MarshalProtobuf(nil)

	// 5 bytes prefix: 1 byte compress flag + 4 bytes body length
	buf := make([]byte, 5)
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(pbData)))

	reqBody := append(buf, pbData...)

	// must use a http2 client
	client := GetHTTP2Client()

	resp, err := client.Post(app.otlpGRPCTracesURL, "application/grpc", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("go error: %s", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("got %d, expected 200", resp.StatusCode)
	}
}

// OTLPHTTPExportRawTraces is a test helper function that exports raw OTLP trace data in []byte
// by sending an HTTP POST request to /insert/opentelemetry/v1/traces
// Vtagent endpoint.
func (app *Vtagent) OTLPHTTPExportRawTraces(t *testing.T, data []byte, opts QueryOpts) {
	t.Helper()

	contentType := "application/x-protobuf"
	if opts.HTTPHeaders != nil && opts.HTTPHeaders["Content-Type"] != "" {
		contentType = opts.HTTPHeaders["Content-Type"]
	}

	body, code := app.cli.Post(t, app.otlpTracesURL, contentType, data)
	if code != 200 {
		t.Fatalf("got %d, expected 200. body: %s", code, body)
	}
}
