package traceql

import (
	"testing"
)

func TestParsePipeRate(t *testing.T) {
	f := func(input, expected string) {
		t.Helper()
		q, err := ParseQuery(input)
		if err != nil {
			t.Fatalf("cannot parse %q: %s", input, err)
		}
		if !q.IsMetricsQuery() {
			t.Fatalf("expected metrics query for %q", input)
		}
		funcName, fieldName, _, _, _, err := q.MetricsComponents() //nolint:dogsled
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if funcName != expected {
			t.Fatalf("unexpected funcName; got %q; want %q", funcName, expected)
		}
		if fieldName != "" {
			t.Fatalf("unexpected fieldName; got %q; want empty", fieldName)
		}
	}

	f(`{} | rate()`, "rate")
	f(`{resource.service.name = "frontend"} | rate()`, "rate")
}

func TestParsePipeRateWithComparison(t *testing.T) {
	q, err := ParseQuery(`{} | rate() > 5`)
	if err != nil {
		t.Fatalf("cannot parse: %s", err)
	}
	if !q.IsMetricsQuery() {
		t.Fatal("expected metrics query")
	}
	funcName, _, _, _, _, err := q.MetricsComponents()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if funcName != "rate" {
		t.Fatalf("unexpected funcName; got %q; want %q", funcName, "rate")
	}
}

func TestParsePipeOverTime(t *testing.T) {
	f := func(input, expectedFunc, expectedField string) {
		t.Helper()
		q, err := ParseQuery(input)
		if err != nil {
			t.Fatalf("cannot parse %q: %s", input, err)
		}
		if !q.IsMetricsQuery() {
			t.Fatalf("expected metrics query for %q", input)
		}
		funcName, fieldName, _, _, _, err := q.MetricsComponents() //nolint:dogsled
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if funcName != expectedFunc {
			t.Fatalf("unexpected funcName; got %q; want %q", funcName, expectedFunc)
		}
		if fieldName != expectedField {
			t.Fatalf("unexpected fieldName; got %q; want %q", fieldName, expectedField)
		}
	}

	// count_over_time takes no field
	f(`{} | count_over_time()`, "count_over_time", "")

	// *_over_time with field
	f(`{} | min_over_time(duration)`, "min_over_time", "duration")
	f(`{} | max_over_time(duration)`, "max_over_time", "duration")
	f(`{} | avg_over_time(duration)`, "avg_over_time", "duration")
	f(`{} | sum_over_time(duration)`, "sum_over_time", "duration")

	// With filter
	f(`{resource.service.name = "api"} | avg_over_time(duration)`, "avg_over_time", "duration")

	// With dotted field name
	f(`{} | sum_over_time(span.http.response_content_length)`, "sum_over_time", "span.http.response_content_length")
}

func TestParsePipeOverTimeWithBy(t *testing.T) {
	// With explicit | separator
	q, err := ParseQuery(`{} | rate() | by(resource.service.name)`)
	if err != nil {
		t.Fatalf("cannot parse: %s", err)
	}
	_, _, _, _, byFields, err := q.MetricsComponents()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(byFields) != 1 || byFields[0] != "resource.service.name" {
		t.Fatalf("unexpected byFields; got %v; want [resource.service.name]", byFields)
	}

	// Tempo-style: by() without | separator
	q, err = ParseQuery(`{} | rate() by(resource.service.name)`)
	if err != nil {
		t.Fatalf("cannot parse Tempo-style by(): %s", err)
	}
	_, _, _, _, byFields, err = q.MetricsComponents()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(byFields) != 1 || byFields[0] != "resource.service.name" {
		t.Fatalf("unexpected byFields (Tempo-style); got %v; want [resource.service.name]", byFields)
	}
}

func TestParsePipeOverTimeWithMultipleByFields(t *testing.T) {
	q, err := ParseQuery(`{} | count_over_time() | by(resource.service.name, span.http.method)`)
	if err != nil {
		t.Fatalf("cannot parse: %s", err)
	}
	_, _, _, _, byFields, err := q.MetricsComponents()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(byFields) != 2 {
		t.Fatalf("unexpected byFields count; got %d; want 2", len(byFields))
	}
	if byFields[0] != "resource.service.name" {
		t.Fatalf("unexpected byFields[0]; got %q; want %q", byFields[0], "resource.service.name")
	}
	if byFields[1] != "span.http.method" {
		t.Fatalf("unexpected byFields[1]; got %q; want %q", byFields[1], "span.http.method")
	}
}

func TestParsePipeOverTimeErrors(t *testing.T) {
	// count_over_time should not accept a field
	_, err := ParseQuery(`{} | count_over_time(duration)`)
	if err == nil {
		t.Fatal("expected error for count_over_time with field argument")
	}

	// avg_over_time requires a field
	_, err = ParseQuery(`{} | avg_over_time()`)
	if err == nil {
		t.Fatal("expected error for avg_over_time without field argument")
	}

	// missing closing paren
	_, err = ParseQuery(`{} | rate(`)
	if err == nil {
		t.Fatal("expected error for rate with missing closing paren")
	}
}

// TestParsePipeQuantileOverTime covers the single- and multi-quantile forms.
// Tempo accepts `quantile_over_time(field, q1[, q2, ...])`; previously VT only
// parsed the single-quantile form and returned a 400 on the multi form sent by
// the Traces Drilldown plugin.
func TestParsePipeQuantileOverTime(t *testing.T) {
	f := func(input string, wantField string, wantQuantiles []string) {
		t.Helper()
		q, err := ParseQuery(input)
		if err != nil {
			t.Fatalf("cannot parse %q: %s", input, err)
		}
		funcName, fieldName, quantiles, _, _, err := q.MetricsComponents()
		if err != nil {
			t.Fatalf("MetricsComponents(%q): %s", input, err)
		}
		if funcName != "quantile_over_time" {
			t.Fatalf("funcName = %q; want quantile_over_time", funcName)
		}
		if fieldName != wantField {
			t.Fatalf("fieldName = %q; want %q", fieldName, wantField)
		}
		if len(quantiles) != len(wantQuantiles) {
			t.Fatalf("quantiles = %v; want %v", quantiles, wantQuantiles)
		}
		for i, q := range quantiles {
			if q != wantQuantiles[i] {
				t.Fatalf("quantiles[%d] = %q; want %q", i, q, wantQuantiles[i])
			}
		}
	}

	// Single quantile.
	f(`{} | quantile_over_time(duration, 0.9)`, "duration", []string{"0.9"})

	// Multi quantile — the failing case from Traces Drilldown.
	f(`{} | quantile_over_time(duration, 0.5, 0.9)`, "duration", []string{"0.5", "0.9"})
	f(`{} | quantile_over_time(duration, 0.5, 0.9, 0.99)`, "duration", []string{"0.5", "0.9", "0.99"})

	// Multi quantile without spaces (matches the URL-encoded form Drilldown sends).
	f(`{} | quantile_over_time(duration,0.5,0.9)`, "duration", []string{"0.5", "0.9"})
}

func TestParsePipeQuantileOverTimeErrors(t *testing.T) {
	// No quantile.
	if _, err := ParseQuery(`{} | quantile_over_time(duration)`); err == nil {
		t.Fatal("expected error for quantile_over_time without quantile")
	}
	// Empty quantile after comma.
	if _, err := ParseQuery(`{} | quantile_over_time(duration, )`); err == nil {
		t.Fatal("expected error for empty quantile")
	}
	// Trailing comma.
	if _, err := ParseQuery(`{} | quantile_over_time(duration, 0.5,)`); err == nil {
		t.Fatal("expected error for trailing comma")
	}
	// Missing field.
	if _, err := ParseQuery(`{} | quantile_over_time()`); err == nil {
		t.Fatal("expected error for quantile_over_time without field")
	}
}

// TestPipeQuantileOverTimeString verifies the String() round-trip preserves
// all quantiles.
func TestPipeQuantileOverTimeString(t *testing.T) {
	f := func(input, want string) {
		t.Helper()
		q, err := ParseQuery(input)
		if err != nil {
			t.Fatalf("cannot parse %q: %s", input, err)
		}
		got := q.String()
		if got != want {
			t.Fatalf("String() = %q; want %q", got, want)
		}
	}

	f(`{} | quantile_over_time(duration, 0.9)`, `* | quantile_over_time(duration, 0.9)`)
	f(`{} | quantile_over_time(duration, 0.5, 0.9, 0.99)`, `* | quantile_over_time(duration, 0.5, 0.9, 0.99)`)
}

func TestIsMetricsQueryFalse(t *testing.T) {
	// Regular queries should not be metrics queries.
	q, err := ParseQuery(`{resource.service.name = "frontend"}`)
	if err != nil {
		t.Fatalf("cannot parse: %s", err)
	}
	if q.IsMetricsQuery() {
		t.Fatal("expected non-metrics query")
	}

	// Query with count pipe is not a metrics query.
	q, err = ParseQuery(`{} | count() > 5`)
	if err != nil {
		t.Fatalf("cannot parse: %s", err)
	}
	if q.IsMetricsQuery() {
		t.Fatal("expected non-metrics query for count()")
	}
}

func TestTraceQLFieldToVTField(t *testing.T) {
	f := func(input, expected string) {
		t.Helper()
		got := TraceQLFieldToVTField(input)
		if got != expected {
			t.Fatalf("TraceQLFieldToVTField(%q); got %q; want %q", input, got, expected)
		}
	}

	f("resource.service.name", "resource_attr:service.name")
	f("span.http.status_code", "span_attr:http.status_code")
	f("status", "status_code")
	f("service.name", "resource_attr:service.name")
	f("duration", "duration")
	f("name", "name")
}

func TestVTFieldToTraceQL(t *testing.T) {
	f := func(input, expected string) {
		t.Helper()
		got := VTFieldToTraceQL(input)
		if got != expected {
			t.Fatalf("VTFieldToTraceQL(%q); got %q; want %q", input, got, expected)
		}
	}

	f("resource_attr:service.name", "resource.service.name")
	f("span_attr:http.status_code", "span.http.status_code")
	f("status_code", "status")
	f("duration", "duration")
	f("name", "name")
}

func TestFieldMappingRoundTrip(t *testing.T) {
	fields := []string{
		"resource.service.name",
		"span.http.status_code",
		"status",
		"duration",
		"name",
	}
	for _, field := range fields {
		vt := TraceQLFieldToVTField(field)
		back := VTFieldToTraceQL(vt)
		if back != field {
			t.Fatalf("round-trip failed for %q: TraceQLFieldToVTField -> %q -> VTFieldToTraceQL -> %q", field, vt, back)
		}
	}
}
