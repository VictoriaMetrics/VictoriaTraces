package traceql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

type filterCommon struct {
	fieldName string
	op        string
	value     string
}

// statusNameRegex matches the three TraceQL status name keywords at word
// boundaries so they can be rewritten into their numeric OTel StatusCode
// equivalents inside regex patterns on the `status` field.
var statusNameRegex = regexp.MustCompile(`(?i)\b(unset|ok|error)\b`)

func rewriteStatusNamesInRegex(s string) string {
	return statusNameRegex.ReplaceAllStringFunc(s, func(m string) string {
		return statusValueMap[strings.ToLower(m)]
	})
}

// statusValueMap maps TraceQL status names to OpenTelemetry StatusCode numeric values.
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

// StatusCodeToName converts a numeric OpenTelemetry StatusCode ("2") to its TraceQL name ("error").
// Returns the input unchanged if not a known status code.
func StatusCodeToName(code string) string {
	if name, ok := statusCodeMap[code]; ok {
		return name
	}
	return code
}

// streamFieldMap contains the field names of stream fields defined by VictoriaTraces.
var streamFieldMap = map[string]bool{
	otelpb.ResourceAttrServiceName: true,
	otelpb.NameField:               true,
}

// spanKindRegex matches the TraceQL SpanKind keywords at word
// boundaries so they can be rewritten into their numeric OpenTelemetry SpanKind
// equivalents inside regex patterns on the `kind` field.
var spanKindRegex = regexp.MustCompile(`(?i)\b(unspecified|internal|server|client|producer|consumer)\b`)

func rewriteSpanKindInRegex(s string) string {
	return spanKindRegex.ReplaceAllStringFunc(s, func(m string) string {
		return spanKindValueMap[strings.ToLower(m)]
	})
}

// spanKindValueMap maps TraceQL SpanKind to OpenTelemetry SpanKind numeric values.
var spanKindValueMap = map[string]string{
	"unspecified": "0",
	"internal":    "1",
	"server":      "2",
	"client":      "3",
	"producer":    "4",
	"consumer":    "5",
}

// spanKindMap is the reverse of spanKindValueMap.
var spanKindMap = func() map[string]string {
	m := make(map[string]string, len(spanKindValueMap))
	for name, code := range spanKindValueMap {
		m[code] = name
	}
	return m
}()

// SpanKindToName converts a numeric OpenTelemetry SpanKind ("2") to its TraceQL name ("server").
// Returns the input unchanged if not a known SpanKind.
func SpanKindToName(code string) string {
	if name, ok := spanKindMap[code]; ok {
		return name
	}
	return code
}

func isTraceDurationField(fieldName string) bool {
	return fieldName == "traceDuration" || fieldName == "trace:duration"
}

func (fc *filterCommon) rootSpanField() (string, bool) {
	switch fc.fieldName {
	case "rootServiceName", "trace:rootService":
		return otelpb.ResourceAttrServiceName, true
	case "rootName", "trace:rootName":
		return otelpb.NameField, true
	default:
		return "", false
	}
}

func (fc *filterCommon) String() string {
	// traceDuration / trace:duration must be treated as pipe
	if isTraceDurationField(fc.fieldName) {
		return "*"
	}

	// rootServiceName / trace:rootService / rootName / trace:rootName match trace
	// properties determined by the trace's root span (parent_span_id=""), so
	// match any span belonging to a trace whose root span satisfies the predicate.
	if targetField, ok := fc.rootSpanField(); ok {
		rootFilter := &filterCommon{
			fieldName: targetField,
			op:        fc.op,
			value:     fc.value,
		}
		return fmt.Sprintf(`%s:in({%s!=""} AND %s:="" AND %s | fields %s)`,
			otelpb.TraceIDField,
			otelpb.ResourceAttrServiceName,
			otelpb.ParentSpanIDField,
			rootFilter.String(),
			otelpb.TraceIDField,
		)
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

	if fieldName == "status_code" {
		// map status names (error, ok, unset) to numeric OpenTelemetry StatusCode values.
		fieldValue = rewriteStatusNamesInRegex(fieldValue)
	} else if fieldName == "kind" {
		// map span kind (server, client, producer, consumer...) to numeric OpenTelemetry SpanKind values.
		fieldValue = rewriteSpanKindInRegex(fieldValue)
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
	}
	switch fieldName {
	case "status", "span:status":
		return otelpb.StatusCodeField
	case "service.name", ".service.name":
		return otelpb.ResourceAttrServiceName
	case "span:name":
		return otelpb.NameField
	case "span:kind":
		return otelpb.KindField
	case "span:duration":
		return otelpb.DurationField
	case "span:id":
		return otelpb.SpanIDField
	case "span:parentID":
		return otelpb.ParentSpanIDField
	case "span:statusMessage", "statusMessage":
		return otelpb.StatusMessageField
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
	if isTraceDurationField(fc.fieldName) {
		return []*filterCommon{fc}
	}
	return nil
}
