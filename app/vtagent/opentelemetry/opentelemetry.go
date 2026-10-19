package opentelemetry

import (
	"flag"
	"fmt"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/flagutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/easyproto"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtagent/remotewrite"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/http2server"
)

var maxRequestSize = flagutil.NewBytes("opentelemetry.traces.maxRequestSize", 32*1024*1024, "The maximum size in bytes of a single OpenTelemetry trace export request.")

var otlpGRPCListenAddr = flag.String("otlpGRPCListenAddr", "", `TCP address for accepting OTLP gRPC requests. Defaults to empty, which means it is disabled. The recommended port is ":4317". See also -otlpGRPC.tls`)

// Init starts the OTLP gRPC server if -otlpGRPCListenAddr is set.
func Init() {
	if *otlpGRPCListenAddr == "" {
		return
	}

	logger.Infof("starting OTLP gRPC server at %q...", *otlpGRPCListenAddr)
	http2server.Serve([]string{*otlpGRPCListenAddr}, OTLPGRPCRequestHandler, http2server.ServeOptions{})
}

// Stop stops the OTLP gRPC server if it was started.
func Stop() {
	if *otlpGRPCListenAddr == "" {
		return
	}

	startTime := time.Now()
	logger.Infof("gracefully shutting down the OTLP gRPC server at %q...", *otlpGRPCListenAddr)
	if err := http2server.Stop([]string{*otlpGRPCListenAddr}); err != nil {
		logger.Fatalf("cannot stop the OTLP gRPC server: %s", err)
	}
	logger.Infof("successfully shut down the OTLP gRPC server in %.3f seconds", time.Since(startTime).Seconds())
}

// pushExportTraceServiceRequest pushes the OTLP ExportTraceServiceRequest protobuf message in data to remote storage.
//
// data isn't unmarshaled. Only its structure down to spans is verified, so obviously malformed requests
// are rejected before they are put into the persistent queue.
func pushExportTraceServiceRequest(data []byte) error {
	spansCount, err := countSpans(data)
	if err != nil {
		return fmt.Errorf("cannot decode ExportTraceServiceRequest from %d bytes: %w", len(data), err)
	}
	if spansCount == 0 {
		return nil
	}
	remotewrite.PushOTLPTraces(data, spansCount)
	return nil
}

// countSpans returns the number of spans in the ExportTraceServiceRequest protobuf message in src.
//
// It walks only through ExportTraceServiceRequest, ResourceSpans and ScopeSpans messages,
// while the contents of Resource, InstrumentationScope and Span messages are skipped without decoding.
//
// See https://github.com/open-telemetry/opentelemetry-proto/blob/v1.5.0/opentelemetry/proto/collector/trace/v1/trace_service.proto#L36
func countSpans(src []byte) (int, error) {
	//message ExportTraceServiceRequest {
	//	repeated opentelemetry.proto.trace.v1.ResourceSpans resource_spans = 1;
	//}
	var (
		fc  easyproto.FieldContext
		err error
		n   int
	)
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return 0, fmt.Errorf("cannot read next field in ExportTraceServiceRequest: %w", err)
		}
		if fc.FieldNum != 1 {
			continue
		}
		data, ok := fc.MessageData()
		if !ok {
			return 0, fmt.Errorf("cannot read resource_spans data")
		}
		m, err := countSpansInResourceSpans(data)
		if err != nil {
			return 0, fmt.Errorf("cannot read resource_spans: %w", err)
		}
		n += m
	}
	return n, nil
}

func countSpansInResourceSpans(src []byte) (int, error) {
	//message ResourceSpans {
	//	opentelemetry.proto.resource.v1.Resource resource = 1;
	//	repeated ScopeSpans scope_spans = 2;
	//	string schema_url = 3;
	//}
	var (
		fc  easyproto.FieldContext
		err error
		n   int
	)
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return 0, fmt.Errorf("cannot read next field in ResourceSpans: %w", err)
		}
		if fc.FieldNum != 2 {
			continue
		}
		data, ok := fc.MessageData()
		if !ok {
			return 0, fmt.Errorf("cannot read scope_spans data")
		}
		m, err := countSpansInScopeSpans(data)
		if err != nil {
			return 0, fmt.Errorf("cannot read scope_spans: %w", err)
		}
		n += m
	}
	return n, nil
}

func countSpansInScopeSpans(src []byte) (int, error) {
	//message ScopeSpans {
	//	opentelemetry.proto.common.v1.InstrumentationScope scope = 1;
	//	repeated Span spans = 2;
	//	string schema_url = 3;
	//}
	var (
		fc  easyproto.FieldContext
		err error
		n   int
	)
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return 0, fmt.Errorf("cannot read next field in ScopeSpans: %w", err)
		}
		if fc.FieldNum != 2 {
			continue
		}
		if _, ok := fc.MessageData(); !ok {
			return 0, fmt.Errorf("cannot read span data")
		}
		n++
	}
	return n, nil
}
