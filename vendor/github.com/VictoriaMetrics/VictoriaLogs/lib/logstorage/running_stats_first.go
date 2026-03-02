package logstorage

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/prefixfilter"
)

type runningStatsFirst struct {
	fieldName string
	offset    int
}

func (sf *runningStatsFirst) String() string {
	s := "first(" + quoteTokenIfNeeded(sf.fieldName)
	if sf.offset > 0 {
		s += fmt.Sprintf(", %d", sf.offset)
	}
	s += ")"
	return s
}

func (sf *runningStatsFirst) updateNeededFields(pf *prefixfilter.Filter) {
	pf.AddAllowFilter(sf.fieldName)
}

func (sf *runningStatsFirst) newRunningStatsProcessor() runningStatsProcessor {
	return &runningStatsFirstProcessor{}
}

type runningStatsFirstProcessor struct {
	value    string
	rowsSeen int
}

func (sfp *runningStatsFirstProcessor) updateRunningStats(rsf runningStatsFunc, row []Field) {
	sf := rsf.(*runningStatsFirst)

	if sfp.rowsSeen == sf.offset {
		for i := range row {
			f := &row[i]
			if f.Name == sf.fieldName {
				sfp.value = strings.Clone(f.Value)
				break
			}
		}
	}
	sfp.rowsSeen++
}

func (sfp *runningStatsFirstProcessor) getRunningStats() string {
	return sfp.value
}

func parseRunningStatsFirst(lex *lexer) (runningStatsFunc, error) {
	args, err := parseStatsFuncArgs(lex, "first")
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("missing the field name inside first()")
	}
	if len(args) > 2 {
		return nil, fmt.Errorf("too many args for the first() function; got %d; want 1 or 2 args; args: %q", len(args), args)
	}

	fieldName := args[0]

	offset := 0
	if len(args) == 2 {
		offsetStr := args[1]
		n, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, fmt.Errorf("cannot parse offset=%q at first(%q, %q): %w", offsetStr, fieldName, offsetStr, err)
		}
		if n < 0 {
			return nil, fmt.Errorf("offset=%d cannot be negative at first(%q, %q)", n, fieldName, offsetStr)
		}
		offset = n
	}

	sf := &runningStatsFirst{
		fieldName: fieldName,
		offset:    offset,
	}
	return sf, nil
}
