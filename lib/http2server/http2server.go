package http2server

import (
	"context"
	"crypto/tls"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/fasttime"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/netutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/stringsutil"
	metrics "github.com/VictoriaMetrics/metrics"
	"github.com/klauspost/compress/gzhttp"
	"github.com/valyala/fastrand"
)

var (
	servers     = make(map[string]*server)
	serversLock sync.Mutex
)

type server struct {
	shutdownDelayDeadline atomic.Int64
	s                     *http.Server
}

// ServeOptions defines optional parameters for http server
type ServeOptions struct{}

// Serve starts an http server on the given addrs with the given optional rh.
//
// By default, all the responses are transparently compressed, since egress traffic is usually expensive.
func Serve(addr string, rh httpserver.RequestHandler, tlsConfig *tls.Config) {
	if rh == nil {
		rh = func(_ http.ResponseWriter, _ *http.Request) bool {
			return false
		}
	}
	go serve(addr, rh, tlsConfig)
}

func serve(addr string, rh httpserver.RequestHandler, tlsConfig *tls.Config) {
	scheme := "http2"
	useProxyProto := false

	ln, err := netutil.NewTCPListener(scheme, addr, useProxyProto, tlsConfig)
	if err != nil {
		logger.Fatalf("cannot start http/2 server at %s: %s", addr, err)
	}
	logger.Infof("started http/2 server at %s://%s/", scheme, ln.Addr())

	serveWithListener(addr, ln, rh)
}

func serveWithListener(addr string, ln net.Listener, rh httpserver.RequestHandler) {
	var s server

	protocols := &http.Protocols{}
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true)

	s.s = &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       time.Minute,
		Protocols:         protocols,

		// Do not set ReadTimeout and WriteTimeout here,
		// since these timeouts must be controlled by request handlers.

		ErrorLog: logger.StdErrorLogger(),
	}
	s.s.SetKeepAlivesEnabled(true)
	s.s.ConnContext = func(ctx context.Context, _ net.Conn) context.Context {
		timeoutSec := 120
		// Add a jitter for connection timeout in order to prevent Thundering herd problem
		// when all the connections are established at the same time.
		// See https://en.wikipedia.org/wiki/Thundering_herd_problem
		jitterSec := fastrand.Uint32n(uint32(timeoutSec / 10))
		deadline := fasttime.UnixTimestamp() + uint64(timeoutSec) + uint64(jitterSec)
		return context.WithValue(ctx, connDeadlineTimeKey, &deadline)
	}
	rhw := rh
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerWrapper(w, r, rhw)
	})
	h = gzipHandlerWrapper(h)
	s.s.Handler = h

	serversLock.Lock()
	servers[addr] = &s
	serversLock.Unlock()
	if err := s.s.Serve(ln); err != nil {
		if err == http.ErrServerClosed {
			// The server gracefully closed.
			return
		}
		logger.Panicf("FATAL: cannot serve http at %s: %s", addr, err)
	}
}

func whetherToCloseConn(r *http.Request) bool {
	ctx := r.Context()
	v := ctx.Value(connDeadlineTimeKey)
	deadline, ok := v.(*uint64)
	return ok && fasttime.UnixTimestamp() > *deadline
}

var connDeadlineTimeKey = any("connDeadlineSecs")

// Stop stops the http server on the given addrs, which has been started via Serve func.
func Stop(addrs []string) error {
	var errGlobalLock sync.Mutex
	var errGlobal error

	var wg sync.WaitGroup
	for _, addr := range addrs {
		if addr == "" {
			continue
		}
		wg.Add(1)
		go func(addr string) {
			if err := stop(addr); err != nil {
				errGlobalLock.Lock()
				errGlobal = err
				errGlobalLock.Unlock()
			}
			wg.Done()
		}(addr)
	}
	wg.Wait()

	return errGlobal
}

func stop(addr string) error {
	serversLock.Lock()
	s := servers[addr]
	delete(servers, addr)
	serversLock.Unlock()
	if s == nil {
		err := fmt.Errorf("BUG: there is no server at %q", addr)
		logger.Panicf("%s", err)
		// The return is needed for golangci-lint: SA5011(related information): this check suggests that the pointer can be nil
		return err
	}

	deadline := time.Now().UnixNano()
	s.shutdownDelayDeadline.Store(deadline)
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	if err := s.s.Shutdown(ctx); err != nil {
		return fmt.Errorf("cannot gracefully shutdown http server at %q in 7s; error: %s", addr, err)
	}
	return nil
}

var gzipHandlerWrapper = func() func(http.Handler) http.HandlerFunc {
	hw, err := gzhttp.NewWrapper(gzhttp.CompressionLevel(1))
	if err != nil {
		panic(fmt.Errorf("BUG: cannot initialize gzip http wrapper: %w", err))
	}
	return hw
}()

var (
	connTimeoutClosedConns = metrics.NewCounter(`vm_http_conn_timeout_closed_conns_total{type="http2"}`)
)

var hostname = func() string {
	h, err := os.Hostname()
	if err != nil {
		// Cannot use logger.Errorf, since it isn't initialized yet.
		// So use log.Printf instead.
		log.Printf("ERROR: cannot determine hostname: %s", err)
		return "unknown"
	}
	return h
}()

func handlerWrapper(w http.ResponseWriter, r *http.Request, rh httpserver.RequestHandler) {
	// All the VictoriaMetrics code assumes that panic stops the process.
	// Unfortunately, the standard net/http.Server recovers from panics in request handlers,
	// so VictoriaMetrics state can become inconsistent after the recovered panic.
	// The following recover() code works around this by explicitly stopping the process after logging the panic.
	// See https://github.com/golang/go/issues/16542#issuecomment-246549902 for details.
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 1<<20)
			n := runtime.Stack(buf, false)
			fmt.Fprintf(os.Stderr, "panic: %v\n\n%s", err, buf[:n])
			os.Exit(1)
		}
	}()

	h := w.Header()
	h.Add("X-Server-Hostname", hostname)
	requestsTotal.Inc()
	if whetherToCloseConn(r) {
		connTimeoutClosedConns.Inc()
		h.Set("Connection", "close")
	}

	w = &responseWriterWithAbort{
		ResponseWriter: w,
	}
	if rh(w, r) {
		return
	}

	Errorf(w, r, "unsupported path requested: %q", r.URL.Path)
	unsupportedRequestErrors.Inc()
}

var (
	unsupportedRequestErrors = metrics.NewCounter(`vm_http_request_errors_total{path="*",reason="unsupported",type="http2"}`)

	requestsTotal = metrics.NewCounter(`vm_http_requests_all_total{type="http2"}`)
)

// GetQuotedRemoteAddr returns quoted remote address.
func GetQuotedRemoteAddr(r *http.Request) string {
	remoteAddr := r.RemoteAddr
	if addr := r.Header.Get("X-Forwarded-For"); addr != "" {
		remoteAddr += ", X-Forwarded-For: " + addr
	}
	// quote remoteAddr and X-Forwarded-For, since they may contain untrusted input
	return stringsutil.JSONString(remoteAddr)
}

type responseWriterWithAbort struct {
	http.ResponseWriter

	sentHeaders bool
	aborted     bool
}

func (rwa *responseWriterWithAbort) Write(data []byte) (int, error) {
	if rwa.aborted {
		return 0, fmt.Errorf("response connection is aborted")
	}
	if !rwa.sentHeaders {
		rwa.sentHeaders = true
	}
	return rwa.ResponseWriter.Write(data)
}

func (rwa *responseWriterWithAbort) WriteHeader(statusCode int) {
	if rwa.aborted {
		logger.WarnfSkipframes(1, "cannot write response headers with statusCode=%d, since the response connection has been aborted", statusCode)
		return
	}
	if rwa.sentHeaders {
		logger.WarnfSkipframes(1, "cannot write response headers with statusCode=%d, since they were already sent", statusCode)
		return
	}
	rwa.ResponseWriter.WriteHeader(statusCode)
	rwa.sentHeaders = true
}

// Flush implements net/http.Flusher interface
func (rwa *responseWriterWithAbort) Flush() {
	if rwa.aborted {
		return
	}
	if !rwa.sentHeaders {
		rwa.sentHeaders = true
	}
	flusher, ok := rwa.ResponseWriter.(http.Flusher)
	if !ok {
		logger.Panicf("BUG: it is expected http.ResponseWriter (%T) supports http.Flusher interface", rwa.ResponseWriter)
	}
	flusher.Flush()
}

// abort aborts the client connection associated with rwa.
//
// The last http chunk in the response stream is intentionally written incorrectly,
// so the client, which reads the response, could notice this error.
func (rwa *responseWriterWithAbort) abort() {
	if !rwa.sentHeaders {
		logger.Panicf("BUG: abort can be called only after http response headers are sent")
	}
	if rwa.aborted {
		// Nothing to do. The connection has been already aborted.
		return
	}
	hj, ok := rwa.ResponseWriter.(http.Hijacker)
	if !ok {
		logger.Panicf("BUG: ResponseWriter must implement http.Hijacker interface")
	}
	conn, bw, err := hj.Hijack()
	if err != nil {
		logger.WarnfSkipframes(2, "cannot hijack response connection: %s", err)
		return
	}

	// Just write an error message into the client connection as is without http chunked encoding.
	// This is needed in order to notify the client about the aborted connection.
	_, _ = bw.WriteString("\nthe connection has been aborted; see the last line in the response and/or in the server log for the reason\n")
	_ = bw.Flush()

	// Forcibly close the client connection in order to break http keep-alive at client side.
	_ = conn.Close()

	rwa.aborted = true
}

// Errorf writes formatted error message to w and to logger.
func Errorf(w http.ResponseWriter, r *http.Request, format string, args ...any) {
	errStr := fmt.Sprintf(format, args...)
	logHTTPError(r, errStr)

	// Extract statusCode from args
	statusCode := http.StatusBadRequest
	var esc *ErrorWithStatusCode
	for _, arg := range args {
		if err, ok := arg.(error); ok && errors.As(err, &esc) {
			statusCode = esc.StatusCode
			break
		}
	}

	if rwa, ok := w.(*responseWriterWithAbort); ok && rwa.sentHeaders {
		// HTTP status code has been already sent to client, so it cannot be sent again.
		// Just write errStr to the response and abort the client connection, so the client could notice the error.
		fmt.Fprintf(w, "\n%s\n", errStr)
		rwa.abort()
		return
	}
	http.Error(w, errStr, statusCode)
}

// logHTTPError logs the errStr with the client remote address and the request URI obtained from r.
func logHTTPError(r *http.Request, errStr string) {
	remoteAddr := GetQuotedRemoteAddr(r)
	requestURI := GetRequestURI(r)
	errStr = fmt.Sprintf("remoteAddr: %s; requestURI: %s; %s", remoteAddr, requestURI, errStr)
	logger.WarnfSkipframes(2, "%s", errStr)
}

// ErrorWithStatusCode is error with HTTP status code.
//
// The given StatusCode is sent to client when the error is passed to Errorf.
type ErrorWithStatusCode struct {
	Err        error
	StatusCode int
}

// Unwrap returns e.Err.
//
// This is used by standard errors package. See https://golang.org/pkg/errors
func (e *ErrorWithStatusCode) Unwrap() error {
	return e.Err
}

// Error implements error interface.
func (e *ErrorWithStatusCode) Error() string {
	return e.Err.Error()
}

// GetRequestURI returns requestURI for r.
func GetRequestURI(r *http.Request) string {
	requestURI := r.RequestURI
	if r.Method != http.MethodPost {
		return requestURI
	}
	_ = r.ParseForm()
	if len(r.PostForm) == 0 {
		return requestURI
	}
	// code copied from url.Query.Encode
	var queryArgs strings.Builder
	for k := range r.PostForm {
		vs := r.PostForm[k]
		// mask authKey as well-known secret
		if k == "authKey" {
			vs = []string{"secret"}
		}
		keyEscaped := url.QueryEscape(k)
		for _, v := range vs {
			if queryArgs.Len() > 0 {
				queryArgs.WriteByte('&')
			}
			queryArgs.WriteString(keyEscaped)
			queryArgs.WriteByte('=')
			queryArgs.WriteString(url.QueryEscape(v))
		}
	}
	delimiter := "?"
	if strings.Contains(requestURI, delimiter) {
		delimiter = "&"
	}
	return requestURI + delimiter + queryArgs.String()
}
