package traceql

import (
	"fmt"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
)

type pipeBy struct {
	// fieldFilters contains list of filters for fields to fetch
	fieldFilters []string
}

func (pf *pipeBy) String() string {
	if len(pf.fieldFilters) == 0 {
		logger.Panicf("BUG: pipeBy must contain at least a single field filter")
	}
	return "by(" + fieldNamesString(pf.fieldFilters) + ")"
}

func parsePipeBy(lex *lexer) (pipe, error) {
	if !lex.isKeyword("by") {
		return nil, fmt.Errorf("expecting 'by'; got %q", lex.token)
	}
	lex.nextToken()
	if !lex.isKeyword("(") {
		return nil, fmt.Errorf("expecting '('; got %q", lex.token)
	}
	lex.nextToken()

	fieldFilters, err := parseCommaSeparatedFields(lex)
	if err != nil {
		return nil, err
	}
	pf := &pipeBy{
		fieldFilters: fieldFilters,
	}
	if !lex.isKeyword(")") {
		return nil, fmt.Errorf("expecting ')' at the end of by pipe; got %q", lex.token)
	}
	lex.nextToken()
	return pf, nil
}
