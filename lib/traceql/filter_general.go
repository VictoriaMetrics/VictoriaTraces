package traceql

import (
	"strconv"
	"strings"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

type filterCommon struct {
	fieldName string
	op        string
	value     string
}

func (fc *filterCommon) String() string {
	// traceDuration must be treated as pipe
	if fc.fieldName == "traceDuration" {
		return "*"
	}

	v := fc.value
	if duration, ok := tryParseDuration(v); ok {
		v = strconv.FormatInt(duration, 10)
	}
	return quoteFieldNameIfNeeded(fc.tagToVTField()) + ":" + fc.op + quoteTokenIfNeeded(v)
}

func (fc *filterCommon) tagToVTField() string {
	if strings.HasPrefix(fc.fieldName, "resource.") {
		return otelpb.ResourceAttrPrefix + fc.fieldName[len("resource."):]
	} else if strings.HasPrefix(fc.fieldName, "span.") {
		return otelpb.SpanAttrPrefixField + fc.fieldName[len("span."):]
	} else if strings.HasPrefix(fc.fieldName, "event.") {
		return otelpb.EventPrefix + otelpb.EventAttrPrefix + fc.fieldName[len("event."):]
	} else if strings.HasPrefix(fc.fieldName, "link.") {
		return otelpb.LinkPrefix + otelpb.LinkAttrPrefix + fc.fieldName[len("link."):]
	} else if strings.HasPrefix(fc.fieldName, "instrumentation.") {
		return otelpb.InstrumentationScopeAttrPrefix + fc.fieldName[len("instrumentation."):]
	} else if fc.fieldName == "status" {
		return otelpb.StatusCodeField
	} else if fc.fieldName == "service.name" || fc.fieldName == ".service.name" {
		return otelpb.ResourceAttrServiceName
	}

	return fc.fieldName
}

func quoteFieldNameIfNeeded(s string) string {
	return quoteTokenIfNeeded(s)
}

func (fc *filterCommon) GetTraceDurationFilters() []*filterCommon {
	if fc.fieldName == "traceDuration" {
		return []*filterCommon{fc}
	}
	return nil
}
