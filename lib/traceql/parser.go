package traceql

import (
	"fmt"
	"github.com/VictoriaMetrics/VictoriaLogs/lib/prefixfilter"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type Query struct {
	f filter

	pipes []pipe

	// timestamp is the timestamp context used for parsing the query.
	timestamp int64
}

// String returns string representation for q.
func (q *Query) String() string {
	s := q.f.String()

	// merge trace duration filter if any
	traceDurationFilters := q.f.GetTraceDurationFilters()
	if len(traceDurationFilters) > 0 {
		tFilterStr := " "
		for _, tf := range traceDurationFilters {
			v := tf.value
			if duration, ok := tryParseDuration(v); ok {
				v = strconv.FormatInt(duration, 10)
			}
			tFilterStr += fmt.Sprintf("AND duration :%s %s ", tf.op, v)
		}
		s += fmt.Sprintf(` | join by (trace_id) ({trace_id_idx_stream!=""} %s | fields trace_id_idx  | rename trace_id_idx as trace_id) inner`, tFilterStr)
	}

	for _, p := range q.pipes {
		s += " | " + p.String()
	}

	return s
}

// HasPipe indicates whether this query contains only filter(s), or contains filter(s) along with pipe(s).
func (q *Query) HasPipe() bool {
	return len(q.pipes) > 0
}

// ParseQuery parses s.
func ParseQuery(s string) (*Query, error) {
	timestamp := time.Now().UnixNano()
	return ParseQueryAtTimestamp(s, timestamp)
}

// ParseQueryAtTimestamp parses s in the context of the given timestamp.
//
// E.g. _time:duration filters are adjusted according to the provided timestamp as _time:[timestamp-duration, duration].
func ParseQueryAtTimestamp(s string, timestamp int64) (*Query, error) {
	lex := newLexer(s, timestamp)

	q, err := parseQuery(lex)
	if err != nil {
		return nil, err
	}
	if !lex.isEnd() {
		return nil, fmt.Errorf("unexpected unparsed tail after [%s]; context: [%s]; tail: [%s]", q, lex.context(), lex.rawToken+lex.s)
	}
	return q, nil
}

func parseQuery(lex *lexer) (*Query, error) {
	var q Query

	f, err := parseFilter(lex, true)
	if err != nil {
		return nil, fmt.Errorf("%w; context: [%s]", err, lex.context())
	}

	q.f = f
	q.timestamp = lex.currentTimestamp

	if lex.isKeyword("|") {
		lex.nextToken()
		pipes, err := parsePipes(lex)
		if err != nil {
			return nil, fmt.Errorf("%w; context: [%s]", err, lex.context())
		}
		q.pipes = pipes
	}

	return &q, nil
}

func parseFilter(lex *lexer, allowPipeKeywords bool) (filter, error) {
	if lex.isKeyword("|", ")", "}", "") {
		return nil, fmt.Errorf("missing query")
	}

	if !allowPipeKeywords {
		// Verify the first token in the filter doesn't match pipe names.
		firstToken := strings.ToLower(lex.rawToken)
		if firstToken == "by" || isPipeName(firstToken) {
			return nil, fmt.Errorf("query filter cannot start with pipe keyword %q; see https://docs.victoriametrics.com/victorialogs/logsql/#query-syntax; "+
				"please put the first word of the filter into quotes", firstToken)
		}
	}

	fo, err := parseFilterOr(lex, "")
	if err != nil {
		return nil, err
	}
	return fo, nil
}

func parseFilterOr(lex *lexer, fieldName string) (filter, error) {
	var filters []filter
	for {
		f, err := parseFilterAnd(lex, fieldName)
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)
		switch {
		case lex.isKeyword("|", ")", "}", ""):
			if len(filters) == 1 {
				return filters[0], nil
			}
			fo := &filterOr{
				filters: filters,
			}
			return fo, nil
		case lex.isKeyword("or", "||"):
			lex.nextToken()
		}
	}
}

func parseFilterAnd(lex *lexer, fieldName string) (filter, error) {
	var filters []filter
	for {
		f, err := parseFilterGeneric(lex, fieldName)
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)
		switch {
		case lex.isKeyword("", "}", "||", ")", "|"):
			if len(filters) == 1 {
				return filters[0], nil
			}
			fa := &filterAnd{
				filters: filters,
			}
			return fa, nil
		case lex.isKeyword("and", "&&"):
			lex.nextToken()
			//case lex.isKeyword("or", "||"):
			//	parseFilterOr(lex, fieldName)
			//	lex.nextToken()
		}
	}
}

func parsePipes(lex *lexer) ([]pipe, error) {
	var pipes []pipe
	for {
		p, err := parsePipe(lex)
		if err != nil {
			return nil, err
		}
		pipes = append(pipes, p)

		switch {
		case lex.isKeyword("|"):
			lex.nextToken()
		case lex.isKeyword(")", ""):
			return pipes, nil
		default:
			return nil, fmt.Errorf("unexpected token after [%s]: %q; expecting '|' or ')'", pipes[len(pipes)-1], lex.token)
		}
	}
}

func parseFilterGeneric(lex *lexer, fieldName string) (filter, error) {
	// Verify the previous adjacent token
	if lex.isKeyword("{", "(") {
		if err := lex.checkPrevAdjacentToken("{", "(", "||", "&&"); err != nil {
			return nil, err
		}
	}
	//else {
	//	if err := lex.checkPrevAdjacentToken("|", "(", "{", "!", "-"); err != nil {
	//		return nil, err
	//	}
	//}

	// Detect the filter.
	switch {
	case lex.isKeyword("{"):
		return parseFilterCurlyBrackets(lex, fieldName)
	case lex.isKeyword("("):
		return parseFilterParens(lex, fieldName)
	case lex.isKeyword(">", "<", ">=", "<=", "=", "=~", "!=", "!~"):
		return parseFilterCommon(lex, fieldName)
	case lex.isKeyword("&&"):
		return parseFilterAnd(lex, fieldName)
	case lex.isKeyword("||"):
		return parseFilterOr(lex, fieldName)
	case lex.isKeyword("not", "!", "-"):
		return nil, nil
	case lex.isKeyword("true", "false"):
		return parseFilterTrue(lex, fieldName)
	default:
		return parseFilterPhrase(lex, fieldName)
	}
}

func parseFilterTrue(lex *lexer, fieldName string) (filter, error) {
	if lex.isKeyword("false") {
		return nil, fmt.Errorf("keyword false is not supported")
	}

	if fieldName != "" {
		return nil, fmt.Errorf("unexpected field name for bool filter: %q", fieldName)
	}

	lex.nextToken()
	if lex.isKeyword(">", "<", ">=", "<=", "=", "=~", "!=", "!~") {
		return nil, fmt.Errorf("unexpected token after keyword true: %q", lex.token)
	}

	return &filterNoop{}, nil
}

func parseFilterParens(lex *lexer, fieldName string) (filter, error) {
	lex.nextToken()

	if lex.isKeyword("") {
		// nothing here, ()
		lex.nextToken()
		return &filterOr{
			filters: []filter{},
		}, nil
	}

	f, err := parseFilterOr(lex, fieldName)
	if err != nil {
		return nil, err
	}

	if !lex.isKeyword(")") {
		return nil, fmt.Errorf("missing ')'; got %q", lex.token)
	}
	lex.nextToken()

	return f, nil
}

func parseFilterCurlyBrackets(lex *lexer, fieldName string) (filter, error) {
	lex.nextToken()

	if lex.isKeyword("}") {
		// nothing here, {}
		lex.nextToken()
		return &filterTrace{
			andFilter: &filterAnd{},
		}, nil
	}

	f, err := parseFilterOr(lex, fieldName)
	if err != nil {
		return nil, err
	}

	if !lex.isKeyword("}") {
		return nil, fmt.Errorf("missing '}'; got %q", lex.token)
	}
	lex.nextToken()

	ft := &filterTrace{
		andFilter: f,
	}
	return ft, nil
}

func parseFilterPhrase(lex *lexer, fieldName string) (filter, error) {
	phrase, err := lex.nextCompoundToken()
	if err != nil {
		return nil, err
	}

	if fieldName == "" {
		return parseFilterGeneric(lex, phrase)
	}

	f := &filterPhrase{
		fieldName: fieldName,
		phrase:    phrase,
	}
	return f, nil

	//// The phrase is either a search phrase or a search prefix.
	//if !lex.isSkippedSpace && lex.isKeyword("*") {
	//	// The phrase is a search prefix in the form `foo*`.
	//	lex.nextToken()
	//	f := &filterPrefix{
	//		fieldName: getCanonicalColumnName(fieldName),
	//		prefix:    phrase,
	//	}
	//	return f, nil
	//}
	//
}

func parseFilterCommon(lex *lexer, fieldName string) (filter, error) {
	op := lex.token
	lex.nextToken()

	phrase, err := lex.nextCompoundToken()
	if err != nil {
		return nil, fmt.Errorf("cannot parse token after %q: %w", op, err)
	}

	f := &filterCommon{
		fieldName: fieldName,
		op:        op,
		value:     phrase,
	}
	return f, nil
}

type lexer struct {
	// s contains unparsed tail of sOrig
	s string

	// sOrig contains the original string
	sOrig string

	// token contains the current token
	//
	// an empty token means the end of s
	token string

	// rawToken contains raw token before unquoting
	rawToken string

	// prevRawToken contains the previously parsed token before unquoting
	prevRawToken string

	// isSkippedSpace is set to true if there was a whitespace before the token in s
	isSkippedSpace bool

	// currentTimestamp is the current timestamp in nanoseconds.
	//
	// It is used for proper initializing of _time filters with relative time ranges.
	currentTimestamp int64
}

// newLexer returns new lexer for the given s at the given timestamp.
//
// The timestamp is used for properly parsing relative timestamps such as _time:1d.
//
// The lex.token points to the first token in s.
func newLexer(s string, timestamp int64) *lexer {
	lex := &lexer{
		s:                s,
		sOrig:            s,
		currentTimestamp: timestamp,
	}
	lex.nextToken()
	return lex
}

// nextToken updates lex.token to the next token.
func (lex *lexer) nextToken() {
	s := lex.s
	lex.prevRawToken = lex.rawToken
	lex.token = ""
	lex.rawToken = ""
	lex.isSkippedSpace = false

	if len(s) == 0 {
		return
	}

again:
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		lex.nextCharToken(s, size)
		return
	}

	// Skip whitespace
	for unicode.IsSpace(r) {
		lex.isSkippedSpace = true
		s = s[size:]
		r, size = utf8.DecodeRuneInString(s)
	}

	if r == '#' {
		// skip comment till \n
		n := strings.IndexByte(s, '\n')
		if n < 0 {
			s = ""
		} else {
			s = s[n+1:]
		}
		goto again
	}

	// Try decoding simple token
	tokenLen := 0
	for isTokenRune(r) {
		tokenLen += size
		r, size = utf8.DecodeRuneInString(s[tokenLen:])
	}
	if tokenLen > 0 {
		lex.nextCharToken(s, tokenLen)
		return
	}

	switch r {
	case '"', '`':
		prefix, err := strconv.QuotedPrefix(s)
		if err != nil {
			lex.nextCharToken(s, 1)
			return
		}
		token, err := strconv.Unquote(prefix)
		if err != nil {
			lex.nextCharToken(s, 1)
			return
		}
		lex.token = token
		lex.rawToken = prefix
		lex.s = s[len(prefix):]
		return
	case '\'':
		var b []byte
		for !strings.HasPrefix(s[size:], "'") {
			ch, _, newTail, err := strconv.UnquoteChar(s[size:], '\'')
			if err != nil {
				lex.nextCharToken(s, 1)
				return
			}
			b = utf8.AppendRune(b, ch)
			size = len(s) - len(newTail)
		}
		size++
		lex.token = string(b)
		lex.rawToken = string(s[:size])
		lex.s = s[size:]
		return
	case '=':
		if strings.HasPrefix(s[size:], "~") {
			lex.nextCharToken(s, 2)
			return
		}
		lex.nextCharToken(s, 1)
		return
	case '!':
		if strings.HasPrefix(s[size:], ">>") || strings.HasPrefix(s[size:], "<<") {
			lex.nextCharToken(s, 3)
			return
		} else if strings.HasPrefix(s[size:], ">") || strings.HasPrefix(s[size:], "<") || strings.HasPrefix(s[size:], "~") || strings.HasPrefix(s[size:], "=") {
			lex.nextCharToken(s, 2)
			return
		}
		lex.nextCharToken(s, 1)
		return
	case '&':
		if strings.HasPrefix(s[size:], ">>") || strings.HasPrefix(s[size:], "<<") {
			lex.nextCharToken(s, 3)
			return
		} else if strings.HasPrefix(s[size:], "&") || strings.HasPrefix(s[size:], ">") || strings.HasPrefix(s[size:], "<") || strings.HasPrefix(s[size:], "~") {
			lex.nextCharToken(s, 2)
			return
		}
		lex.nextCharToken(s, 1)
		return
	case '|':
		if strings.HasPrefix(s[size:], "|") {
			lex.nextCharToken(s, 2)
			return
		}
		lex.nextCharToken(s, 1)
		return
	case '>', '<':
		if strings.HasPrefix(s[size:], "=") {
			lex.nextCharToken(s, 2)
			return
		}
		lex.nextCharToken(s, 1)
		return
	default:
		lex.nextCharToken(s, size)
		return
	}
}

type lexerState struct {
	lex lexer
}

func (lex *lexer) copyFrom(src *lexer) {
	*lex = *src
}

func (lex *lexer) backupState() *lexerState {
	var ls lexerState
	ls.lex.copyFrom(lex)
	return &ls
}

func (lex *lexer) restoreState(ls *lexerState) {
	lex.copyFrom(&ls.lex)
}

func (lex *lexer) isEnd() bool {
	return len(lex.s) == 0 && len(lex.token) == 0 && len(lex.rawToken) == 0
}

func (lex *lexer) isQuotedToken() bool {
	return lex.token != lex.rawToken
}

func (lex *lexer) nextCompoundToken() (string, error) {
	return lex.nextCompoundTokenExt(nil)
}

func (lex *lexer) nextCompoundMathToken() (string, error) {
	return lex.nextCompoundTokenExt(mathStopCompoundTokens)
}

func (lex *lexer) nextCompoundTokenExt(stopTokens []string) (string, error) {
	if lex.isQuotedToken() {
		// Quoted tokens cannot be a part of compound token, so return them as is.
		s := lex.token
		lex.nextToken()
		return s, nil
	}

	if !lex.isSkippedSpace && lex.isKeywordAny(deniedFirstCompoundTokens) && isWord(lex.prevRawToken) {
		return "", fmt.Errorf("missing whitespace between %q and %q", lex.prevRawToken, lex.token)
	}

	if !lex.isAllowedCompoundToken(stopTokens) {
		return "", fmt.Errorf("compound token cannot start with %q; put it into quotes if needed", lex.token)
	}

	s := lex.token
	lex.nextToken()

	for !lex.isSkippedSpace && lex.isAllowedCompoundToken(stopTokens) {
		s += lex.rawToken
		lex.nextToken()
	}

	if slices.Contains(glueCompoundTokens, s) {
		// Disallow a single-char compound token with glue chars, since this is error-prone.
		// See https://github.com/VictoriaMetrics/VictoriaLogs/issues/590
		return "", fmt.Errorf("compound token cannot be equal to %q; put it into quotes if needed", s)
	}

	return s, nil
}

func (lex *lexer) isAllowedCompoundToken(stopTokens []string) bool {
	if lex.isQuotedToken() {
		// Quoted token cannot be a part of compound token
		return false
	}

	if len(lex.token) == 0 {
		// Missing token (EOF).
		return false
	}

	// Stop tokens are disallowed in the compound token.
	if lex.isKeywordAny(stopTokens) {
		return false
	}

	// Glue tokens are allowed to be a part of compound token.
	if lex.isKeywordAny(glueCompoundTokens) {
		return true
	}

	// Regular word token is allowed to be a part of compound token.
	return isWord(lex.token)
}

func isWord(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isTokenRune(r) {
			return false
		}
	}
	return true
}

// deniedFirstCompoundTokens contains disallowed starting tokens for compound tokens without the whitespace in front of these tokens.
var deniedFirstCompoundTokens = []string{
	"/",
	".",
	"$",
}

// glueCompoundTokens contains tokens allowed inside unquoted compound tokens.
var glueCompoundTokens = []string{
	"+", // Seen in time formats: 2025-07-20T10:20:30+03:00
	"-", // Seen in hostnames: foo-bar-baz
	"/", // Seen in paths: foo/bar/baz
	":", // Seen in tcp addresses: foo:1235
	".", // Seen in hostnames: foobar.com
	"$", // Seen in PHP-like vars: $foo
}

// mathStopCompoundTokens contains tokens from the glueCompoundTokens, which are disallowed in math compound tokens.
var mathStopCompoundTokens = []string{
	"+",
	"-",
	"/",
}

func (lex *lexer) isPrevRawToken(tokens []string) bool {
	prevTokenLower := strings.ToLower(lex.prevRawToken)
	for _, token := range tokens {
		if token == prevTokenLower {
			return true
		}
	}
	return false
}

func (lex *lexer) checkPrevAdjacentToken(tokens ...string) error {
	if lex.prevRawToken == "" {
		return nil
	}

	if !lex.isPrevRawToken(tokens) {
		return fmt.Errorf("missing whitespace or ':' between %q and %q; probably, the whole string must be put into quotes", lex.prevRawToken, lex.token)
	}

	return nil
}

func (lex *lexer) isKeyword(keywords ...string) bool {
	return lex.isKeywordAny(keywords)
}

func (lex *lexer) isKeywordAny(keywords []string) bool {
	if lex.isQuotedToken() {
		return false
	}
	tokenLower := strings.ToLower(lex.token)
	for _, kw := range keywords {
		if kw == tokenLower {
			return true
		}
	}
	return false
}

func (lex *lexer) context() string {
	tail := lex.sOrig
	tail = tail[:len(tail)-len(lex.s)]
	if len(tail) > 50 {
		tail = tail[len(tail)-50:]
	}
	return tail
}

func (lex *lexer) nextCharToken(s string, size int) {
	lex.token = s[:size]
	lex.rawToken = lex.token
	lex.s = s[size:]
}

func quoteTokenIfNeeded(s string) string {
	if !needQuoteToken(s) {
		return s
	}
	return strconv.Quote(s)
}

func needQuoteToken(s string) bool {
	sLower := strings.ToLower(s)
	if _, ok := reservedKeywords[sLower]; ok {
		return true
	}
	if isPipeName(sLower) {
		return true
	}
	for _, r := range s {
		if !isTokenRune(r) && r != '.' {
			return true
		}
	}
	return false
}

func quoteFieldFilterIfNeeded(s string) string {
	if !prefixfilter.IsWildcardFilter(s) {
		return quoteTokenIfNeeded(s)
	}

	wildcard := s[:len(s)-1]
	if wildcard == "" || !needQuoteToken(wildcard) {
		return s
	}
	return strconv.Quote(s)
}

var reservedKeywords = func() map[string]struct{} {
	kws := []string{
		// An empty keyword means end of parsed string
		"",

		// boolean operator tokens for 'foo and bar or baz not xxx'
		"and",
		"or",
		"not",
		"!", // synonym for "not"

		// parens for '(foo or bar) and baz'
		"(",
		")",

		// stream filter tokens for '_stream:{foo=~"bar", baz="a"}'
		"{",
		"}",
		"=",
		"!=",
		"=~",
		"!~",
		",",

		// delimiter between query parts:
		// 'foo and bar | extract "<*> foo <time>" | filter x:y | ...'
		"|",

		// delimiter between field name and query in filter: 'foo:bar'
		":",

		// prefix search: 'foo*'
		"*",

		// keywords for _time filter: '_time:(now-1h, now]'
		"[",
		"]",
		"now",
		"offset",
		"-",

		// functions
		"contains_all",
		"contains_any",
		"contains_common_case",
		"eq_field",
		"equals_common_case",
		"exact",
		"i",
		"in",
		"ipv4_range",
		"le_field",
		"len_range",
		"lt_field",
		"pattern_match",
		"pattern_match_full",
		"pattern_match_prefix",
		"pattern_match_suffix",
		"range",
		"re",
		"seq",
		"string_range",
		"value_type",

		// queryOptions start with this keyword
		"options",

		// 'if' is used in conditional pipes such as `format if (...) ...`
		"if",

		// 'by' is used in various pipes such as `stats by (...) ...`
		"by",

		// 'as' is used in various pipes such as `format ... as ...`
		"as",
	}
	m := make(map[string]struct{}, len(kws))
	for _, kw := range kws {
		m[kw] = struct{}{}
	}
	return m
}()

func subNoOverflowInt64(a, b int64) int64 {
	if a == math.MinInt64 || a == math.MaxInt64 {
		// Assume that a is either +Inf or -Inf.
		// Subtracting any number from Inf must result in Inf.
		return a
	}
	if b >= 0 {
		if a < math.MinInt64+b {
			return math.MinInt64
		}
		return a - b
	}
	if a > math.MaxInt64+b {
		return math.MaxInt64
	}
	return a - b
}
