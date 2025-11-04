package vtinsert

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/flagutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/netutil"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/internalinsert"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/opentelemetry"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/grpc"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/http2server"
)

var (
	disableInsert   = flag.Bool("insert.disable", false, "Whether to disable /insert/* HTTP endpoints")
	disableInternal = flag.Bool("internalinsert.disable", false, "Whether to disable /internal/insert HTTP endpoint. See https://docs.victoriametrics.com/victoriatraces/cluster/#security")
)

var (
	otlpGRPCListenAddr = flag.String("otlpGRPCListenAddr", "", `TCP address for accepting OTLP gRPC requests. Defaults to empty, which means it is disabled. The recommended port is ":4317".`)

	otlpGRPCTlsEnable   = flag.Bool("otlpGRPC.tls", true, "Enable TLS for incoming gRPC request at the given -otlpGRPCListenAddr. It's set to true by default, and -otlpGRPC.tlsCertFile and -otlpGRPC.tlsKeyFile must be set. It could be configured to false to allow insecure connection.")
	otlpGRPCTlsCertFile = flag.String("otlpGRPC.tlsCertFile", "", "Path to file with TLS certificate for the corresponding -otlpGRPCListenAddr if -otlpGRPC.tls is not set to false. "+
		"Prefer ECDSA certs instead of RSA certs as RSA certs are slower. The provided certificate file is automatically re-read every second, so it can be dynamically updated.")
	otlpGRPCTlsKeyFile = flag.String("otlpGRPC.tlsKeyFile", "", "Path to file with TLS key for the corresponding -otlpGRPCListenAddr if -otlpGRPC.tls is not set to false. "+
		"The provided key file is automatically re-read every second, so it can be dynamically updated.")
	otlpGRPCTlsCipherSuites = flagutil.NewArrayString("otlpGRPC.tlsCipherSuites", "Optional TLS cipher suites for incoming requests over HTTPS if -otlpGRPC.tls is not set to false. See the list of supported cipher suites at https://pkg.go.dev/crypto/tls#pkg-constants")
	otlpGRPCTlsMinVersion   = flag.String("otlpGRPC.tlsMinVersion", "", "Optional minimum TLS version to use for the corresponding -otlpGRPCListenAddr if -otlpGRPC.tls is not set to false. "+
		"Supported values: TLS10, TLS11, TLS12, TLS13.")
)

// Init initializes vtinsert
func Init() {
	if *otlpGRPCListenAddr != "" {
		initGRPCServer()
	}
}

// Stop stops vtinsert
func Stop() {
	if *otlpGRPCListenAddr != "" {
		stopGRPCServer()
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

func initGRPCServer() {
	var (
		err       error
		tlsConfig *tls.Config
	)

	if *otlpGRPCTlsEnable {
		if *otlpGRPCTlsKeyFile == "" {
			logger.Fatalf("-otlpGRPC.tlsKeyFile is required when -otlpGRPC.tls is true.")
		}
		if *otlpGRPCTlsCertFile == "" {
			logger.Fatalf("-otlpGRPC.tlsCertFile is required when -otlpGRPC.tls is true.")
		}
		tlsConfig, err = netutil.GetServerTLSConfig(*otlpGRPCTlsCertFile, *otlpGRPCTlsKeyFile, *otlpGRPCTlsMinVersion, *otlpGRPCTlsCipherSuites)
		if err != nil {
			logger.Fatalf("cannot load TLS cert from -tlsCertFile=%q, -tlsKeyFile=%q, -tlsMinVersion=%q, -tlsCipherSuites=%q: %s", *otlpGRPCTlsCertFile, *otlpGRPCTlsKeyFile, *otlpGRPCTlsMinVersion, *otlpGRPCTlsCipherSuites, err)
		}
	}

	logger.Infof("starting OTLP gPRC server at %q...", *otlpGRPCListenAddr)
	go http2server.Serve(
		*otlpGRPCListenAddr,
		otlpGRPCRequestHandler,
		tlsConfig,
	)
}

func stopGRPCServer() {
	startTime := time.Now()
	logger.Infof("gracefully shutting down the OTLP gPRC server at %q...", *otlpGRPCListenAddr)
	if err := http2server.Stop([]string{*otlpGRPCListenAddr}); err != nil {
		logger.Fatalf("cannot stop the OTLP gRPC server: %s", err)
	}
	logger.Infof("successfully shut down the OTLP gPRC in %.3f seconds", time.Since(startTime).Seconds())
}
