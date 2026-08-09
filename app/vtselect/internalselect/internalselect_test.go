package internalselect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestRequestHandlerDeleteRunTask_MethodNotAllowed checks that the endpoint which removes
// spans answers 405 to every method except POST.
//
// See https://github.com/VictoriaMetrics/VictoriaTraces/issues/225
func TestRequestHandlerDeleteRunTask_MethodNotAllowed(t *testing.T) {
	Init()
	defer Stop()

	f := func(method string) {
		t.Helper()

		r := httptest.NewRequest(method, "/internal/delete/run_task?filter=*", nil)
		w := httptest.NewRecorder()
		RequestHandler(context.Background(), w, r)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("unexpected status code for %s request; got %d; want %d; response: %q",
				method, w.Code, http.StatusMethodNotAllowed, w.Body.String())
		}
	}

	f(http.MethodGet)
	f(http.MethodHead)
	f(http.MethodPut)
	f(http.MethodDelete)
}

// TestRequestHandlerDeleteRunTask_PostPassesTheMethodCheck checks that a POST request
// reaches the args parsing, so the method check above rejects nothing which vtselect sends.
//
// vtselect always sends POST here, see getResponseBodyForPathAndArgs in app/vtstorage/netselect.
func TestRequestHandlerDeleteRunTask_PostPassesTheMethodCheck(t *testing.T) {
	Init()
	defer Stop()

	args := url.Values{}
	args.Set("filter", "*")
	r := httptest.NewRequest(http.MethodPost, "/internal/delete/run_task", strings.NewReader(args.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// the request carries no version arg, so it must fail on the protocol version check
	// rather than on the method check.
	RequestHandler(context.Background(), w, r)

	if w.Code == http.StatusMethodNotAllowed {
		t.Fatalf("a POST request must not be rejected by the method check; response: %q", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "unexpected protocol version") {
		t.Fatalf("unexpected response for a POST request; got %q; want the protocol version error", w.Body.String())
	}
}
