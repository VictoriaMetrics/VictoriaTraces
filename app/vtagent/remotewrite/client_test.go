package remotewrite

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/flagutil"
	"github.com/VictoriaMetrics/metrics"
)

func TestSendBlockHTTPStatusCodes(t *testing.T) {
	f := func(statusCode int, wantOK bool, wantSent, wantDropped uint64) {
		t.Helper()

		// The client is stopped before it gets the response, so a block that would be retried makes sendBlockHTTP return false.
		stopCh := make(chan struct{})
		var requests atomic.Uint64
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if requests.Add(1) == 1 {
				close(stopCh)
			}
			w.WriteHeader(statusCode)
		}))
		defer srv.Close()

		c := newHTTPClient(0, srv.URL, fmt.Sprintf("test-%d", statusCode), nil, 1)
		defer c.hc.CloseIdleConnections()
		c.stopCh = stopCh
		c.retryMinInterval = time.Hour
		c.retryMaxTime = time.Hour
		s := metrics.NewSet()
		c.bytesSent = s.NewCounter("bytes_sent")
		c.blocksSent = s.NewCounter("blocks_sent")
		c.requestDuration = s.NewHistogram("request_duration")
		c.requestsOKCount = s.NewCounter("requests_ok")
		c.errorsCount = s.NewCounter("errors")
		c.packetsDropped = s.NewCounter("packets_dropped")
		c.retriesCount = s.NewCounter("retries")

		if ok := c.sendBlockHTTP([]byte("block")); ok != wantOK {
			t.Fatalf("status %d: unexpected result; got %v; want %v", statusCode, ok, wantOK)
		}
		if n := requests.Load(); n != 1 {
			t.Fatalf("status %d: unexpected number of requests; got %d; want 1", statusCode, n)
		}
		if n := c.blocksSent.Get(); n != wantSent {
			t.Fatalf("status %d: unexpected blocks sent; got %d; want %d", statusCode, n, wantSent)
		}
		if n := c.packetsDropped.Get(); n != wantDropped {
			t.Fatalf("status %d: unexpected blocks dropped; got %d; want %d", statusCode, n, wantDropped)
		}
	}

	// success
	f(200, true, 1, 0)

	// the same block can never succeed, so it is dropped
	f(400, true, 0, 1)
	f(404, true, 0, 1)
	f(413, true, 0, 1)
	f(415, true, 0, 1)

	// retried
	f(412, false, 0, 0)
	f(414, false, 0, 0)
	f(429, false, 0, 0)
	f(503, false, 0, 0)
}

func TestParseRetryAfterHeader(t *testing.T) {
	f := func(retryAfterString string, expectResult time.Duration) {
		t.Helper()

		result := parseRetryAfterHeader(retryAfterString)
		// expect `expectResult == result` when retryAfterString is in seconds or invalid
		// expect the difference between result and expectResult to be lower than 10%
		if !(expectResult == result || math.Abs(float64(expectResult-result))/float64(expectResult) < 0.10) {
			t.Fatalf(
				"incorrect retry after duration, want (ms): %d, got (ms): %d",
				expectResult.Milliseconds(), result.Milliseconds(),
			)
		}
	}

	// retry after header in seconds
	f("10", 10*time.Second)
	// retry after header in date time
	f(time.Now().Add(30*time.Second).UTC().Format(http.TimeFormat), 30*time.Second)
	// retry after header invalid
	f("invalid-retry-after", 0)
	// retry after header not in GMT
	f(time.Now().Add(10*time.Second).Format("Mon, 02 Jan 2006 15:04:05 FAKETZ"), 0)
}

func TestInitSecretFlags(t *testing.T) {
	showRemoteWriteURLOrig := *showRemoteWriteURL
	defer func() {
		*showRemoteWriteURL = showRemoteWriteURLOrig
		flagutil.UnregisterAllSecretFlags()
	}()

	flagutil.UnregisterAllSecretFlags()
	*showRemoteWriteURL = false
	InitSecretFlags()
	if !flagutil.IsSecretFlag("remotewrite.url") {
		t.Fatalf("expecting remoteWrite.url to be secret")
	}
	if !flagutil.IsSecretFlag("remotewrite.proxyurl") {
		t.Fatalf("expecting remoteWrite.proxyURL to be secret")
	}
	if !flagutil.IsSecretFlag("remotewrite.headers") {
		t.Fatalf("expecting remoteWrite.headers to be secret")
	}

	flagutil.UnregisterAllSecretFlags()
	*showRemoteWriteURL = true
	InitSecretFlags()
	if flagutil.IsSecretFlag("remotewrite.url") {
		t.Fatalf("remoteWrite.url must remain visible when -remoteWrite.showURL is set")
	}
	if !flagutil.IsSecretFlag("remotewrite.proxyurl") {
		t.Fatalf("expecting remoteWrite.proxyURL to remain secret")
	}
	if !flagutil.IsSecretFlag("remotewrite.headers") {
		t.Fatalf("expecting remoteWrite.headers to remain secret")
	}
}
