package traceql

import (
	"testing"
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
