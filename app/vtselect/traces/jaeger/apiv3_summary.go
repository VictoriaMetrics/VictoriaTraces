package jaeger

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/tracecommon"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// statusCodeError is the value of the status_code field for a span which reports an error.
//
// See https://opentelemetry.io/docs/specs/otel/trace/api/#set-status
const statusCodeError = 2

// serviceSummaryV3 holds the span counts of a single service within a trace.
type serviceSummaryV3 struct {
	name           string
	spanCount      int
	errorSpanCount int
}

// traceSummaryV3 is the summary of a single trace returned by /api/v3/trace-summaries.
type traceSummaryV3 struct {
	traceID              string
	rootServiceName      string
	rootOperationName    string
	minStartTimeUnixNano uint64
	maxEndTimeUnixNano   uint64
	spanCount            int
	errorSpanCount       int
	orphanSpanCount      int
	services             []serviceSummaryV3
}

// summarySpan holds the fields of a span which are needed for building a trace summary.
type summarySpan struct {
	spanID            string
	parentSpanID      string
	name              string
	serviceName       string
	startTimeUnixNano uint64
	endTimeUnixNano   uint64
	isError           bool
}

// summarizeTraces groups rows by trace and reduces every trace to a traceSummaryV3.
//
// The result is sorted by the trace start time, newest first.
//
// The counting rules follow the Jaeger implementation, so that the numbers shown by Jaeger UI
// mean the same thing regardless of the backend which serves them.
// See https://github.com/jaegertracing/jaeger/blob/main/cmd/jaeger/internal/extension/jaegerquery/querysvc/summary.go
func summarizeTraces(rows []*tracecommon.Row) ([]traceSummaryV3, error) {
	traceIDs := make([]string, 0, len(rows))
	spansPerTrace := make(map[string][]summarySpan)
	for _, row := range rows {
		traceID, span, err := rowToSummarySpan(row)
		if err != nil {
			return nil, err
		}
		if _, ok := spansPerTrace[traceID]; !ok {
			traceIDs = append(traceIDs, traceID)
		}
		spansPerTrace[traceID] = append(spansPerTrace[traceID], span)
	}

	summaries := make([]traceSummaryV3, 0, len(traceIDs))
	for _, traceID := range traceIDs {
		summaries = append(summaries, summarizeTrace(traceID, spansPerTrace[traceID]))
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].minStartTimeUnixNano > summaries[j].minStartTimeUnixNano
	})
	return summaries, nil
}

// rowToSummarySpan extracts the trace id and the summary fields from a single span row.
func rowToSummarySpan(row *tracecommon.Row) (string, summarySpan, error) {
	var traceID string
	var span summarySpan
	for _, field := range row.Fields {
		var err error
		switch field.Name {
		case otelpb.TraceIDField:
			traceID = field.Value
		case otelpb.SpanIDField:
			span.spanID = field.Value
		case otelpb.ParentSpanIDField:
			span.parentSpanID = field.Value
		case otelpb.NameField:
			span.name = field.Value
		case otelpb.ResourceAttrServiceName:
			span.serviceName = field.Value
		case otelpb.StartTimeUnixNanoField:
			span.startTimeUnixNano, err = strconv.ParseUint(field.Value, 10, 64)
			if err != nil {
				return "", span, fmt.Errorf("invalid %s field [%s]: %w", field.Name, field.Value, err)
			}
		case otelpb.EndTimeUnixNanoField:
			span.endTimeUnixNano, err = strconv.ParseUint(field.Value, 10, 64)
			if err != nil {
				return "", span, fmt.Errorf("invalid %s field [%s]: %w", field.Name, field.Value, err)
			}
		case otelpb.StatusCodeField:
			statusCode, err := strconv.ParseInt(field.Value, 10, 32)
			if err != nil {
				return "", span, fmt.Errorf("invalid %s field [%s]: %w", field.Name, field.Value, err)
			}
			span.isError = statusCode == statusCodeError
		}
	}
	if traceID == "" {
		return "", span, fmt.Errorf("cannot find %s field in the span row", otelpb.TraceIDField)
	}
	return traceID, span, nil
}

// summarizeTrace reduces the spans of a single trace to a traceSummaryV3.
func summarizeTrace(traceID string, spans []summarySpan) traceSummaryV3 {
	spanIDs := make(map[string]struct{}, len(spans))
	for _, span := range spans {
		spanIDs[span.spanID] = struct{}{}
	}

	summary := traceSummaryV3{traceID: traceID}
	serviceStats := make(map[string]*serviceSummaryV3)
	var rootStartTimeUnixNano uint64
	rootFound := false
	for _, span := range spans {
		stats, ok := serviceStats[span.serviceName]
		if !ok {
			stats = &serviceSummaryV3{name: span.serviceName}
			serviceStats[span.serviceName] = stats
		}
		stats.spanCount++
		summary.spanCount++
		if span.isError {
			stats.errorSpanCount++
			summary.errorSpanCount++
		}

		if summary.minStartTimeUnixNano == 0 || span.startTimeUnixNano < summary.minStartTimeUnixNano {
			summary.minStartTimeUnixNano = span.startTimeUnixNano
		}
		if span.endTimeUnixNano > summary.maxEndTimeUnixNano {
			summary.maxEndTimeUnixNano = span.endTimeUnixNano
		}

		if span.parentSpanID == "" {
			// a trace may contain more than one span without a parent, so the earliest one wins.
			if !rootFound || span.startTimeUnixNano < rootStartTimeUnixNano {
				summary.rootServiceName = span.serviceName
				summary.rootOperationName = span.name
				rootStartTimeUnixNano = span.startTimeUnixNano
				rootFound = true
			}
		} else if _, ok := spanIDs[span.parentSpanID]; !ok {
			// the parent of this span is missing from the trace.
			summary.orphanSpanCount++
		}
	}

	summary.services = make([]serviceSummaryV3, 0, len(serviceStats))
	for _, stats := range serviceStats {
		summary.services = append(summary.services, *stats)
	}
	sort.Slice(summary.services, func(i, j int) bool {
		return summary.services[i].name < summary.services[j].name
	})
	return summary
}
