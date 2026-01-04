package vtselect

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/buildinfo"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/cgroup"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httputil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/metrics"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/internalselect"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/logsql"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/jaeger"
)

var (
	maxConcurrentRequests = flag.Int("search.maxConcurrentRequests", getDefaultMaxConcurrentRequests(), "The maximum number of concurrent search requests. "+
		"It shouldn't be high, since a single request can saturate all the CPU cores, while many concurrently executed requests may require high amounts of memory. "+
		"See also -search.maxQueueDuration")
	maxQueueDuration = flag.Duration("search.maxQueueDuration", 10*time.Second, "The maximum time the search request waits for execution when -search.maxConcurrentRequests "+
		"limit is reached; see also -search.maxQueryDuration")
	maxQueryDuration = flag.Duration("search.maxQueryDuration", time.Second*30, "The maximum duration for query execution. It can be overridden to a smaller value on a per-query basis via 'timeout' query arg")

	disableSelect         = flag.Bool("select.disable", false, "Whether to disable /select/* HTTP endpoints")
	disableInternalSelect = flag.Bool("internalselect.disable", false, "Whether to disable /internal/select/* HTTP endpoints")

	enableDelete         = flag.Bool("delete.enable", false, "Whether to enable /delete/* HTTP endpoints")
	enableInternalDelete = flag.Bool("internaldelete.enable", false, "Whether to enable /internal/delete/* HTTP endpoints, which are used by vtselect for deleting spans "+
		"via delete API at vtstorage nodes")
	logSlowQueryDuration = flag.Duration("search.logSlowQueryDuration", 5*time.Second,
		"Log queries with execution time exceeding this value. Zero disables slow query logging")
)

func getDefaultMaxConcurrentRequests() int {
	n := cgroup.AvailableCPUs()
	if n <= 4 {
		n *= 2
	}
	if n > 16 {
		// A single request can saturate all the CPU cores, so there is no sense
		// in allowing higher number of concurrent requests - they will just contend
		// for unavailable CPU time.
		n = 16
	}
	return n
}

// Init initializes vtselect
func Init() {
	concurrencyLimitCh = make(chan struct{}, *maxConcurrentRequests)
}

// Stop stops vtselect
func Stop() {
}

var concurrencyLimitCh chan struct{}

var (
	concurrencyLimitReached = metrics.NewCounter(`vt_concurrent_select_limit_reached_total`)
	concurrencyLimitTimeout = metrics.NewCounter(`vt_concurrent_select_limit_timeout_total`)

	_ = metrics.NewGauge(`vt_concurrent_select_capacity`, func() float64 {
		return float64(cap(concurrencyLimitCh))
	})
	_ = metrics.NewGauge(`vt_concurrent_select_current`, func() float64 {
		return float64(len(concurrencyLimitCh))
	})
)

//go:embed vmui
var vmuiFiles embed.FS

var vmuiFileServer = http.FileServer(http.FS(vmuiFiles))

// RequestHandler handles select requests for VictoriaTraces
func RequestHandler(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ReplaceAll(r.URL.Path, "//", "/")

	if strings.HasPrefix(path, "/delete/") {
		if !*enableDelete {
			httpserver.Errorf(w, r, "requests to /delete/* are disabled; pass -delete.enable command-line flag for enabling them")
			return true
		}
		deleteHandler(w, r, path)
		return true
	}

	if strings.HasPrefix(path, "/select/") {
		if *disableSelect {
			httpserver.Errorf(w, r, "requests to /select/* are disabled with -select.disable command-line flag")
			return true
		}

		return selectHandler(w, r, path)
	}

	if strings.HasPrefix(path, "/internal/delete/") {
		if !*enableInternalDelete {
			httpserver.Errorf(w, r, "requests to /internal/delete/*` are disabled; pass -internaldelete.enable command-line flag for enabling them")
			return true
		}
		internalselect.RequestHandler(r.Context(), w, r)
		return true
	}

	if strings.HasPrefix(path, "/internal/select/") {
		if *disableInternalSelect {
			httpserver.Errorf(w, r, "requests to /internal/select/* are disabled with -internalselect.disable command-line flag")
			return true
		}
		if *disableSelect {
			httpserver.Errorf(w, r, "requests to /internal/select/* are disabled with -select.disable command-line flag")
			return true
		}
		internalselect.RequestHandler(r.Context(), w, r)
		return true
	}

	return false
}

func selectHandler(w http.ResponseWriter, r *http.Request, path string) bool {
	ctx := r.Context()

	if path == "/select/buildinfo" {
		httpserver.EnableCORS(w, r)

		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, `{"status":"error","msg":"method %q isn't allowed"}`, r.Method)
			return true
		}

		v := buildinfo.ShortVersion()
		if v == "" {
			// buildinfo.ShortVersion() may return empty result for builds without tags
			v = buildinfo.Version
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"success","data":{"version":%q}}`, v)
		return true
	}

	if path == "/select/vmui" {
		// VMUI access via incomplete url without `/` in the end. Redirect to complete url.
		// Use relative redirect, since the hostname and path prefix may be incorrect if VictoriaMetrics
		// is hidden behind vmauth or similar proxy.
		_ = r.ParseForm()
		newURL := "vmui/?" + r.Form.Encode()
		httpserver.Redirect(w, newURL)
		return true
	}
	if strings.HasPrefix(path, "/select/vmui/") {
		if strings.HasPrefix(path, "/select/vmui/static/") {
			// Allow clients caching static contents for long period of time, since it shouldn't change over time.
			// Path to static contents (such as js and css) must be changed whenever its contents is changed.
			// See https://developer.chrome.com/docs/lighthouse/performance/uses-long-cache-ttl/
			w.Header().Set("Cache-Control", "max-age=31536000")
		}
		r.URL.Path = strings.TrimPrefix(path, "/select")
		vmuiFileServer.ServeHTTP(w, r)
		return true
	}

	if path == "/select/logsql/tail" {
		logsqlTailRequests.Inc()
		// Process live tailing request without timeout, since it is OK to run live tailing requests for very long time.
		// Also do not apply concurrency limit to tail requests, since these limits are intended for non-tail requests.
		logsql.ProcessLiveTailRequest(ctx, w, r)
		return true
	}

	// Limit the number of concurrent queries, which can consume big amounts of CPU time.
	startTime := time.Now()
	d := getMaxQueryDuration(r)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, d)
	defer cancel()

	if !incRequestConcurrency(ctxWithTimeout, w, r) {
		return true
	}
	defer decRequestConcurrency()

	if strings.HasPrefix(path, "/select/jaeger/") {
		// Jaeger HTTP APIs for distributed tracing.
		// Could be used by Grafana Jaeger datasource, Jaeger UI, and more.
		return jaeger.RequestHandler(ctxWithTimeout, w, r)
	}

	ok := processSelectRequest(ctxWithTimeout, w, r, path)
	if !ok {
		return false
	}

	// Log slow queries
	if *logSlowQueryDuration > 0 {
		d := time.Since(startTime)
		if d >= *logSlowQueryDuration {
			remoteAddr := httpserver.GetQuotedRemoteAddr(r)
			requestURI := httpserver.GetRequestURI(r)
			logger.Warnf("slow query according to -search.logSlowQueryDuration=%s: remoteAddr=%s, duration=%.3f seconds; requestURI: %q",
				*logSlowQueryDuration, remoteAddr, d.Seconds(), requestURI)
			slowQueries.Inc()
		}
	}

	logRequestErrorIfNeeded(ctxWithTimeout, w, r, startTime)
	return true
}

func incRequestConcurrency(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	startTime := time.Now()
	stopCh := ctx.Done()
	select {
	case concurrencyLimitCh <- struct{}{}:
		return true
	default:
		// Sleep for a while until giving up. This should resolve short bursts in requests.
		concurrencyLimitReached.Inc()
		select {
		case concurrencyLimitCh <- struct{}{}:
			return true
		case <-stopCh:
			switch ctx.Err() {
			case context.Canceled:
				remoteAddr := httpserver.GetQuotedRemoteAddr(r)
				requestURI := httpserver.GetRequestURI(r)
				logger.Infof("client has canceled the pending request after %.3f seconds: remoteAddr=%s, requestURI: %q",
					time.Since(startTime).Seconds(), remoteAddr, requestURI)
			case context.DeadlineExceeded:
				concurrencyLimitTimeout.Inc()
				err := &httpserver.ErrorWithStatusCode{
					Err: fmt.Errorf("couldn't start executing the request in %.3f seconds, since -search.maxConcurrentRequests=%d concurrent requests "+
						"are executed. Possible solutions: to reduce query load; to add more compute resources to the server; "+
						"to increase -search.maxQueueDuration=%s; to increase -search.maxQueryDuration=%s; to increase -search.maxConcurrentRequests; "+
						"to pass bigger value to 'timeout' query arg",
						time.Since(startTime).Seconds(), *maxConcurrentRequests, maxQueueDuration, maxQueryDuration),
					StatusCode: http.StatusServiceUnavailable,
				}
				httpserver.Errorf(w, r, "%s", err)
			}
			return false
		}
	}
}

func decRequestConcurrency() {
	<-concurrencyLimitCh
}

// getMaxQueryDuration returns the maximum duration for query from r.
func getMaxQueryDuration(r *http.Request) time.Duration {
	dms, err := httputil.GetDuration(r, "timeout", 0)
	if err != nil {
		dms = 0
	}
	d := time.Duration(dms) * time.Millisecond
	if d <= 0 || d > *maxQueryDuration {
		d = *maxQueryDuration
	}
	return d
}
