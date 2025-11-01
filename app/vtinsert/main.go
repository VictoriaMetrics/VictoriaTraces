package vtinsert

import (
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/flagutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/internalinsert"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/opentelemetry"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/grpc"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/http2server"
)

var (
	disableInsert   = flag.Bool("insert.disable", false, "Whether to disable /insert/* HTTP endpoints")
	disableInternal = flag.Bool("internalinsert.disable", false, "Whether to disable /internal/insert HTTP endpoint. See https://docs.victoriametrics.com/victoriatraces/cluster/#security")

	otlpGRPCListenAddr       = flag.String("otlpGRPCListenAddr", "", `TCP address for accepting OTLP gRPC requests. Defaults to empty, which means it is disabled. The recommended port is ":4317".`)
	otlpGRPCUseProxyProtocol = flag.Bool("otlpGRPCListenAddr.useProxyProtocol", false, "Whether to use proxy protocol for connections accepted at -otlpGRPCListenAddr . "+
		"See https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt")
)

// Init initializes vtinsert
func Init() {
	if *otlpGRPCListenAddr != "" {
		logger.Infof("starting OTLP gPRC server at %q...", *otlpGRPCListenAddr)
		go http2server.Serve(
			[]string{*otlpGRPCListenAddr},
			otlpGRPCRequestHandler,
			http2server.ServeOptions{UseProxyProtocol: &flagutil.ArrayBool{*otlpGRPCUseProxyProtocol}},
		)
	}
}

// Stop stops vtinsert
func Stop() {
	if *otlpGRPCListenAddr != "" {
		startTime := time.Now()
		logger.Infof("gracefully shutting down the OTLP gPRC server at %q...", *otlpGRPCListenAddr)
		if err := http2server.Stop([]string{*otlpGRPCListenAddr}); err != nil {
			logger.Fatalf("cannot stop the OTLP gRPC server: %s", err)
		}
		logger.Infof("successfully shut down the OTLP gPRC in %.3f seconds", time.Since(startTime).Seconds())
	}
}

// RequestHandler handles HTTP insert requests for VictoriaTraces
func RequestHandler(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ReplaceAll(r.URL.Path, "//", "/")

	if strings.HasPrefix(path, "/insert/") {
		if *disableInsert {
			http2server.Errorf(w, r, "requests to /insert/* are disabled with -insert.disable command-line flag")
			return true
		}

		return insertHandler(w, r, path)
	}

	if path == "/internal/insert" {
		if *disableInternal || *disableInsert {
			http2server.Errorf(w, r, "requests to /internal/insert are disabled with -internalinsert.disable or -insert.disable command-line flag")
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
		grpc.WriteErrorGrpcResponse(w, grpc.StatusCodeUnavailable, "requests to grpc export are disabled with -insert.disable command-line flag")
		return true
	}
	return opentelemetry.OTLPGRPCRequestHandler(r, w)
}
