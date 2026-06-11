package common

import "testing"

func TestParseExtraFilters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"json string", `{"foo":"bar"}`, `foo:=bar`},
		{"json array", `{"foo":["bar","baz"]}`, `foo:in(bar,baz)`},
		{"json mixed", `{"z":"=b ","c":["d","e,"],"a":[],"_msg":"x"}`, `z:="=b " c:in(d,"e,") =x`},
		{"logsql", `foo:(bar or baz) error _time:5m {"foo"=bar,baz="z"}`, `{foo="bar",baz="z"} (foo:bar or foo:baz) error _time:5m`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ParseExtraFilters(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got := f.String(); got != tt.want {
				t.Fatalf("got %q; want %q", got, tt.want)
			}
		})
	}

	for _, input := range []string{`{"foo"}`, `[1,2]`, `{"foo":[1]}`, `foo:(bar`, `foo | count()`} {
		if _, err := ParseExtraFilters(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestParseExtraStreamFilters(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{`{"foo":"bar"}`, `{foo="bar"}`},
		{`{"foo":["bar","baz"]}`, `{foo=~"bar|baz"}`},
		{`{"z":"b","c":["d","e|\""],"a":[],"_msg":"x"}`, `{z="b",c=~"d|e\\|\"",_msg="x"}`},
		{`foo:(bar or baz) error _time:5m {"foo"=bar,baz="z"}`, `{foo="bar",baz="z"} (foo:bar or foo:baz) error _time:5m`},
	}
	for _, tt := range tests {
		f, err := ParseExtraStreamFilters(tt.input)
		if err != nil {
			t.Fatal(err)
		}
		if got := f.String(); got != tt.want {
			t.Fatalf("got %q; want %q", got, tt.want)
		}
	}

	for _, input := range []string{`{"foo"}`, `[1,2]`, `{"foo":[1]}`, `foo:(bar`, `foo | count()`} {
		if _, err := ParseExtraStreamFilters(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}
