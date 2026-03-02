package traceql

import (
	"fmt"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"strings"
)

type pipeSelect struct {
	// fieldFilters contains list of filters for fields to fetch
	fieldFilters []string
}

func (pf *pipeSelect) String() string {
	if len(pf.fieldFilters) == 0 {
		logger.Panicf("BUG: pipeSelect must contain at least a single field filter")
	}
	return "select(" + fieldNamesString(pf.fieldFilters) + ")"
}

func parsePipeSelect(lex *lexer) (pipe, error) {
	if !lex.isKeyword("select") {
		return nil, fmt.Errorf("expecting 'select'; got %q", lex.token)
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
	pf := &pipeSelect{
		fieldFilters: fieldFilters,
	}
	if !lex.isKeyword(")") {
		return nil, fmt.Errorf("expecting ')' at the end of select pipe; got %q", lex.token)
	}
	lex.nextToken()
	return pf, nil
}

func parseCommaSeparatedFields(lex *lexer) ([]string, error) {
	var fields []string
	for {
		field, err := parseFieldFilter(lex)
		if err != nil {
			return nil, fmt.Errorf("cannot parse field name: %w", err)
		}
		fields = append(fields, field)
		if !lex.isKeyword(",") {
			return fields, nil
		}
		lex.nextToken()
	}
}

func fieldNamesString(fields []string) string {
	a := make([]string, len(fields))
	for i, f := range fields {
		a[i] = quoteFieldFilterIfNeeded(f)
	}
	return strings.Join(a, ", ")
}

func parseFieldFilter(lex *lexer) (string, error) {
	if lex.isKeyword("*") {
		lex.nextToken()
		return "*", nil
	}

	fieldName, err := lex.nextCompoundToken()
	if err != nil {
		return "", err
	}
	if !lex.isSkippedSpace && lex.isKeyword("*") {
		lex.nextToken()
		fieldName += "*"
	}

	return fieldName, nil
}
