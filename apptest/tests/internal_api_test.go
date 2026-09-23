package tests

import (
	"net/http"
	"os"
	"testing"

	at "github.com/VictoriaMetrics/VictoriaTraces/apptest"
)

// TestSingleInternalEndpointsRequirePost checks that the /internal/* endpoints drop every
// method except POST, so a server-side request forgery cannot reach them with a plain URL fetch.
//
// See https://github.com/VictoriaMetrics/VictoriaTraces/issues/225
func TestSingleInternalEndpointsRequirePost(t *testing.T) {
	os.RemoveAll(t.Name())

	tc := at.NewTestCase(t)
	defer tc.Stop()

	sut := tc.MustStartVtsingle("vtsingle", []string{
		"-storageDataPath=" + tc.Dir() + "/vtsingle",
		"-retentionPeriod=100y",
		"-internaldelete.enable",
	})
	cli := tc.Client()
	baseURL := "http://" + sut.HTTPAddr()

	f := func(method, path string) {
		t.Helper()

		body, statusCode := cli.Do(t, method, baseURL+path, "", nil)
		if statusCode != http.StatusMethodNotAllowed {
			t.Fatalf("unexpected status code for %s %s; got %d; want %d; response: %q",
				method, path, statusCode, http.StatusMethodNotAllowed, body)
		}
	}

	// the RPC endpoints between vtinsert, vtselect and vtstorage, plus the endpoints an operator calls
	paths := []string{
		"/internal/insert",
		"/internal/select/query",
		"/internal/delete/run_task",
		"/internal/force_flush",
		"/internal/force_merge",
		"/internal/partition/list",
	}
	for _, path := range paths {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodPatch} {
			f(method, path)
		}
	}

	// a POST request must pass the method check
	body, statusCode := cli.Do(t, http.MethodPost, baseURL+"/internal/force_flush", "", nil)
	if statusCode != http.StatusOK {
		t.Fatalf("unexpected status code for POST /internal/force_flush; got %d; want %d; response: %q",
			statusCode, http.StatusOK, body)
	}
}
