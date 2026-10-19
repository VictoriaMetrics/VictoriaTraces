package vtinsert

import (
	"flag"
	"fmt"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/nativeinsert"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/nativeinsert/nativemultitenant"
	"net/http"
	"strings"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/insertutil"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/internalinsert"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/opentelemetry"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/grpc"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/http2server"
)

var (
	disableInsert         = flag.Bool("insert.disable", false, "Whether to disable both /insert/* and /internal/insert HTTP endpoints and OpenTelemetry gRPC ingestion endpoint. Useful for dedicated vtselect nodes. See also -internalinsert.disable. See https://docs.victoriametrics.com/victoriatraces/cluster/#security")
	disableInternalInsert = flag.Bool("internalinsert.disable", false, "Whether to disable /internal/insert HTTP endpoint. See also -insert.disable. See https://docs.victoriametrics.com/victoriatraces/cluster/#security")
)

var otlpGRPCListenAddr = flag.String("otlpGRPCListenAddr", "", `TCP address for accepting OTLP gRPC requests. Defaults to empty, which means it is disabled. The recommended port is ":4317". See also -otlpGRPC.tls`)

// Init initializes vtinsert
func Init() {
	if *otlpGRPCListenAddr != "" {
		initGRPCServer()
	}

	insertutil.MustStartIndexWorker()
}

// Stop stops vtinsert
func Stop() {
	if *otlpGRPCListenAddr != "" {
		stopGRPCServer()
	}

	insertutil.MustStopIndexWorker()
}

// RequestHandler handles HTTP insert requests for VictoriaTraces
func RequestHandler(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ReplaceAll(r.URL.Path, "//", "/")

	if strings.HasPrefix(path, "/insert/") {
		if *disableInsert {
			httpserver.Errorf(w, r, "requests to /insert/* are disabled with -insert.disable command-line flag")
			return true
		}

		return insertHandler(w, r, path)
	}

	if path == "/internal/insert" {
		if *disableInternalInsert || *disableInsert {
			httpserver.Errorf(w, r, "requests to /internal/insert are disabled with -internalinsert.disable or -insert.disable command-line flag")
			return true
		}
		internalinsert.RequestHandler(w, r)
		return true
	}

	return false
}

// insertHandler handles HTTP insert request from public APIs.
func insertHandler(w http.ResponseWriter, r *http.Request, path string) bool {
	switch path {
	case "/insert/native":
		nativeinsert.RequestHandler(w, r)
		return true
	case "/insert/multitenant/native":
		nativemultitenant.RequestHandler(w, r)
		return true
	case "/insert/ready":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, `{"status":"ok"}`)
		return true
	}
	switch {
	case strings.HasPrefix(path, "/insert/opentelemetry/"):
		return opentelemetry.RequestHandler(path, w, r)
	}

	return false
}

// otlpGRPCRequestHandler handles OTLP gRPC insert requests over HTTP for VictoriaTraces.
func otlpGRPCRequestHandler(w http.ResponseWriter, r *http.Request) bool {
	if *disableInsert {
		grpc.Errorf(w, r, grpc.StatusCodeUnavailable, "requests to grpc export are disabled with -insert.disable command-line flag")
		return true
	}
	return opentelemetry.OTLPGRPCRequestHandler(r, w)
}

func initGRPCServer() {
	logger.Infof("starting OTLP gRPC server at %q...", *otlpGRPCListenAddr)
	http2server.Serve([]string{*otlpGRPCListenAddr}, otlpGRPCRequestHandler, http2server.ServeOptions{})
}

func stopGRPCServer() {
	startTime := time.Now()
	logger.Infof("gracefully shutting down the OTLP gRPC server at %q...", *otlpGRPCListenAddr)
	if err := http2server.Stop([]string{*otlpGRPCListenAddr}); err != nil {
		logger.Fatalf("cannot stop the OTLP gRPC server: %s", err)
	}
	logger.Infof("successfully shut down the OTLP gRPC in %.3f seconds", time.Since(startTime).Seconds())
}
