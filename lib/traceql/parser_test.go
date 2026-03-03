package traceql

import (
	"fmt"
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

func Test_parseQuery(t *testing.T) {
	f := func(s string) {
		t.Helper()
		lex := newLexer(s, 0)
		q, err := parseQuery(lex)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(q.String())
	}

	//f(`{ac.name=~".*frontend.*" && name = "POST /api/orders"} || {ac.nn="sdf"}|| {ac.nnf="sdfasf"}`)
	//f(`{ac.name=~".*frontend.*" && name = "POST /api/orders"} && {ac.nn="sdf"}`)
	//f(`({ac.name=~".*frontend.*" && name = "POST /api/orders"} || {ac.name=~".*frontend.*" && name = "POST /api/orders"}) && ({ac.name=~".*frontend.*" && name = "POST /api/orders"} || {ac.name=~".*frontend.*" && name = "POST /api/orders"}) `)
	//f(`(({a=b && a=c} || {a=d}) && {a=e})`)
	//f(`{nestedSetParent<0 && true}`)
	//f(`{nestedSetParent<0 && true && status=error}`)
	//f(`({nestedSetParent<0 && {true}}) && ({status=error})`)
	//f(`{} && {nestedSetParent<0 && true && "span.app.ads.ad_request_type" != "nil"}`)
	//f(`{} && {nestedSetParent<0 && ({true}) && "span.app.ads.ad_request_type" != "nil"}`)
	//f(`{ span.http.request_content_length > "10 * 1024 * 1024" }`)
	//f(`{ span.http.request_content_length > 10} | select(span.http.request_content_length) | by(span.http.request_content_length, span.http.request_content_length2) | sum(other_field) > 2m`)
	f(`{(a=b && c=d && c=d)}`)
}

func TestGetTraceDurationFilters(t *testing.T) {
	f := func(s string) {
		t.Helper()
		lex := newLexer(s, 0)
		q, err := parseQuery(lex)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(q.String())
	}

	f(`{(a=b && c=d && c=d)}`)
	f(`{(traceDuration=10ms && c=d && c=d)}`)
	f(`{(traceDuration>10ms && c=d && c=d)}`)
	f(`{(traceDuration>=10ms && traceDuration<1s && c=d)}`)
}
