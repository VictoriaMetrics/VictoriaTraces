package traceql

import (
	"fmt"
	"testing"
)

func TestParsePipeFieldsSuccess(t *testing.T) {
	f := func(pipeStr string) {
		t.Helper()
		expectParsePipesSuccess(t, pipeStr)
	}
	//
	//f(`select(*)`)
	//f(`select (f1, f2, f3)`)
	f(`select(f1,f2, f3) | by(f2,f3) | count() > 2m`)
	//f(`count() > 2m`)
}

func expectParsePipesSuccess(t *testing.T, pipeStr string) {
	t.Helper()

	lex := newLexer(pipeStr, 0)
	p, err := parsePipes(lex)
	if err != nil {
		t.Fatalf("cannot parse [%s]: %s", pipeStr, err)
	}
	if !lex.isEnd() {
		t.Fatalf("unexpected tail after parsing [%s]: [%s]", pipeStr, lex.s)
	}

	fmt.Println(p)
}
