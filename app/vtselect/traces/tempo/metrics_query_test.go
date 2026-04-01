package tempo

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTranslateMetricsQuery(t *testing.T) {
	ts := time.Now().UnixNano()

	f := func(input, expectedContains string) {
		t.Helper()
		tr, err := translateMetricsQuery(input, ts)
		if err != nil {
			t.Fatalf("translateMetricsQuery(%q): %s", input, err)
		}
		if !strings.Contains(tr.baseQuery, expectedContains) {
			t.Fatalf("translateMetricsQuery(%q) = %q; expected to contain %q", input, tr.baseQuery, expectedContains)
		}
	}

	// rate()
	f(`{} | rate()`, `rate() as value`)

	// count_over_time()
	f(`{} | count_over_time()`, `count() as value`)

	// *_over_time with field
	f(`{} | min_over_time(duration)`, `min(duration) as value`)
	f(`{} | max_over_time(duration)`, `max(duration) as value`)
	f(`{} | avg_over_time(duration)`, `avg(duration) as value`)
	f(`{} | sum_over_time(duration)`, `sum(duration) as value`)

	// With filter
	f(`{resource.service.name = "frontend"} | rate()`, `rate() as value`)

	// Field name mapping in *_over_time
	f(`{} | avg_over_time(span.http.response_content_length)`, `avg("span_attr:http.response_content_length") as value`)
}

func TestTranslateMetricsQueryWithBy(t *testing.T) {
	ts := time.Now().UnixNano()

	tr, err := translateMetricsQuery(`{} | rate() | by(resource.service.name)`, ts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !strings.Contains(tr.baseQuery, `by ("resource_attr:service.name")`) {
		t.Fatalf("expected by clause with mapped field; got %q", tr.baseQuery)
	}
	if !strings.Contains(tr.baseQuery, `rate() as value`) {
		t.Fatalf("expected rate() stats; got %q", tr.baseQuery)
	}
	if len(tr.byFields) != 1 || tr.byFields[0] != "resource_attr:service.name" {
		t.Fatalf("unexpected byFields; got %v", tr.byFields)
	}
}

func TestTranslateMetricsQueryWithMultipleByFields(t *testing.T) {
	ts := time.Now().UnixNano()

	tr, err := translateMetricsQuery(`{} | count_over_time() | by(resource.service.name, span.http.method)`, ts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !strings.Contains(tr.baseQuery, `"resource_attr:service.name"`) {
		t.Fatalf("expected resource_attr field in by clause; got %q", tr.baseQuery)
	}
	if !strings.Contains(tr.baseQuery, `"span_attr:http.method"`) {
		t.Fatalf("expected span_attr field in by clause; got %q", tr.baseQuery)
	}
	if len(tr.byFields) != 2 {
		t.Fatalf("unexpected byFields count; got %d; want 2", len(tr.byFields))
	}
}

func TestTranslateMetricsQueryCompare(t *testing.T) {
	ts := time.Now().UnixNano()

	tr, err := translateMetricsQuery(`{} | compare({duration >= 500ms}, 10)`, ts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !tr.isCompare {
		t.Fatal("expected isCompare=true")
	}
	if tr.baseFilter != "*" {
		t.Fatalf("expected baseFilter=*; got %q", tr.baseFilter)
	}
	if !strings.Contains(tr.compareFilter, "duration") {
		t.Fatalf("expected duration in compareFilter; got %q", tr.compareFilter)
	}
	if tr.topN != 10 {
		t.Fatalf("expected topN=10; got %d", tr.topN)
	}
}

func TestTranslateMetricsQueryCompareWithTimestamps(t *testing.T) {
	ts := time.Now().UnixNano()

	tr, err := translateMetricsQuery(`{} | compare({duration >= 6s}, 10, 1775053673000000000, 1775054024000000000)`, ts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !tr.isCompare {
		t.Fatal("expected isCompare=true")
	}
	if tr.topN != 10 {
		t.Fatalf("expected topN=10; got %d", tr.topN)
	}
	if tr.selectionStartNs != 1775053673000000000 {
		t.Fatalf("unexpected selectionStartNs; got %d", tr.selectionStartNs)
	}
	if tr.selectionEndNs != 1775054024000000000 {
		t.Fatalf("unexpected selectionEndNs; got %d", tr.selectionEndNs)
	}
}

func TestTranslateMetricsQueryErrors(t *testing.T) {
	ts := time.Now().UnixNano()

	// Non-metrics query should fail.
	_, err := translateMetricsQuery(`{resource.service.name = "frontend"}`, ts)
	if err == nil {
		t.Fatal("expected error for non-metrics query")
	}

	// Invalid syntax should fail.
	_, err = translateMetricsQuery(`{invalid`, ts)
	if err == nil {
		t.Fatal("expected error for invalid query")
	}
}

func TestParseMetricsQueryRangeParams(t *testing.T) {
	// Test with minimal params.
	r := newTestRequest("q=%7B%7D+%7C+rate()&step=60s")
	p, err := parseMetricsQueryRangeParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if p.q != "{} | rate()" {
		t.Fatalf("unexpected q; got %q", p.q)
	}
	if p.step != int64(60*time.Second) {
		t.Fatalf("unexpected step; got %d; want %d", p.step, int64(60*time.Second))
	}

	// Test missing q.
	r = newTestRequest("step=60s")
	_, err = parseMetricsQueryRangeParams(r)
	if err == nil {
		t.Fatal("expected error for missing q parameter")
	}

	// Test auto step calculation.
	r = newTestRequest("q=%7B%7D+%7C+rate()&start=1700000000&end=1700003600")
	p, err = parseMetricsQueryRangeParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if p.step <= 0 {
		t.Fatalf("expected positive step; got %d", p.step)
	}
}

func newTestRequest(query string) *http.Request {
	r, _ := http.NewRequest("GET", "/api/metrics/query_range?"+query, nil)
	return r
}
