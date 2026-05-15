package traceql

import (
	"fmt"
	"strconv"
	"strings"
)

// pipeRate represents `rate()` — count of matching spans per second per time bucket.
type pipeRate struct{}

func (pr *pipeRate) String() string {
	return "rate()"
}

func parsePipeRate(lex *lexer) (pipe, error) {
	if !lex.isKeyword("rate") {
		return nil, fmt.Errorf("expecting 'rate'; got %q", lex.token)
	}
	lex.nextToken()
	if !lex.isKeyword("(") {
		return nil, fmt.Errorf("expecting '('; got %q", lex.token)
	}
	lex.nextToken()
	if !lex.isKeyword(")") {
		return nil, fmt.Errorf("expecting ')' at the end of rate pipe; got %q", lex.token)
	}
	lex.nextToken()

	pa, err := parsePipeAggregator(lex)
	if err != nil {
		return nil, err
	}
	if pa != nil {
		pa.aggregator = &pipeRate{}
		return pa, nil
	}
	return &pipeRate{}, nil
}

// pipeOverTime represents `<func>_over_time(<field>)` metrics functions.
//
// Supported functions: count_over_time(), min_over_time(field), max_over_time(field),
// avg_over_time(field), sum_over_time(field), histogram_over_time(field).
type pipeOverTime struct {
	funcName  string
	fieldName string // empty for count_over_time
}

func (pot *pipeOverTime) String() string {
	if pot.fieldName == "" {
		return pot.funcName + "()"
	}
	return pot.funcName + "(" + quoteFieldFilterIfNeeded(pot.fieldName) + ")"
}

func parsePipeOverTime(lex *lexer) (pipe, error) {
	if !lex.isKeyword("count_over_time", "min_over_time", "max_over_time", "avg_over_time", "sum_over_time", "histogram_over_time") {
		return nil, fmt.Errorf("expecting '*_over_time'; got %q", lex.token)
	}
	funcName := lex.token

	lex.nextToken()
	if !lex.isKeyword("(") {
		return nil, fmt.Errorf("expecting '('; got %q", lex.token)
	}
	lex.nextToken()

	var fieldName string
	if !lex.isKeyword(")") {
		// Parse the field argument.
		f, err := lex.nextCompoundToken()
		if err != nil {
			return nil, err
		}
		fieldName = f
	}

	// Validate: count_over_time takes no field, others require one.
	if funcName == "count_over_time" && fieldName != "" {
		return nil, fmt.Errorf("count_over_time() does not accept a field argument")
	}
	if funcName != "count_over_time" && fieldName == "" {
		return nil, fmt.Errorf("%s() requires a field argument", funcName)
	}

	if !lex.isKeyword(")") {
		return nil, fmt.Errorf("expecting ')' at the end of %s pipe; got %q", funcName, lex.token)
	}
	lex.nextToken()

	pot := &pipeOverTime{
		funcName:  funcName,
		fieldName: fieldName,
	}

	pa, err := parsePipeAggregator(lex)
	if err != nil {
		return nil, err
	}
	if pa != nil {
		pa.aggregator = pot
		return pa, nil
	}
	return pot, nil
}

// pipeQuantileOverTime represents `quantile_over_time(field, q1, q2, ...)`.
type pipeQuantileOverTime struct {
	fieldName string
	quantiles []string // kept as strings to preserve original precision
}

func (pq *pipeQuantileOverTime) String() string {
	s := "quantile_over_time(" + quoteFieldFilterIfNeeded(pq.fieldName)
	for _, q := range pq.quantiles {
		s += ", " + q
	}
	return s + ")"
}

func parsePipeQuantileOverTime(lex *lexer) (pipe, error) {
	if !lex.isKeyword("quantile_over_time") {
		return nil, fmt.Errorf("expecting 'quantile_over_time'; got %q", lex.token)
	}
	lex.nextToken()
	if !lex.isKeyword("(") {
		return nil, fmt.Errorf("expecting '('; got %q", lex.token)
	}
	lex.nextToken()

	// Parse field name.
	fieldName, err := lex.nextCompoundToken()
	if err != nil {
		return nil, err
	}
	if fieldName == "" {
		return nil, fmt.Errorf("quantile_over_time() requires a field argument")
	}

	if !lex.isKeyword(",") {
		return nil, fmt.Errorf("expecting ',' after field in quantile_over_time; got %q", lex.token)
	}

	// Parse one or more quantile values: q1[, q2, ...]
	var quantiles []string
	for {
		if !lex.isKeyword(",") {
			return nil, fmt.Errorf("expecting ',' before quantile in quantile_over_time; got %q", lex.token)
		}
		lex.nextToken()
		quantile, err := lex.nextCompoundToken()
		if err != nil {
			return nil, err
		}
		if quantile == "" {
			return nil, fmt.Errorf("quantile_over_time() requires at least one quantile value")
		}
		quantiles = append(quantiles, quantile)
		if !lex.isKeyword(",") {
			break
		}
	}

	if !lex.isKeyword(")") {
		return nil, fmt.Errorf("expecting ')' at the end of quantile_over_time; got %q", lex.token)
	}
	lex.nextToken()

	pq := &pipeQuantileOverTime{
		fieldName: fieldName,
		quantiles: quantiles,
	}

	pa, err := parsePipeAggregator(lex)
	if err != nil {
		return nil, err
	}
	if pa != nil {
		pa.aggregator = pq
		return pa, nil
	}
	return pq, nil
}

// pipeCompare represents `compare({selectionFilter}, topN, startNs, endNs)`.
// It auto-discovers attributes and computes per-attribute-value counts for baseline vs selection.
type pipeCompare struct {
	selectionFilter filter // the inner {filter} for the selection subset
	topN            int    // max values per attribute; 0 means default (10)
	startNs         int64  // selection window start in nanoseconds; 0 = use query range
	endNs           int64  // selection window end in nanoseconds; 0 = use query range
}

func (pc *pipeCompare) String() string {
	return "compare()"
}

// SelectionFilterString returns the LogsQL representation of the inner selection filter.
func (pc *pipeCompare) SelectionFilterString() string {
	if pc.selectionFilter == nil {
		return ""
	}
	return pc.selectionFilter.String()
}

func parsePipeCompare(lex *lexer) (pipe, error) {
	if !lex.isKeyword("compare") {
		return nil, fmt.Errorf("expecting 'compare'; got %q", lex.token)
	}
	lex.nextToken()
	if !lex.isKeyword("(") {
		return nil, fmt.Errorf("expecting '('; got %q", lex.token)
	}
	lex.nextToken()

	// Parse the inner selection filter — capture raw tokens until we hit "," or ")" at depth 0.
	var filterTokens []string
	depth := 0
	for {
		if lex.isKeyword("") {
			return nil, fmt.Errorf("unexpected end of query inside compare()")
		}
		if lex.isKeyword(",") && depth == 0 {
			break
		}
		if lex.isKeyword(")") && depth == 0 {
			break
		}
		if lex.isKeyword("{") {
			depth++
		} else if lex.isKeyword("}") {
			depth--
		}
		filterTokens = append(filterTokens, lex.rawToken)
		lex.nextToken()
	}

	// Parse the captured filter text.
	filterText := strings.Join(filterTokens, " ")
	var f filter
	if filterText != "" {
		parsedQ, parseErr := ParseQuery(filterText)
		if parseErr == nil {
			f = parsedQ.f
		}
	}

	// Parse optional comma-separated arguments: topN, startNs, endNs.
	var args []string
	for lex.isKeyword(",") {
		lex.nextToken()
		arg, _ := lex.nextCompoundToken()
		args = append(args, arg)
	}

	pc := &pipeCompare{selectionFilter: f}

	if len(args) >= 1 {
		if n, err := strconv.Atoi(args[0]); err == nil {
			pc.topN = n
		}
	}
	if len(args) >= 2 {
		if ns, err := strconv.ParseInt(args[1], 10, 64); err == nil {
			pc.startNs = ns
		}
	}
	if len(args) >= 3 {
		if ns, err := strconv.ParseInt(args[2], 10, 64); err == nil {
			pc.endNs = ns
		}
	}

	if !lex.isKeyword(")") {
		return nil, fmt.Errorf("expecting ')' at the end of compare; got %q", lex.token)
	}
	lex.nextToken()

	return pc, nil
}

// pipeWith represents `with(key=value, ...)` — query hints (sampling, exemplars).
// These are silently consumed and ignored by VictoriaTraces.
type pipeWith struct{}

func (pw *pipeWith) String() string {
	return "with()"
}

func parsePipeWith(lex *lexer) (pipe, error) {
	if !lex.isKeyword("with") {
		return nil, fmt.Errorf("expecting 'with'; got %q", lex.token)
	}
	lex.nextToken()
	if !lex.isKeyword("(") {
		return nil, fmt.Errorf("expecting '('; got %q", lex.token)
	}
	lex.nextToken()

	// Consume everything inside with(...) — key=value pairs separated by commas.
	depth := 1
	for depth > 0 {
		if lex.isKeyword("") {
			return nil, fmt.Errorf("unexpected end of query inside with()")
		}
		if lex.isKeyword("(") {
			depth++
		} else if lex.isKeyword(")") {
			depth--
		}
		if depth > 0 {
			lex.nextToken()
		}
	}
	lex.nextToken() // consume the final ")"

	return &pipeWith{}, nil
}
