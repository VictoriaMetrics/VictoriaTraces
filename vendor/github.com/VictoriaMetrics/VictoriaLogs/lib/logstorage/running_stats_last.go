package logstorage

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/prefixfilter"
)

type runningStatsLast struct {
	fieldName string
	offset    int
}

func (sl *runningStatsLast) String() string {
	s := "last(" + quoteTokenIfNeeded(sl.fieldName)
	if sl.offset > 0 {
		s += fmt.Sprintf(", %d", sl.offset)
	}
	s += ")"
	return s
}

func (sl *runningStatsLast) updateNeededFields(pf *prefixfilter.Filter) {
	pf.AddAllowFilter(sl.fieldName)
}

func (sl *runningStatsLast) newRunningStatsProcessor() runningStatsProcessor {
	return &runningStatsLastProcessor{
		sl: sl,
	}
}

type runningStatsLastProcessor struct {
	sl     *runningStatsLast
	values []string
}

func (slp *runningStatsLastProcessor) updateRunningStats(_ runningStatsFunc, row []Field) {
	sl := slp.sl

	value := ""
	for i := range row {
		f := &row[i]
		if f.Name == sl.fieldName {
			value = strings.Clone(f.Value)
			break
		}
	}

	slp.values = append(slp.values, value)
	if len(slp.values) > sl.offset+1 {
		slp.values = slp.values[len(slp.values)-sl.offset-1:]
	}
}

func (slp *runningStatsLastProcessor) getRunningStats() string {
	if len(slp.values) <= slp.sl.offset {
		return ""
	}
	return slp.values[len(slp.values)-slp.sl.offset-1]
}

func parseRunningStatsLast(lex *lexer) (runningStatsFunc, error) {
	args, err := parseStatsFuncArgs(lex, "last")
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("missing the field name inside last()")
	}
	if len(args) > 2 {
		return nil, fmt.Errorf("too many args for the last() function; got %d; want 1 or 2 args; args: %q", len(args), args)
	}

	fieldName := args[0]

	offset := 0
	if len(args) == 2 {
		offsetStr := args[1]
		n, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, fmt.Errorf("cannot parse offset=%q at last(%q, %q): %w", offsetStr, fieldName, offsetStr, err)
		}
		if n < 0 {
			return nil, fmt.Errorf("offset=%d cannot be negative at last(%q, %q)", n, fieldName, offsetStr)
		}
		offset = n
	}

	sf := &runningStatsLast{
		fieldName: fieldName,
		offset:    offset,
	}
	return sf, nil
}
