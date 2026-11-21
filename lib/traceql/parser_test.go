package traceql

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

func TestLexer(t *testing.T) {
	f := func(s string, tokensExpected []string) {
		t.Helper()
		lex := newLexer(s, 0)
		for _, tokenExpected := range tokensExpected {
			if lex.token != tokenExpected {
				t.Fatalf("unexpected token; got %q; want %q", lex.token, tokenExpected)
			}
			lex.nextToken()
		}
		if lex.token != "" {
			t.Fatalf("unexpected tail token: %q", lex.token)
		}
	}

	//f("", nil)
	//f("  ", nil)
	//f("foo", []string{"foo"})
	//f("тест123", []string{"тест123"})
	//f("foo:bar", []string{"foo", ":", "bar"})
	//f(` re   (  "тест(\":"  )  `, []string{"re", "(", `тест(":`, ")"})
	//f(" `foo, bar`* AND baz:(abc or 'd\\'\"ЙЦУК `'*)", []string{"foo, bar", "*", "AND", "baz", ":", "(", "abc", "or", `d'"ЙЦУК ` + "`", "*", ")"})
	//f(`{foo="bar",a=~"baz", b != 'cd',"d,}a"!~abc} def`,
	//	[]string{"{", "foo", "=", "bar", ",", "a", "=~", "baz", ",", "b", "!=", "cd", ",", "d,}a", "!~", "abc", "}", "def"})
	//f(`_stream:{foo="bar",a=~"baz", b != 'cd',"d,}a"!~abc}`,
	//	[]string{"_stream", ":", "{", "foo", "=", "bar", ",", "a", "=~", "baz", ",", "b", "!=", "cd", ",", "d,}a", "!~", "abc", "}"})
	//
	//f(`foo:~*`, []string{"foo", ":", "~", "*"})

	// TraceQL lexer
	f(`a.n`, []string{"a.n"})
	f(`{ac.name >= "frontend"}`, []string{"{", "ac.name", ">=", "frontend", "}"})
	f(`{"ac.name" = "frontend"}`, []string{"{", "ac.name", "=", "frontend", "}"})
	f(`{a && b}`, []string{"{", "a", "&&", "b", "}"})
	f(`{a &>> b}`, []string{"{", "a", "&>>", "b", "}"})
	f(`{a &~ b}`, []string{"{", "a", "&~", "b", "}"})
	f(`{a || b} | c`, []string{"{", "a", "||", "b", "}", "|", "c"})

	f(`{ resource.cloud.region = "us-east-1" } && { resource.cloud.region = "us-west-1" }`,
		[]string{"{", "resource.cloud.region", "=", "us-east-1", "}", "&&", "{", "resource.cloud.region", "=", "us-west-1", "}"})

	f(`{ a } | count() > 2`, []string{"{", "a", "}", "|", "count", "(", ")", ">", "2"})

	// pipe
	f(`select(*)`, []string{"select", "(", "*", ")"})
}

// TestParseQuery test the query conversion from TraceQL to LogsQL.
func TestParseQuery(t *testing.T) {
	f := func(s, expect string) {
		t.Helper()
		lex := newLexer(s, 0)
		q, err := parseQuery(lex)
		if err != nil {
			t.Fatal(err)
		}
		if q.String() != expect {
			t.Fatalf("unexpected query; got %q; want %q", q.String(), expect)
		}
	}

	// normal queries
	f(`{ac.name=~".*frontend.*" && name = "POST /api/orders"} && {ac.nn="sdf"} || {ac.nn="sdf"}`,
		`((ac.name:~".*frontend.*" and {name="POST /api/orders"}) or (ac.nn:=sdf)) and trace_id:in({resource_attr:service.name!=""} AND (ac.name:~".*frontend.*" and {name="POST /api/orders"}) | uniq by (trace_id)) and trace_id:in({resource_attr:service.name!=""} AND (ac.nn:=sdf) | uniq by (trace_id)) or ac.nn:=sdf`)
	f(`(({a=b && a=c} || {a=d}) && {a=e})`,
		`((a:=b and a:=c or a:=d) or (a:=e)) and trace_id:in({resource_attr:service.name!=""} AND (a:=b and a:=c or a:=d) | uniq by (trace_id)) and trace_id:in({resource_attr:service.name!=""} AND (a:=e) | uniq by (trace_id))`)
	f(`{nestedSetParent<0 && true}`, `parent_span_id:="" and *`)
	f(`{nestedSetParent<0 && true && status=error}`, `parent_span_id:="" and * and status_code:=2`)
	f(`{} && {nestedSetParent<0 && true && "span.app.ads.ad_request_type" != "nil"}`, `parent_span_id:="" and * and "span_attr:app.ads.ad_request_type":*`)
	f(`{ span.http.request_content_length > 10} | select(span.http.request_content_length) | by(span.http.request_content_length, span.http.request_content_length2) | sum(other_field) > 2m`, `"span_attr:http.request_content_length":>10 | select(span.http.request_content_length) | by(span.http.request_content_length, span.http.request_content_length2) | sum(other_field) > 2m`)
	f(`{(a=b && c=d && c=d)}`, `a:=b and c:=d and c:=d`)
	f(`{span.http.status_code = "200"}`, `"span_attr:http.status_code":=200`)
	f(`{status = "error"}`, `status_code:=2`)
	f(`{status = "unset"}`, `status_code:=0`)
	f(`{resource.service.name = "my_service"}`, `{"resource_attr:service.name"=my_service}`)
	f(`{kind = "internal"}`, `kind:=1`)
	f(`{kind = "client"}`, `kind:=3`)

	// span.* attribute regex. The internal field name "span_attr:http.status_code"
	// contains ':' so quoteFieldNameIfNeeded wraps it in quotes.
	f(`{span.http.status_code =~ "^[1-5].."}`, `"span_attr:http.status_code":~"^[1-5].."`)

	// resource.service.name is a stream field but the stream-filter shortcut
	// only fires for =/!=, so regex falls through to the regular branch.
	f(`{resource.service.name !~ "test-.*"}`, `{"resource_attr:service.name"!~"test-.*"}`)
	f(`{resource.service.name =~ "test-.*"}`, `{"resource_attr:service.name"=~"test-.*"}`)

	// status field: name -> code rewrite at word boundaries.
	f(`{status =~ "ok|error"}`, `status_code:~"1|2"`)
	f(`{status =~ "^(ok|error)$"}`, `status_code:~"^(1|2)$"`)
	f(`{status !~ "unset"}`, `status_code:!~"0"`)

	// status field: case-insensitive name match.
	f(`{status =~ "OK|Error"}`, `status_code:~"1|2"`)

	// span kind field: name -> code rewrite at word boundaries.
	f(`{kind =~ "internal|server"}`, `kind:~"1|2"`)
	f(`{kind =~ "^(consumer|producer)$"}`, `kind:~"^(5|4)$"`)
	f(`{kind !~ "UNSPECIFIED"}`, `kind:!~"0"`)

	// span kind field: case-insensitive name match.
	f(`{kind =~ "UNSPECIFIED|Server|consumer"}`, `kind:~"0|2|5"`)

	// Regex match operator `=~` must become LogsQL `~` (not `=~`).
	f(`{resource.host.name =~ "kimi-k2-a.*"}`, `"resource_attr:host.name":~"kimi-k2-a.*"`)
}

// TestParseQueryInvalid asserts that malformed inputs return an error
// rather than crashing the parser via unbounded recursion or non-progressing
// loops. Previously, queries like "{&&}" caused parseFilterGeneric to recurse
// into parseFilterAnd without consuming the operator token, overflowing the
// goroutine stack.
func TestParseQueryInvalid(t *testing.T) {
	f := func(s string) {
		t.Helper()
		_, err := ParseQuery(s)
		if err == nil {
			t.Fatalf("expected error for malformed query %q, got nil", s)
		}
	}

	// Orphaned binary operators inside a filter — previously recursed forever.
	f(`{&&}`)
	f(`{||}`)
	f(`{and}`)
	f(`{or}`)
	f(`{&& a}`)
	f(`{|| a}`)
	f(`{a &&}`)
	f(`{a ||}`)
	f(`{a && && b}`)
	f(`{a || || b}`)
	f(`{a && (b || )}`)
	f(`{(a && && b)}`)

	// Unary operators are not implemented — previously returned (nil, nil)
	// silently, spinning the caller's loop without consuming the token.
	f(`{not}`)
	f(`{!}`)
	f(`{-}`)
	f(`{not a}`)
}

func TestGetTraceDurationFilters(t *testing.T) {
	f := func(s, expect string) {
		t.Helper()
		lex := newLexer(s, 0)
		q, err := parseQuery(lex)
		if err != nil {
			t.Fatal(err)
		}
		if q.String() != expect {
			t.Fatalf("for %q: got %q, want %q", s, q.String(), expect)
		}
	}

	f(`{(a=b && c=d && c=d)}`, `a:=b and c:=d and c:=d`)
	f(`{(traceDuration=10ms && c=d && c=d)}`, `* and c:=d and c:=d | join by (trace_id) ({trace_id_idx_stream!=""}  AND duration := 10000000  | fields trace_id_idx  | rename trace_id_idx as trace_id) inner`)
	f(`{(traceDuration>10ms && c=d && c=d)}`, `* and c:=d and c:=d | join by (trace_id) ({trace_id_idx_stream!=""}  AND duration :> 10000000  | fields trace_id_idx  | rename trace_id_idx as trace_id) inner`)
	f(`{(traceDuration>=10ms && traceDuration<1s && c=d)}`, `* and * and c:=d | join by (trace_id) ({trace_id_idx_stream!=""}  AND duration :>= 10000000 AND duration :< 1000000000  | fields trace_id_idx  | rename trace_id_idx as trace_id) inner`)
}

func TestSpansetAndQueryExecution(t *testing.T) {
	cfg := &logstorage.StorageConfig{
		Retention: 7 * 24 * time.Hour,
	}
	s := logstorage.MustOpenStorage(t.TempDir(), cfg)
	defer s.MustClose()

	now := time.Now()
	tsParent := now.Add(-2 * time.Minute).UnixNano()
	tsChild := now.Add(-1 * time.Minute).UnixNano()

	lr := logstorage.GetLogRows([]string{otelpb.ResourceAttrServiceName, otelpb.NameField}, nil, nil, nil, "")
	addSpan := func(ts int64, traceID, spanID, parentSpanID, serviceName, spanName string, extra ...logstorage.Field) {
		fields := []logstorage.Field{
			{Name: otelpb.TraceIDField, Value: traceID},
			{Name: otelpb.SpanIDField, Value: spanID},
			{Name: otelpb.ParentSpanIDField, Value: parentSpanID},
			{Name: otelpb.ResourceAttrServiceName, Value: serviceName},
			{Name: otelpb.NameField, Value: spanName},
		}
		fields = append(fields, extra...)
		lr.MustAdd(logstorage.TenantID{}, ts, fields, -1)
	}

	// Trace c1: parent has a=x, child has b=y (issue #324 repro).
	addSpan(tsParent, "c1", "s1", "", "demo", "parent", logstorage.Field{Name: "span_attr:a", Value: "x"})
	addSpan(tsChild, "c1", "s2", "s1", "demo", "child", logstorage.Field{Name: "span_attr:b", Value: "y"})

	// Trace c2: only has a=x (must NOT match `{ span.a = "x" } && { span.b = "y" }`).
	addSpan(tsParent, "c2", "s3", "", "demo", "only-a", logstorage.Field{Name: "span_attr:a", Value: "x"})

	// Trace c3: parent has c=z in another service.
	addSpan(tsParent, "c3", "s4", "", "other-svc", "root-c", logstorage.Field{Name: "span_attr:c", Value: "z"})
	addSpan(tsChild, "c3", "s5", "s4", "demo", "child-a", logstorage.Field{Name: "span_attr:a", Value: "x"})

	s.MustAddRows(lr)
	logstorage.PutLogRows(lr)
	s.DebugFlush()

	runTraceQL := func(traceQL string) []string {
		t.Helper()
		q, err := ParseQueryAtTimestamp(traceQL, now.UnixNano())
		if err != nil {
			t.Fatalf("ParseQuery(%q): %v", traceQL, err)
		}
		qStr := `{resource_attr:service.name!=""} AND ` + q.String() + ` | uniq by (trace_id)`
		lq, err := logstorage.ParseQueryAtTimestamp(qStr, now.UnixNano())
		if err != nil {
			t.Fatalf("logstorage.ParseQuery(%q): %v", qStr, err)
		}
		lq.AddTimeFilter(now.Add(-10*time.Minute).UnixNano(), now.UnixNano())
		qctx := logstorage.NewQueryContext(context.Background(), &logstorage.QueryStats{}, []logstorage.TenantID{{}}, lq, false, nil)
		var got []string
		if err := s.RunQuery(qctx, func(_ uint, db *logstorage.DataBlock) {
			for _, col := range db.GetColumns(false) {
				if col.Name == otelpb.TraceIDField {
					for _, v := range col.Values {
						got = append(got, strings.Clone(v))
					}
				}
			}
		}); err != nil {
			t.Fatalf("RunQuery(%q): %v", qStr, err)
		}
		slices.Sort(got)
		return slices.Compact(got)
	}

	assertTraces := func(traceQL string, want ...string) {
		t.Helper()
		got := runTraceQL(traceQL)
		if !slices.Equal(got, want) {
			t.Fatalf("query %q: got traces %v, want %v", traceQL, got, want)
		}
	}

	assertTraces(`{ span.a = "x" }`, "c1", "c2", "c3")
	assertTraces(`{ span.a = "x" } && { span.b = "y" }`, "c1")
	assertTraces(`{ span.b = "y" } && { span.a = "x" }`, "c1")
	assertTraces(`{ resource.service.name = "other-svc" } && { span.a = "x" } && { span.c = "z" }`, "c3")
	assertTraces(`({ resource.service.name = "other-svc" } && { span.a = "x" }) && { span.c = "z" }`, "c3")
	assertTraces(`({ span.a = "x" } && { span.b = "y" }) || { span.c = "z" }`, "c1", "c3")
	assertTraces(`({ span.b = "y" } || { span.c = "z" }) && { span.a = "x" }`, "c1", "c3")
	assertTraces(`{ resource.service.name = "other-svc" } && { span.a = "x" }`, "c3")
	assertTraces(`{ span.a = "x" && span.b = "y" }`) // single span must have both -> none

	// Verify GetTraceList step-2 query syntax returns both matching spans (parent + child) plus root spans.
	q, err := ParseQueryAtTimestamp(`{ span.a = "x" } && { span.b = "y" }`, now.UnixNano())
	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	step2Str := `{resource_attr:service.name!=""} AND (` + q.FilterString() + ` OR parent_span_id:="") AND trace_id:in(c1)`
	step2Q, err := logstorage.ParseQueryAtTimestamp(step2Str, now.UnixNano())
	if err != nil {
		t.Fatalf("step-2 logstorage.ParseQuery(%q): %v", step2Str, err)
	}
	step2Q.AddTimeFilter(now.Add(-10*time.Minute).UnixNano(), now.UnixNano())
	qctx := logstorage.NewQueryContext(context.Background(), &logstorage.QueryStats{}, []logstorage.TenantID{{}}, step2Q, false, nil)
	var spanIDs []string
	if err := s.RunQuery(qctx, func(_ uint, db *logstorage.DataBlock) {
		for _, col := range db.GetColumns(false) {
			if col.Name == otelpb.SpanIDField {
				for _, v := range col.Values {
					spanIDs = append(spanIDs, strings.Clone(v))
				}
			}
		}
	}); err != nil {
		t.Fatalf("step-2 RunQuery: %v", err)
	}
	slices.Sort(spanIDs)
	if !slices.Equal(spanIDs, []string{"s1", "s2"}) {
		t.Fatalf("step-2 span IDs: got %v, want [s1 s2]", spanIDs)
	}
}
