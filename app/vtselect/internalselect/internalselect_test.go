package internalselect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestRequestHandlerRequiresPOST checks that the /internal/select/* and /internal/delete/*
// endpoints answer 405 to every method except POST.
//
// See https://github.com/VictoriaMetrics/VictoriaTraces/issues/225
func TestRequestHandlerRequiresPOST(t *testing.T) {
	Init()
	defer Stop()

	f := func(method, path string) {
		t.Helper()

		r := httptest.NewRequest(method, path, nil)
		w := httptest.NewRecorder()
		RequestHandler(context.Background(), w, r)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("unexpected status code for %s %s; got %d; want %d; response: %q",
				method, path, w.Code, http.StatusMethodNotAllowed, w.Body.String())
		}
	}

	f(http.MethodGet, "/internal/select/query")
	f(http.MethodGet, "/internal/select/field_names")
	f(http.MethodGet, "/internal/delete/run_task?filter=*")

	f(http.MethodDelete, "/internal/select/query")
	f(http.MethodDelete, "/internal/select/field_names")
	f(http.MethodDelete, "/internal/delete/run_task?filter=*")
}

// TestRequestHandlerPostPassesTheMethodCheck checks that a POST request reaches the args
// parsing, so the method check above rejects nothing which vtselect sends.
//
// vtselect always sends POST here, see getResponseBodyForPathAndArgs in app/vtstorage/netselect.
func TestRequestHandlerPostPassesTheMethodCheck(t *testing.T) {
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
