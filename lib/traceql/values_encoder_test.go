package traceql

import "testing"

func TestTryParseDuration(t *testing.T) {
	f := func(input string, wantNsecs int64, wantOK bool) {
		t.Helper()

		gotNsecs, gotOK := tryParseDuration(input)
		if gotOK != wantOK || gotNsecs != wantNsecs {
			t.Fatalf("tryParseDuration(%q) = (%d, %v), want (%d, %v)",
				input, gotNsecs, gotOK, wantNsecs, wantOK)
		}
	}

	// Valid durations.
	f("5s", 5_000_000_000, true)
	f("-5s", -5_000_000_000, true)
	f("1ms", 1_000_000, true)

	// Invalid durations.
	f("", 0, false)
	f("abc", 0, false)
	f("-", 0, false)
}

func TestParseQueryLoneMinusValue(t *testing.T) {
	lex := newLexer(`{span.foo = "-"}`, 0)
	q, err := parseQuery(lex)
	if err != nil {
		t.Fatalf("cannot parse query: %s", err)
	}

	const want = `"span_attr:foo":="-"`

	if got := q.String(); got != want {
		t.Fatalf("unexpected query: got %q; want %q", got, want)
	}
}
