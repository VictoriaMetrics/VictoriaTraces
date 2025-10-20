package vtinsert

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/internalinsert"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/opentelemetry"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/grpc"
)

var (
	disableInsert   = flag.Bool("insert.disable", false, "Whether to disable /insert/* HTTP endpoints")
	disableInternal = flag.Bool("internalinsert.disable", false, "Whether to disable /internal/insert HTTP endpoint. See https://docs.victoriametrics.com/victoriatraces/cluster/#security")

	otlpGRPCListenAddr = flag.String("otlpGRPCListenAddr", "", `TCP address for accepting OTLP gRPC requests. ":4317" is the recommend value when needed.`)
)

var (
	otlpGRPCServer http.Server
)

// Init initializes vtinsert
func Init() {
	if *otlpGRPCListenAddr != "" {
		otlpGRPCServer = http.Server{
			Addr:    *otlpGRPCListenAddr,
			Handler: h2c.NewHandler(http.HandlerFunc(otlpGRPCRequestHandler), &http2.Server{}),
		}
		logger.Infof("starting OTLP gPRC server at %q...", *otlpGRPCListenAddr)
		go func() {
			if err := otlpGRPCServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Fatalf("http2 server start error: %s", err)
			}
		}()
	}
}

// Stop stops vtinsert
func Stop() {
	if *otlpGRPCListenAddr != "" {
		startTime := time.Now()
		logger.Infof("gracefully shutting down the OTLP gPRC server at %q...", *otlpGRPCListenAddr)
		otlpGRPCServer.Shutdown(context.Background())
		logger.Infof("successfully shut down the OTLP gPRC  in %.3f seconds", time.Since(startTime).Seconds())
	}
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

// otlpGRPCRequestHandler handles OTLP gRPC insert requests over HTTP for VictoriaTraces.
func otlpGRPCRequestHandler(w http.ResponseWriter, r *http.Request) {
	if *disableInsert {
		opentelemetry.WriteErrorGrpcResponse(w, grpc.StatusCodeUnavailable, "requests to grpc export are disabled with -insert.disable command-line flag")
		return
	}
	opentelemetry.OTLPGRPCRequestHandler(r, w)
}
