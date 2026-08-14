package tracecommon

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/extrafilters"
)

func TestGetExtraFilters_Success(t *testing.T) {
	f := func(query string, resultExpected []string) {
		t.Helper()

		r := httptest.NewRequest("GET", "/select/jaeger/api/services?"+query, nil)
		fs, err := getExtraFilters(r)
		if err != nil {
			t.Fatalf("unexpected error in getExtraFilters: %s", err)
		}
		result := make([]string, len(fs))
		for i, filter := range fs {
			result[i] = filter.String()
		}
		if len(result) != len(resultExpected) {
			t.Fatalf("unexpected number of filters\ngot\n%q\nwant\n%q", result, resultExpected)
		}
		for i := range result {
			if result[i] != resultExpected[i] {
				t.Fatalf("unexpected filter\ngot\n%q\nwant\n%q", result, resultExpected)
			}
		}
	}

	// no args
	f("", nil)

	// JSON form
	f(`extra_filters={"foo":"bar"}`, []string{"foo:=bar"})

	// LogsQL form
	f(`extra_filters=foo:bar`, []string{"foo:bar"})

	// the arg may repeat, and every filter must be kept
	f(`extra_filters={"foo":"bar"}&extra_filters={"baz":"qux"}`, []string{"foo:=bar", "baz:=qux"})

	// stream filters are parsed too
	f(`extra_stream_filters={"foo":"bar"}`, []string{`{foo="bar"}`})

	// an empty value parses to no filter, so it must not be added
	f(`extra_filters=&extra_filters={"foo":"bar"}`, []string{"foo:=bar"})

	// both args together
	f(`extra_filters={"foo":"bar"}&extra_stream_filters={"baz":"qux"}`, []string{"foo:=bar", `{baz="qux"}`})
}

func TestGetExtraFilters_Failure(t *testing.T) {
	f := func(query string) {
		t.Helper()

		r := httptest.NewRequest("GET", "/select/jaeger/api/services?"+query, nil)
		if _, err := getExtraFilters(r); err == nil {
			t.Fatalf("expecting non-nil error for %q", query)
		}
	}

	f(`extra_filters={"foo":}`)
	f(`extra_filters={"foo":"bar"`)
	f(`extra_stream_filters={"foo":}`)
}

// TestNewQueryContextAppliesExtraFilters checks the funnel which every traces API
// goes through, so a query built by any of them carries the extra filters.
func TestNewQueryContextAppliesExtraFilters(t *testing.T) {
	q, err := logstorage.ParseQuery("*")
	if err != nil {
		t.Fatalf("cannot parse query: %s", err)
	}
	ef, err := extrafilters.ParseExtraFilters(`{"foo":"bar"}`)
	if err != nil {
		t.Fatalf("cannot parse extra filters: %s", err)
	}

	cp := &CommonParams{
		Query:        q,
		ExtraFilters: []*logstorage.Filter{ef},
	}
	_ = cp.NewQueryContext(context.Background())

	if !strings.Contains(cp.Query.String(), "foo:=bar") {
		t.Fatalf("the extra filter is missing from the query; got %q", cp.Query.String())
	}
}
