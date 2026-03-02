package traceql

import (
	"strings"
)

// filterAnd contains filters joined by AND operator.
//
// It is expressed as `f1 AND f2 ... AND fN` in LogsQL.
type filterAnd struct {
	filters []filter
}

func (fa *filterAnd) String() string {
	filters := fa.filters
	a := make([]string, len(filters))
	for i, f := range filters {
		s := f.String()
		if _, ok := f.(*filterOr); ok {
			s = "(" + s + ")"
		}
		a[i] = s
	}
	return strings.Join(a, " and ")
}

func (fa *filterAnd) GetTraceDurationFilters() []*filterCommon {
	result := make([]*filterCommon, 0)
	for _, f := range fa.filters {
		result = append(result, f.GetTraceDurationFilters()...)
	}
	return result
}
