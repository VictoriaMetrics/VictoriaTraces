package traceql

import (
	"strings"
)

// filterOr contains filters joined by OR operator.
//
// It is expressed as `f1 OR f2 ... OR fN` in LogsQL.
type filterOr struct {
	filters []filter
}

func (fo *filterOr) String() string {
	filters := fo.filters
	a := make([]string, len(filters))
	for i, f := range filters {
		a[i] = filterStringInlineDuration(f)
	}
	return strings.Join(a, " or ")
}

func (fo *filterOr) GetTraceDurationFilters() []*filterCommon {
	// Any trace duration predicate inside an OR branch is rendered inline via
	// trace_id:in(...) in String() so extracting it into a top-level join would
	// not convert the OR branch into a global AND filter.
	return nil
}

// filterStringInlineDuration renders f while keeping any traceDuration /
// trace:duration leaves inline at their boolean position.
func filterStringInlineDuration(f filter) string {
	switch t := f.(type) {
	case *filterCommon:
		if isTraceDurationField(t.fieldName) {
			return t.inlineTraceDurationString()
		}
		return t.String()
	case *filterAnd:
		a := make([]string, len(t.filters))
		for i, sub := range t.filters {
			s := filterStringInlineDuration(sub)
			if _, ok := sub.(*filterOr); ok {
				s = "(" + s + ")"
			}
			a[i] = s
		}
		return strings.Join(a, " and ")
	case *filterOr:
		return t.String()
	case *filterTrace:
		s := filterStringInlineDuration(t.andFilter)
		if s == "" {
			return "*"
		}
		return s
	default:
		return f.String()
	}
}
