package traceql

import (
	"fmt"
	"strings"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// filterAnd contains filters joined by AND operator.
//
// Inside a spanset (`{...}`), it is expressed as `f1 and f2 ... and fN` on a single span.
// Between spansets (`{...} && {...}`), it matches traces that have spans satisfying every operand.
type filterAnd struct {
	filters   []filter
	isSpanset bool
}

func (fa *filterAnd) String() string {
	if fa.isSpanset {
		return fa.spansetString()
	}
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

func (fa *filterAnd) spansetString() string {
	nonWildcard := make([]string, 0, len(fa.filters))
	for _, f := range fa.filters {
		s := f.String()
		if s == "" || s == "*" {
			continue
		}
		nonWildcard = append(nonWildcard, s)
	}
	if len(nonWildcard) == 0 {
		return "*"
	}
	if len(nonWildcard) == 1 {
		return nonWildcard[0]
	}

	orParts := make([]string, len(nonWildcard))
	inParts := make([]string, len(nonWildcard))
	for i, s := range nonWildcard {
		orParts[i] = "(" + s + ")"
		inParts[i] = fmt.Sprintf(`%s:in({%s!=""} AND (%s) | uniq by (%s))`, otelpb.TraceIDField, otelpb.ResourceAttrServiceName, s, otelpb.TraceIDField)
	}
	return "(" + strings.Join(orParts, " or ") + ") and " + strings.Join(inParts, " and ")
}

func (fa *filterAnd) GetTraceDurationFilters() []*filterCommon {
	result := make([]*filterCommon, 0)
	for _, f := range fa.filters {
		result = append(result, f.GetTraceDurationFilters()...)
	}
	return result
}
