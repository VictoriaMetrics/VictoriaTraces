package tests

import (
	"net"
	"net/http"
	"testing"

	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
)

// TestQueryAPIsUnavailableStorageNode checks that the Jaeger and Tempo query APIs
// return 502 Bad Gateway instead of 400 Bad Request when a -storageNode is unavailable.
//
// See https://github.com/VictoriaMetrics/VictoriaTraces/issues/276
func TestQueryAPIsUnavailableStorageNode(t *testing.T) {
	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartVtselect("vtselect", []string{unusedTCPAddr(t)}, nil)
	cli := tc.Client()
	baseURL := "http://" + sut.HTTPAddr()

	paths := []string{
		"/select/jaeger/api/services",
		"/select/jaeger/api/services/svc/operations",
		"/select/jaeger/api/traces?service=svc",
		"/select/jaeger/api/traces/0123456789abcdef0123456789abcdef",
		"/select/jaeger/api/dependencies?endTs=1700000000000&lookback=3600000",
		"/select/tempo/api/search?q=%7B%7D",
		"/select/tempo/api/v2/search/tags",
		"/select/tempo/api/v2/search/tags?scope=span",
		"/select/tempo/api/v2/search/tag/resource.service.name/values",
		"/select/tempo/api/traces/0123456789abcdef0123456789abcdef",
		"/select/tempo/api/v2/traces/0123456789abcdef0123456789abcdef",
		"/select/tempo/api/metrics/query_range?q=%7B%7D%20%7C%20rate()",
	}
	for _, path := range paths {
		body, statusCode := cli.Get(t, baseURL+path)
		if statusCode != http.StatusBadGateway {
			t.Errorf("unexpected status code for GET %s; got %d; want %d; response: %q",
				path, statusCode, http.StatusBadGateway, body)
		}
	}
}

// unusedTCPAddr returns a local TCP address where nothing listens.
func unusedTCPAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot listen on a TCP port: %s", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("cannot close the TCP listener: %s", err)
	}
	return addr
}
