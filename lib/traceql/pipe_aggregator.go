package traceql

import (
	"fmt"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"math"
	"strconv"
	"strings"
)

type pipeCount struct{}

func (pc *pipeCount) String() string {
	return "count()"
}

func parsePipeCount(lex *lexer) (pipe, error) {
	if !lex.isKeyword("count") {
		return nil, fmt.Errorf("expecting 'select'; got %q", lex.token)
	}
	lex.nextToken()
	if !lex.isKeyword("(") {
		return nil, fmt.Errorf("expecting '('; got %q", lex.token)
	}
	lex.nextToken()
	if !lex.isKeyword(")") {
		return nil, fmt.Errorf("expecting ')' at the end of count pipe; got %q", lex.token)
	}
	lex.nextToken()

	pa, err := parsePipeAggregator(lex)
	if err != nil {
		return nil, err
	}
	if pa != nil {
		pa.aggregator = &pipeCount{}
		return pa, nil
	}
	return &pipeCount{}, nil
}

type pipeAggregatorOnField struct {
	aggregatorName string
	fieldFilter    string
}

func (paof *pipeAggregatorOnField) String() string {
	if paof.fieldFilter == "" {
		logger.Panicf("BUG: pipeAggregatorOnField must contain exact a single field filter")
	}
	return paof.aggregatorName + "(" + quoteFieldFilterIfNeeded(paof.fieldFilter) + ")"
}

func parsePipeAggregatorOnField(lex *lexer) (pipe, error) {
	if !lex.isKeyword("sum", "avg", "min", "max") {
		return nil, fmt.Errorf("expecting 'sum', 'avg', 'min', 'max'; got %q", lex.token)
	}
	aggregatorName := lex.token

	lex.nextToken()
	if !lex.isKeyword("(") {
		return nil, fmt.Errorf("expecting '('; got %q", lex.token)
	}
	lex.nextToken()

	fieldFilter, err := lex.nextCompoundToken()
	if err != nil {
		return nil, err
	}
	pf := &pipeAggregatorOnField{
		aggregatorName: aggregatorName,
		fieldFilter:    fieldFilter,
	}

	if !lex.isKeyword(")") {
		return nil, fmt.Errorf("expecting ')' at the end of select pipe; got %q", lex.token)
	}
	lex.nextToken()

	pa, err := parsePipeAggregator(lex)
	if err != nil {
		return nil, err
	}
	if pa != nil {
		pa.aggregator = pf
		return pa, nil
	}
	return pf, nil
}

type pipeAggregator struct {
	// fieldFilters contains list of filters for fields to fetch
	aggregator pipe
	op         string
	value      float64
	stringRepr string
}

func (pa *pipeAggregator) String() string {
	if pa.aggregator == nil {
		logger.Panicf("BUG: pipeAggregator must contain an aggregator")
	}

	if pa.op == "" {
		return pa.aggregator.String()
	}
	return fmt.Sprintf("%s %s", pa.aggregator.String(), pa.stringRepr)
}

func parsePipeAggregator(lex *lexer) (*pipeAggregator, error) {
	if !lex.isKeyword(">", "<", "=", "!=", ">=", "<=") {
		// the pipe does not contain condition such as `count() > 20`. return without pipeAggregator.
		return nil, nil
	}
	op := lex.token
	lex.nextToken()

	fValue, fStr, err := parseNumber(lex)
	if err != nil {
		return nil, fmt.Errorf("cannot parse '%s %s': %w", op, fStr, err)
	}
	pa := &pipeAggregator{
		op:         op,
		value:      fValue,
		stringRepr: op + " " + fStr,
	}
	return pa, nil
}

func parseNumber(lex *lexer) (float64, string, error) {
	s, err := lex.nextCompoundToken()
	if err != nil {
		return 0, "", fmt.Errorf("cannot read number: %w", err)
	}

	f := parseMathNumber(s)
	if !math.IsNaN(f) || strings.EqualFold(s, "nan") {
		return f, s, nil
	}

	return 0, s, fmt.Errorf("cannot parse %q as float64", s)
}

func parseMathNumber(s string) float64 {
	f, ok := tryParseNumber(s)
	if ok {
		return f
	}
	nsecs, ok := TryParseTimestampRFC3339Nano(s)
	if ok {
		return float64(nsecs)
	}
	ipNum, ok := tryParseIPv4(s)
	if ok {
		return float64(ipNum)
	}
	return nan
}

func tryParseNumber(s string) (float64, bool) {
	if len(s) == 0 {
		return 0, false
	}
	f, ok := tryParseFloat64(s)
	if ok {
		return f, true
	}
	nsecs, ok := tryParseDuration(s)
	if ok {
		return float64(nsecs), true
	}
	bytes, ok := tryParseBytes(s)
	if ok {
		return float64(bytes), true
	}
	if isLikelyNumber(s) {
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return f, true
		}
		n, err := strconv.ParseInt(s, 0, 64)
		if err == nil {
			return float64(n), true
		}
	}
	return 0, false
}

func isLikelyNumber(s string) bool {
	if !isNumberPrefix(s) {
		return false
	}
	if strings.Count(s, ".") > 1 {
		// This is likely IP address
		return false
	}
	if strings.IndexByte(s, ':') >= 0 || strings.Count(s, "-") > 2 {
		// This is likely a timestamp
		return false
	}
	return true
}

func isNumberPrefix(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] == '-' || s[0] == '+' {
		s = s[1:]
		if len(s) == 0 {
			return false
		}
	}
	if len(s) >= 3 && strings.EqualFold(s, "inf") {
		return true
	}
	if s[0] == '.' {
		s = s[1:]
		if len(s) == 0 {
			return false
		}
	}
	return s[0] >= '0' && s[0] <= '9'
}

var nan = math.NaN()
