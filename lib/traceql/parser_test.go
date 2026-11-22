package traceql

import (
	"context"
	"sort"
	"strings"
	"sync"
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
	f(`{ac.name=~".*frontend.*" && name = "POST /api/orders"} && {ac.nn="sdf"} || {ac.nn="sdf"}`, `ac.name:~".*frontend.*" and {name="POST /api/orders"} and ac.nn:=sdf or ac.nn:=sdf`)
	f(`(({a=b && a=c} || {a=d}) && {a=e})`, `(a:=b and a:=c or a:=d) and a:=e`)
	f(`{nestedSetParent<0 && true}`, `parent_span_id:="" and *`)
	f(`{nestedSetParent<0 && true && status=error}`, `parent_span_id:="" and * and status_code:=2`)
	f(`{} && {nestedSetParent<0 && true && "span.app.ads.ad_request_type" != "nil"}`, `* and parent_span_id:="" and * and "span_attr:app.ads.ad_request_type":*`)
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
	f(`{trace:duration=10ms}`, `* | join by (trace_id) ({trace_id_idx_stream!=""}  AND duration := 10000000  | fields trace_id_idx  | rename trace_id_idx as trace_id) inner`)
	f(`{trace:duration>10ms && c=d}`, `* and c:=d | join by (trace_id) ({trace_id_idx_stream!=""}  AND duration :> 10000000  | fields trace_id_idx  | rename trace_id_idx as trace_id) inner`)
	f(`{trace:duration>10ms || rootName = "health"}`,
		`trace_id:in({trace_id_idx_stream!=""} AND duration:>10000000 | fields trace_id_idx) or trace_id:in({resource_attr:service.name!=""} AND parent_span_id:="" AND {name=health} | fields trace_id)`)
}

func TestTempoIntrinsicFilters(t *testing.T) {
	f := func(s, expect string) {
		t.Helper()
		q, err := ParseQuery(s)
		if err != nil {
			t.Fatalf("ParseQuery(%q): %v", s, err)
		}
		got := q.String()
		if got != expect {
			t.Fatalf("ParseQuery(%q):\n  got:  %q\n  want: %q", s, got, expect)
		}
		full := `{resource_attr:service.name!=""} AND ` + got
		if _, err := logstorage.ParseQuery(full); err != nil {
			t.Fatalf("logstorage.ParseQuery(%q): %v", full, err)
		}
	}

	// span:* intrinsics map to the same VictoriaTraces fields as unprefixed intrinsics.
	f(`{span:name = "query"}`, `{name=query}`)
	f(`{span:status = "error"}`, `status_code:=2`)
	f(`{span:status =~ "ok|error"}`, `status_code:~"1|2"`)
	f(`{span:kind = "server"}`, `kind:=2`)
	f(`{span:duration > 10ms}`, `duration:>10000000`)
	f(`{span:id = "0a0b"}`, `span_id:=0a0b`)
	f(`{span:parentID = "0c0d"}`, `parent_span_id:=0c0d`)
	f(`{span:statusMessage = "boom"}`, `status_message:=boom`)

	// trace-root intrinsics query the trace's root span (parent_span_id="").
	f(`{rootServiceName = "demo"}`,
		`trace_id:in({resource_attr:service.name!=""} AND parent_span_id:="" AND {"resource_attr:service.name"=demo} | fields trace_id)`)
	f(`{trace:rootService = "demo"}`,
		`trace_id:in({resource_attr:service.name!=""} AND parent_span_id:="" AND {"resource_attr:service.name"=demo} | fields trace_id)`)
	f(`{rootName = "query"}`,
		`trace_id:in({resource_attr:service.name!=""} AND parent_span_id:="" AND {name=query} | fields trace_id)`)
	f(`{trace:rootName = "query"}`,
		`trace_id:in({resource_attr:service.name!=""} AND parent_span_id:="" AND {name=query} | fields trace_id)`)
	f(`{span:name = "select" && trace:rootService = "demo"}`,
		`{name="select"} and trace_id:in({resource_attr:service.name!=""} AND parent_span_id:="" AND {"resource_attr:service.name"=demo} | fields trace_id)`)
}

// TestTempoIntrinsicFiltersAgainstStorage verifies that the LogsQL queries produced
// for span:name, span:status, trace:rootService, rootServiceName, and rootName
// match the expected trace spans when executed against real logstorage.Storage.
func TestTempoIntrinsicFiltersAgainstStorage(t *testing.T) {
	storage := logstorage.MustOpenStorage(t.TempDir(), &logstorage.StorageConfig{})
	t.Cleanup(storage.MustClose)

	ts := time.Now().Add(-2 * time.Minute).UnixNano()
	streamFields := []string{otelpb.ResourceAttrServiceName, otelpb.NameField}
	lr := logstorage.GetLogRows(streamFields, nil, nil, nil, "-")
	defer logstorage.PutLogRows(lr)

	addSpan := func(traceID, spanID, parentSpanID, service, name, statusCode string) {
		lr.MustAdd(logstorage.TenantID{}, ts, []logstorage.Field{
			{Name: "_msg", Value: "-"},
			{Name: otelpb.TraceIDField, Value: traceID},
			{Name: otelpb.SpanIDField, Value: spanID},
			{Name: otelpb.ParentSpanIDField, Value: parentSpanID},
			{Name: otelpb.ResourceAttrServiceName, Value: service},
			{Name: otelpb.NameField, Value: name},
			{Name: otelpb.StatusCodeField, Value: statusCode},
		}, -1)
	}

	// Trace 1: root "demo"/"query" (error, 500ms) + child "db"/"select" (ok)
	addSpan("trace-demo", "span-1", "", "demo", "query", "2")
	addSpan("trace-demo", "span-2", "span-1", "db", "select", "1")
	// Trace 2: root "other"/"health" (ok, 5ms)
	addSpan("trace-other", "span-3", "", "other", "health", "1")
	storage.MustAddRows(lr)

	idxLR := logstorage.GetLogRows([]string{otelpb.TraceIDIndexStreamName}, nil, nil, nil, "-")
	defer logstorage.PutLogRows(idxLR)
	addIdx := func(part, traceID, duration string) {
		idxLR.MustAdd(logstorage.TenantID{}, ts, []logstorage.Field{
			{Name: "_msg", Value: "-"},
			{Name: otelpb.TraceIDIndexStreamName, Value: part},
			{Name: otelpb.TraceIDIndexFieldName, Value: traceID},
			{Name: otelpb.TraceIDIndexDuration, Value: duration},
		}, -1)
	}
	addIdx("0", "trace-demo", "500000000")
	addIdx("1", "trace-other", "5000000")
	storage.MustAddRows(idxLR)
	storage.DebugFlush()

	runSpanIDs := func(traceQL string) []string {
		t.Helper()
		fq, err := ParseQuery(traceQL)
		if err != nil {
			t.Fatalf("ParseQuery(%q): %v", traceQL, err)
		}
		qStr := `{resource_attr:service.name!=""} AND ` + fq.String()
		lq, err := logstorage.ParseQueryAtTimestamp(qStr, time.Now().UnixNano())
		if err != nil {
			t.Fatalf("logstorage.ParseQuery(%q): %v", qStr, err)
		}
		var qs logstorage.QueryStats
		qctx := logstorage.NewQueryContext(context.Background(), &qs, []logstorage.TenantID{{}}, lq, false, nil)
		var mu sync.Mutex
		var got []string
		if err := storage.RunQuery(qctx, func(_ uint, db *logstorage.DataBlock) {
			for _, col := range db.GetColumns(false) {
				if col.Name == otelpb.SpanIDField {
					mu.Lock()
					for _, v := range col.Values {
						got = append(got, strings.Clone(v))
					}
					mu.Unlock()
				}
			}
		}); err != nil {
			t.Fatalf("storage.RunQuery(%q): %v", qStr, err)
		}
		sort.Strings(got)
		return got
	}

	cases := []struct {
		traceQL string
		want    []string
	}{
		{`{span:name = "query"}`, []string{"span-1"}},
		{`{span:status = "error"}`, []string{"span-1"}},
		{`{rootServiceName = "demo"}`, []string{"span-1", "span-2"}},
		{`{trace:rootService = "demo"}`, []string{"span-1", "span-2"}},
		{`{rootName = "query"}`, []string{"span-1", "span-2"}},
		{`{span:name = "select" && trace:rootService = "demo"}`, []string{"span-2"}},
		{`{rootServiceName != "demo"}`, []string{"span-3"}},
		{`{rootServiceName = "demo" || rootName = "health"}`, []string{"span-1", "span-2", "span-3"}},
		{`{trace:duration > 100ms || rootName = "health"}`, []string{"span-1", "span-2", "span-3"}},
		{`{trace:duration > 100ms && span:name = "health"}`, nil},
	}
	for _, tc := range cases {
		got := runSpanIDs(tc.traceQL)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: got %v, want %v", tc.traceQL, got, tc.want)
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Fatalf("%s: got %v, want %v", tc.traceQL, got, tc.want)
			}
		}
	}
}
