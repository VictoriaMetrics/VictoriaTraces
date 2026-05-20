package tempo

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
)

func TestTranslateMetricsQueryFull(t *testing.T) {
	ts := time.Now().UnixNano()

	f := func(traceQL, expectedLogsQL string) {
		t.Helper()
		tr, err := translateMetricsQuery(traceQL, ts)
		if err != nil {
			t.Fatalf("translateMetricsQuery(%q): %s", traceQL, err)
		}
		if tr.baseQuery != expectedLogsQL {
			t.Fatalf("translateMetricsQuery(%q):\n  got:  %q\n  want: %q", traceQL, tr.baseQuery, expectedLogsQL)
		}
		// Verify the generated LogsQL actually parses.
		_, err = logstorage.ParseQueryAtTimestamp(tr.baseQuery, ts)
		if err != nil {
			t.Fatalf("generated LogsQL does not parse: %q: %s", tr.baseQuery, err)
		}
	}

	// Basic rate
	f(`{} | rate()`, `{resource_attr:service.name!=""} AND * | stats rate() as value`)

	// Status value mapping
	f(`{status = error} | rate()`, `{resource_attr:service.name!=""} AND status_code:=2 | stats rate() as value`)

	// Duration filter (100ms = 100000000ns)
	f(`{duration > 100ms} | rate()`, `{resource_attr:service.name!=""} AND duration:>100000000 | stats rate() as value`)

	// Resource attribute filter
	f(`{resource.attr_name = "api"} | rate()`,
		`{resource_attr:service.name!=""} AND "resource_attr:attr_name":=api | stats rate() as value`)

	// Resource attribute (stream field) filter
	f(`{resource.service.name = "api"} | rate()`,
		`{resource_attr:service.name!=""} AND {"resource_attr:service.name"=api} | stats rate() as value`)

	// Span attribute filter
	f(`{span.http.status_code >= 400} | count_over_time()`,
		`{resource_attr:service.name!=""} AND "span_attr:http.status_code":>=400 | stats count() as value`)

	// Name filter
	f(`{name = "http_request"} | rate()`, `{resource_attr:service.name!=""} AND {name=http_request} | stats rate() as value`)

	// nestedSetParent → root spans; intersect with trace ID index via in(subquery)
	f(`{nestedSetParent < 0} | rate()`,
		`{resource_attr:service.name!=""} AND parent_span_id:="" | stats rate() as value`)

	// `attr != nil` → any-value filter; `attr = nil` → empty-value filter.
	f(`{resource.service.name != nil} | rate()`,
		`{resource_attr:service.name!=""} AND "resource_attr:service.name":* | stats rate() as value`)
	f(`{resource.service.name = nil} | rate()`,
		`{resource_attr:service.name!=""} AND "resource_attr:service.name":"" | stats rate() as value`)

	// Grafana sends {true && true}
	f(`{true && true} | rate()`, `{resource_attr:service.name!=""} AND * and * | stats rate() as value`)

	// All aggregation functions
	f(`{} | count_over_time()`, `{resource_attr:service.name!=""} AND * | stats count() as value`)
	f(`{} | min_over_time(duration)`, `{resource_attr:service.name!=""} AND * | stats min(duration) as value`)
	f(`{} | max_over_time(duration)`, `{resource_attr:service.name!=""} AND * | stats max(duration) as value`)
	f(`{} | avg_over_time(duration)`, `{resource_attr:service.name!=""} AND * | stats avg(duration) as value`)
	f(`{} | sum_over_time(duration)`, `{resource_attr:service.name!=""} AND * | stats sum(duration) as value`)
	f(`{} | histogram_over_time(duration)`, `{resource_attr:service.name!=""} AND * | stats histogram(duration) as value`)
	f(`{} | quantile_over_time(duration, 0.9)`, `{resource_attr:service.name!=""} AND * | stats quantile(0.9, duration) as p_0`)
	// Multiple quantiles emit one stats output per quantile so each becomes its own series.
	f(`{} | quantile_over_time(duration, 0.5, 0.9, 0.99)`,
		`{resource_attr:service.name!=""} AND * | stats quantile(0.5, duration) as p_0, quantile(0.9, duration) as p_1, quantile(0.99, duration) as p_2`)

	// Field name mapping in aggregation
	f(`{} | sum_over_time(span.kafka.lag)`, `{resource_attr:service.name!=""} AND * | stats sum("span_attr:kafka.lag") as value`)
	f(`{} | avg_over_time(span.http.response_content_length)`,
		`{resource_attr:service.name!=""} AND * | stats avg("span_attr:http.response_content_length") as value`)
	f(`{} | max_over_time(span.http.status_code)`,
		`{resource_attr:service.name!=""} AND * | stats max("span_attr:http.status_code") as value`)
}

func TestTranslateMetricsQueryByFields(t *testing.T) {
	ts := time.Now().UnixNano()

	f := func(traceQL, expectedLogsQL string) {
		t.Helper()
		tr, err := translateMetricsQuery(traceQL, ts)
		if err != nil {
			t.Fatalf("translateMetricsQuery(%q): %s", traceQL, err)
		}
		if tr.baseQuery != expectedLogsQL {
			t.Fatalf("translateMetricsQuery(%q):\n  got:  %q\n  want: %q", traceQL, tr.baseQuery, expectedLogsQL)
		}
		_, err = logstorage.ParseQueryAtTimestamp(tr.baseQuery, ts)
		if err != nil {
			t.Fatalf("generated LogsQL does not parse: %q: %s", tr.baseQuery, err)
		}
	}

	// Tempo-style by() without | separator
	f(`{} | rate() by(resource.service.name)`,
		`{resource_attr:service.name!=""} AND * | stats by ("resource_attr:service.name") rate() as value`)

	// With explicit | separator
	f(`{} | rate() | by(resource.service.name)`,
		`{resource_attr:service.name!=""} AND * | stats by ("resource_attr:service.name") rate() as value`)

	// Intrinsic field
	f(`{} | rate() | by(name)`,
		`{resource_attr:service.name!=""} AND * | stats by (name) rate() as value`)

	// Status field mapping in by()
	f(`{} | rate() | by(status)`,
		`{resource_attr:service.name!=""} AND * | stats by (status_code) rate() as value`)

	// Multiple by fields
	f(`{} | avg_over_time(duration) | by(resource.service.name, span.http.method)`,
		`{resource_attr:service.name!=""} AND * | stats by ("resource_attr:service.name", "span_attr:http.method") avg(duration) as value`)

	// Quantile with by
	f(`{} | quantile_over_time(duration, 0.5) | by(resource.service.name)`,
		`{resource_attr:service.name!=""} AND * | stats by ("resource_attr:service.name") quantile(0.5, duration) as p_0`)
	// Multi-quantile with by
	f(`{} | quantile_over_time(duration, 0.5, 0.9) | by(resource.service.name)`,
		`{resource_attr:service.name!=""} AND * | stats by ("resource_attr:service.name") quantile(0.5, duration) as p_0, quantile(0.9, duration) as p_1`)

	// Complex: filter + aggregation + by
	f(`{resource.attr_name = "api"} | max_over_time(span.http.status_code) | by(resource.service.name)`,
		`{resource_attr:service.name!=""} AND "resource_attr:attr_name":=api | stats by ("resource_attr:service.name") max("span_attr:http.status_code") as value`)

	// Complex: stream filter
	f(`{resource.service.name = "api"} | max_over_time(span.http.status_code) | by(resource.service.name)`,
		`{resource_attr:service.name!=""} AND {"resource_attr:service.name"=api} | stats by ("resource_attr:service.name") max("span_attr:http.status_code") as value`)

}

func TestTranslateMetricsQueryWithHints(t *testing.T) {
	ts := time.Now().UnixNano()

	// with() hints should be silently ignored — output must match the hint-free version.
	baseRate, err := translateMetricsQuery(`{} | rate()`, ts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	withSample, err := translateMetricsQuery(`{} | rate() with(sample=true)`, ts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if withSample.baseQuery != baseRate.baseQuery {
		t.Fatalf("with(sample=true) changed output:\n  got:  %q\n  want: %q", withSample.baseQuery, baseRate.baseQuery)
	}

	withExemplars, err := translateMetricsQuery(`{} | rate() with(exemplars=0)`, ts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if withExemplars.baseQuery != baseRate.baseQuery {
		t.Fatalf("with(exemplars=0) changed output:\n  got:  %q\n  want: %q", withExemplars.baseQuery, baseRate.baseQuery)
	}

	// with() + by() combined
	baseBy, err := translateMetricsQuery(`{} | rate() | by(resource.service.name)`, ts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	withBy, err := translateMetricsQuery(`{} | rate() by(resource.service.name) with(sample=0.5)`, ts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if withBy.baseQuery != baseBy.baseQuery {
		t.Fatalf("with(sample=0.5) + by() changed output:\n  got:  %q\n  want: %q", withBy.baseQuery, baseBy.baseQuery)
	}
}

func TestTranslateMetricsQueryCompare(t *testing.T) {
	ts := time.Now().UnixNano()

	// Basic compare with status filter
	tr, err := translateMetricsQuery(`{} | compare({status = error}, 10)`, ts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !tr.isCompare {
		t.Fatal("expected isCompare=true")
	}
	if tr.baseFilter != `{resource_attr:service.name!=""} AND *` {
		t.Fatalf("expected baseFilter=*; got %q", tr.baseFilter)
	}
	if !strings.Contains(tr.compareFilter, "status_code") {
		t.Fatalf("expected status_code in compareFilter; got %q", tr.compareFilter)
	}
	if tr.topN != 10 {
		t.Fatalf("expected topN=10; got %d", tr.topN)
	}

	// Compare with non-trivial base filter
	tr, err = translateMetricsQuery(`{resource.service.name = "api"} | compare({duration >= 100ms}, 5)`, ts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !strings.Contains(tr.baseFilter, "resource_attr:service.name") {
		t.Fatalf("expected resource_attr in baseFilter; got %q", tr.baseFilter)
	}
	if !strings.Contains(tr.compareFilter, "duration") {
		t.Fatalf("expected duration in compareFilter; got %q", tr.compareFilter)
	}
	if tr.topN != 5 {
		t.Fatalf("expected topN=5; got %d", tr.topN)
	}

	// Full 4-arg form with timestamps
	tr, err = translateMetricsQuery(`{true && true} | compare({duration >= 6s && duration <= 230s}, 10, 1775053673000000000, 1775054024000000000)`, ts)
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

func TestTranslateMetricsQueryWithComparison(t *testing.T) {
	ts := time.Now().UnixNano()

	// rate() > 5 should parse without error
	tr, err := translateMetricsQuery(`{} | rate() > 5`, ts)
	if err != nil {
		t.Fatalf("unexpected error for rate() > 5: %s", err)
	}
	if tr.baseQuery == "" {
		t.Fatal("expected non-empty baseQuery")
	}

	// count_over_time() >= 100
	tr, err = translateMetricsQuery(`{} | count_over_time() >= 100`, ts)
	if err != nil {
		t.Fatalf("unexpected error for count_over_time() >= 100: %s", err)
	}
	if tr.baseQuery == "" {
		t.Fatal("expected non-empty baseQuery")
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

// TestTranslateMetricsQueryQuantileLabels verifies that quantile_over_time
// queries record a column→label map so the executor can attach a `p` label
// matching Tempo's response shape (one series per quantile, single quantile
// included).
func TestTranslateMetricsQueryQuantileLabels(t *testing.T) {
	ts := time.Now().UnixNano()

	f := func(traceQL string, wantKey string, wantLabels map[string]string) {
		t.Helper()
		tr, err := translateMetricsQuery(traceQL, ts)
		if err != nil {
			t.Fatalf("translateMetricsQuery(%q): %s", traceQL, err)
		}
		if tr.valueColumnLabelKey != wantKey {
			t.Fatalf("translateMetricsQuery(%q): valueColumnLabelKey = %q, want %q", traceQL, tr.valueColumnLabelKey, wantKey)
		}
		if len(tr.valueColumnLabels) != len(wantLabels) {
			t.Fatalf("translateMetricsQuery(%q): valueColumnLabels has %d entries, want %d (got %v)", traceQL, len(tr.valueColumnLabels), len(wantLabels), tr.valueColumnLabels)
		}
		for k, v := range wantLabels {
			if got := tr.valueColumnLabels[k]; got != v {
				t.Fatalf("translateMetricsQuery(%q): valueColumnLabels[%q] = %q, want %q", traceQL, k, got, v)
			}
		}
	}

	// Single quantile still gets a `p` label so the response matches Tempo.
	f(`{} | quantile_over_time(duration, 0.9)`, "p", map[string]string{"p_0": "0.9"})

	// Multiple quantiles produce one column→quantile mapping per quantile.
	f(`{} | quantile_over_time(duration, 0.5, 0.9, 0.99)`, "p", map[string]string{
		"p_0": "0.5",
		"p_1": "0.9",
		"p_2": "0.99",
	})

	// Non-quantile metrics do not carry the column→label map.
	tr, err := translateMetricsQuery(`{} | rate()`, ts)
	if err != nil {
		t.Fatalf("translateMetricsQuery(rate): %s", err)
	}
	if tr.valueColumnLabelKey != "" {
		t.Fatalf("rate() should not set valueColumnLabelKey; got %q", tr.valueColumnLabelKey)
	}
	if len(tr.valueColumnLabels) != 0 {
		t.Fatalf("rate() should have empty valueColumnLabels; got %v", tr.valueColumnLabels)
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
