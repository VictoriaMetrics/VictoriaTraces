package vtinsert

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/internalinsert"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/opentelemetry"
)

var (
	disableInsert   = flag.Bool("insert.disable", false, "Whether to disable /insert/* HTTP endpoints")
	disableInternal = flag.Bool("internalinsert.disable", false, "Whether to disable /internal/insert HTTP endpoint. See https://docs.victoriametrics.com/victoriatraces/cluster/#security")
)

// Init initializes vtinsert
func Init() {
}

// Stop stops vtinsert
func Stop() {
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
		if *disableInternal || *disableInsert {
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

// GRPCRequestHandler handles gRPC insert requests over HTTP for VictoriaTraces.
func GRPCRequestHandler(w http.ResponseWriter, r *http.Request) {
	if *disableInsert {
		opentelemetry.WriteErrorGrpcResponse(w, opentelemetry.GrpcUnavailable, "requests to grpc export are disabled with -insert.disable command-line flag")
		return
	}
	opentelemetry.GrpcExportHandler(r, w)
}
