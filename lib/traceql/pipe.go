package traceql

import (
	"fmt"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"strings"
	"sync"
)

type pipe interface {
	// String returns string representation of the pipe.
	String() string

	//// visitSubqueries must call visitFunc for all the subqueries, which exist at the pipe (recursively).
	//visitSubqueries(visitFunc func(q *Query))
}

func parsePipe(lex *lexer) (pipe, error) {
	pps := getPipeParsers()
	for pipeName, parseFunc := range pps {
		if !lex.isKeyword(pipeName) {
			continue
		}
		p, err := parseFunc(lex)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %q pipe: %w", pipeName, err)
		}
		return p, nil
	}

	return nil, fmt.Errorf("unexpected pipe %q", lex.token)
}

var pipeParsers map[string]pipeParseFunc
var pipeParsersOnce sync.Once

type pipeParseFunc func(lex *lexer) (pipe, error)

func getPipeParsers() map[string]pipeParseFunc {
	pipeParsersOnce.Do(initPipeParsers)
	return pipeParsers
}

func initPipeParsers() {
	pipeParsers = map[string]pipeParseFunc{
		// Aggregators
		"count": parsePipeCount,
		"avg":   parsePipeAggregatorOnField,
		"max":   parsePipeAggregatorOnField,
		"min":   parsePipeAggregatorOnField,
		"sum":   parsePipeAggregatorOnField,

		// Selection
		"select": parsePipeSelect,

		// Grouping
		"by": parsePipeBy,
	}
}

func isPipeName(s string) bool {
	pps := getPipeParsers()
	sLower := strings.ToLower(s)
	return pps[sLower] != nil
}

func mustParsePipes(s string, timestamp int64) []pipe {
	lex := newLexer(s, timestamp)
	pipes, err := parsePipes(lex)
	if err != nil {
		logger.Panicf("BUG: cannot parse [%s]: %s", s, err)
	}
	if !lex.isEnd() {
		logger.Panicf("BUG: unexpected tail left after parsing [%s]: %s", s, lex.context())
	}
	return pipes
}

func mustParsePipe(s string, timestamp int64) pipe {
	lex := newLexer(s, timestamp)
	p, err := parsePipe(lex)
	if err != nil {
		logger.Panicf("BUG: cannot parse [%s]: %s", s, err)
	}
	if !lex.isEnd() {
		logger.Panicf("BUG: unexpected tail left after parsing [%s]: %s", s, lex.context())
	}
	return p
}
