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
		s := f.String()
		a[i] = s
	}
	return strings.Join(a, " or ")
}

func (fo *filterOr) GetTraceDurationFilters() []*filterCommon {
	result := make([]*filterCommon, 0)
	for _, f := range fo.filters {
		result = append(result, f.GetTraceDurationFilters()...)
	}
	return result
}
