package tracecommon

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
)

func TestGetCommonParamsAppliesExtraFilters(t *testing.T) {
	req := httptest.NewRequest("GET", "/select/tempo/api/search?extra_filters=%7B%22env%22%3A%22prod%22%7D", nil)

	cp, err := GetCommonParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	q, err := logstorage.ParseQuery(`status_code:=2`)
	if err != nil {
		t.Fatalf("cannot parse query: %s", err)
	}

	got := cp.ApplyExtraFilters(q).String()
	if !strings.Contains(got, "env:=prod") {
		t.Fatalf("missing extra filter in query %q", got)
	}
	if !strings.Contains(got, "status_code:=2") {
		t.Fatalf("missing original filter in query %q", got)
	}
}

func TestGetCommonParamsMergesMultipleExtraFilters(t *testing.T) {
	req := httptest.NewRequest("GET", "/select/tempo/api/search?extra_filters=env:%3Dprod&extra_filters=cluster:%3Deast", nil)

	cp, err := GetCommonParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	q, err := logstorage.ParseQuery(`status_code:=2`)
	if err != nil {
		t.Fatalf("cannot parse query: %s", err)
	}

	got := cp.ApplyExtraFilters(q).String()
	for _, want := range []string{"env:=prod", "cluster:=east", "status_code:=2"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in query %q", want, got)
		}
	}
}

func TestGetCommonParamsRejectsInvalidExtraFilters(t *testing.T) {
	req := httptest.NewRequest("GET", "/select/tempo/api/search?extra_filters=foo:(bar", nil)

	if _, err := GetCommonParams(req); err == nil {
		t.Fatal("expected error for invalid extra_filters")
	}
}
