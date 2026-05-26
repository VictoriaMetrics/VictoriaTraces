package traceql

import (
	"regexp"
	"strconv"
	"strings"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// statusNameRegex matches the three TraceQL status name keywords at word
// boundaries so they can be rewritten into their numeric OTel StatusCode
// equivalents inside regex patterns on the `status` field.
var statusNameRegex = regexp.MustCompile(`(?i)\b(unset|ok|error)\b`)

func rewriteStatusNamesInRegex(s string) string {
	return statusNameRegex.ReplaceAllStringFunc(s, func(m string) string {
		return statusValueMap[strings.ToLower(m)]
	})
}

type filterCommon struct {
	fieldName string
	op        string
	value     string
}

// statusValueMap maps TraceQL status names to OTEL StatusCode numeric values.
var statusValueMap = map[string]string{
	"unset": "0",
	"ok":    "1",
	"error": "2",
}

// statusCodeMap is the reverse of statusValueMap.
var statusCodeMap = func() map[string]string {
	m := make(map[string]string, len(statusValueMap))
	for name, code := range statusValueMap {
		m[code] = name
	}
	return m
}()

// streamFieldMap contains the field names of stream fields defined by VictoriaTraces.
var streamFieldMap = map[string]bool{
	otelpb.ResourceAttrServiceName: true,
	otelpb.NameField:               true,
}

// StatusCodeToName converts a numeric OTEL StatusCode ("2") to its TraceQL name ("error").
// Returns the input unchanged if not a known status code.
func StatusCodeToName(code string) string {
	if name, ok := statusCodeMap[code]; ok {
		return name
	}
	return code
}

func (fc *filterCommon) String() string {
	// traceDuration must be treated as pipe
	if fc.fieldName == "traceDuration" {
		return "*"
	}

	// nestedSetParent<0 is Tempo's way to select root spans.
	// Combine an empty parent_span_id check with trace_id:in(<subquery>) — the
	// subquery pulls trace IDs from the trace ID index stream (one entry per
	// trace with has_root_span=1). This narrows the span scan to only traces
	// known to contain a root span, orders of magnitude faster than scanning
	// parent_span_id across all spans.
	if fc.fieldName == "nestedSetParent" && fc.op == "<" && fc.value == "0" {
		return otelpb.ParentSpanIDField + `:=""`
	}

	// TraceQL's `attr = nil` / `attr != nil` map to LogsQL's empty-value and
	// any-value filters respectively — the canonical forms per
	// https://docs.victoriametrics.com/victorialogs/logsql/#empty-value-filter
	// and https://docs.victoriametrics.com/victorialogs/logsql/#any-value-filter.
	if fc.value == "nil" {
		switch fc.op {
		case "=":
			return quoteFieldNameIfNeeded(fc.tagToVTField()) + `:""`
		case "!=":
			return quoteFieldNameIfNeeded(fc.tagToVTField()) + ":*"
		}
	}

	fieldName := fc.tagToVTField()
	fieldValue := fc.value

	// map status names (error, ok, unset) to numeric OTEL StatusCode values.
	if fieldName == "status_code" {
		fieldValue = rewriteStatusNamesInRegex(fieldValue)
	}

	// translate duration to nanosecond.
	if duration, ok := tryParseDuration(fieldValue); ok {
		fieldValue = strconv.FormatInt(duration, 10)
	}

	// for stream filter, just use the source op (=, !=, =~, !~), as they're identical in LogsQL.
	if isStreamField(fieldName) && (fc.op == "=" || fc.op == "!=" || fc.op == "=~" || fc.op == "!~") {
		return `{` + quoteFieldNameIfNeeded(fieldName) + fc.op + quoteTokenIfNeeded(fieldValue) + `}`
	}

	// regex ops translate to LogsQL's :~ / :!~ filters.
	if fc.op == "=~" || fc.op == "!~" {
		op := ":~"
		if fc.op == "!~" {
			op = ":!~"
		}
		return quoteFieldNameIfNeeded(fieldName) + op + strconv.Quote(fieldValue)
	}

	return quoteFieldNameIfNeeded(fieldName) + ":" + fc.op + quoteTokenIfNeeded(fieldValue)
}

func (fc *filterCommon) tagToVTField() string {
	return TraceQLFieldToVTField(fc.fieldName)
}

// TraceQLFieldToVTField converts a TraceQL field name to a VictoriaTraces internal field name.
// e.g., "resource.service.name" -> "resource_attr:service.name"
//
//	"span.http.status_code" -> "span_attr:http.status_code"
//	"status"                -> "status_code"
func TraceQLFieldToVTField(fieldName string) string {
	if strings.HasPrefix(fieldName, "resource.") {
		return otelpb.ResourceAttrPrefix + fieldName[len("resource."):]
	} else if strings.HasPrefix(fieldName, "span.") {
		return otelpb.SpanAttrPrefixField + fieldName[len("span."):]
	} else if strings.HasPrefix(fieldName, "event.") {
		return otelpb.EventPrefix + otelpb.EventAttrPrefix + fieldName[len("event."):]
	} else if strings.HasPrefix(fieldName, "link.") {
		return otelpb.LinkPrefix + otelpb.LinkAttrPrefix + fieldName[len("link."):]
	} else if strings.HasPrefix(fieldName, "instrumentation.") {
		return otelpb.InstrumentationScopeAttrPrefix + fieldName[len("instrumentation."):]
	} else if fieldName == "status" {
		return otelpb.StatusCodeField
	} else if fieldName == "service.name" || fieldName == ".service.name" {
		return otelpb.ResourceAttrServiceName
	}

	return fieldName
}

// VTFieldToTraceQL converts a VictoriaTraces internal field name back to a TraceQL field name.
// e.g., "resource_attr:service.name" -> "resource.service.name"
//
//	"span_attr:http.status_code" -> "span.http.status_code"
//	"status_code"                -> "status"
func VTFieldToTraceQL(fieldName string) string {
	if strings.HasPrefix(fieldName, otelpb.ResourceAttrPrefix) {
		return "resource." + fieldName[len(otelpb.ResourceAttrPrefix):]
	} else if strings.HasPrefix(fieldName, otelpb.SpanAttrPrefixField) {
		return "span." + fieldName[len(otelpb.SpanAttrPrefixField):]
	} else if strings.HasPrefix(fieldName, otelpb.EventPrefix+otelpb.EventAttrPrefix) {
		return "event." + fieldName[len(otelpb.EventPrefix+otelpb.EventAttrPrefix):]
	} else if strings.HasPrefix(fieldName, otelpb.LinkPrefix+otelpb.LinkAttrPrefix) {
		return "link." + fieldName[len(otelpb.LinkPrefix+otelpb.LinkAttrPrefix):]
	} else if strings.HasPrefix(fieldName, otelpb.InstrumentationScopeAttrPrefix) {
		return "instrumentation." + fieldName[len(otelpb.InstrumentationScopeAttrPrefix):]
	} else if fieldName == otelpb.StatusCodeField {
		return "status"
	} else if fieldName == otelpb.ResourceAttrServiceName {
		return "resource.service.name"
	}

	return fieldName
}

func quoteFieldNameIfNeeded(s string) string {
	return quoteTokenIfNeeded(s)
}

func isStreamField(fieldName string) bool {
	return streamFieldMap[fieldName]
}

func (fc *filterCommon) GetTraceDurationFilters() []*filterCommon {
	if fc.fieldName == "traceDuration" {
		return []*filterCommon{fc}
	}
	return nil
}
