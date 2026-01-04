package opentelemetry

import (
	"context"
	"strconv"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/flagutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/fastcache"
	"github.com/cespare/xxhash/v2"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/insertutil"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

var maxRequestSize = flagutil.NewBytes("opentelemetry.traces.maxRequestSize", 64*1024*1024, "The maximum size in bytes of a single OpenTelemetry trace export request.")

var (
	mandatoryStreamFields = []string{otelpb.ResourceAttrServiceName, otelpb.NameField}
	msgFieldValue         = "-"
)

var (
	// traceIDCache for deduplicating trace_id
	traceIDCache = fastcache.New(32 * 1024 * 1024)
)

// pushExportTraceServiceRequest is the entry point of OTLP data processing. It should be called by different
// request handlers such as OTLPHTTP handler, OTLPgRPC handler.
func pushExportTraceServiceRequest(req *otelpb.ExportTraceServiceRequest, lmp insertutil.LogMessageProcessor) error {
	var commonFields []logstorage.Field
	for _, rs := range req.ResourceSpans {
		commonFields = commonFields[:0]
		attributes := rs.Resource.Attributes
		commonFields = appendKeyValuesWithPrefix(commonFields, attributes, "", otelpb.ResourceAttrPrefix)
		commonFieldsLen := len(commonFields)
		for _, ss := range rs.ScopeSpans {
			commonFields = pushFieldsFromScopeSpans(ss, commonFields[:commonFieldsLen], lmp)
		}
	}
	return nil
}

func pushFieldsFromScopeSpans(ss *otelpb.ScopeSpans, commonFields []logstorage.Field, lmp insertutil.LogMessageProcessor) []logstorage.Field {
	commonFields = append(commonFields, logstorage.Field{
		Name:  otelpb.InstrumentationScopeName,
		Value: ss.Scope.Name,
	}, logstorage.Field{
		Name:  otelpb.InstrumentationScopeVersion,
		Value: ss.Scope.Version,
	})
	commonFields = appendKeyValuesWithPrefix(commonFields, ss.Scope.Attributes, "", otelpb.InstrumentationScopeAttrPrefix)
	commonFieldsLen := len(commonFields)
	for _, span := range ss.Spans {
		commonFields = pushFieldsFromSpan(span, commonFields[:commonFieldsLen], lmp)
	}
	return commonFields
}

func pushFieldsFromSpan(span *otelpb.Span, scopeCommonFields []logstorage.Field, lmp insertutil.LogMessageProcessor) []logstorage.Field {
	fields := scopeCommonFields
	fields = append(fields,
		logstorage.Field{Name: otelpb.SpanIDField, Value: span.SpanID},
		logstorage.Field{Name: otelpb.TraceStateField, Value: span.TraceState},
		logstorage.Field{Name: otelpb.ParentSpanIDField, Value: span.ParentSpanID},
		logstorage.Field{Name: otelpb.FlagsField, Value: strconv.FormatUint(uint64(span.Flags), 10)},
		logstorage.Field{Name: otelpb.NameField, Value: span.Name},
		logstorage.Field{Name: otelpb.KindField, Value: strconv.FormatInt(int64(span.Kind), 10)},
		logstorage.Field{Name: otelpb.StartTimeUnixNanoField, Value: strconv.FormatUint(span.StartTimeUnixNano, 10)},
		logstorage.Field{Name: otelpb.EndTimeUnixNanoField, Value: strconv.FormatUint(span.EndTimeUnixNano, 10)},
		logstorage.Field{Name: otelpb.DurationField, Value: strconv.FormatUint(span.EndTimeUnixNano-span.StartTimeUnixNano, 10)},

		logstorage.Field{Name: otelpb.DroppedAttributesCountField, Value: strconv.FormatUint(uint64(span.DroppedAttributesCount), 10)},
		logstorage.Field{Name: otelpb.DroppedEventsCountField, Value: strconv.FormatUint(uint64(span.DroppedEventsCount), 10)},
		logstorage.Field{Name: otelpb.DroppedLinksCountField, Value: strconv.FormatUint(uint64(span.DroppedLinksCount), 10)},

		logstorage.Field{Name: otelpb.StatusMessageField, Value: span.Status.Message},
		logstorage.Field{Name: otelpb.StatusCodeField, Value: strconv.FormatInt(int64(span.Status.Code), 10)},
	)

	// append span attributes
	fields = appendKeyValuesWithPrefix(fields, span.Attributes, "", otelpb.SpanAttrPrefixField)

	for idx, event := range span.Events {
		eventFieldPrefix := otelpb.EventPrefix
		eventFieldSuffix := ":" + strconv.Itoa(idx)
		fields = append(fields,
			logstorage.Field{Name: eventFieldPrefix + otelpb.EventTimeUnixNanoField + eventFieldSuffix, Value: strconv.FormatUint(event.TimeUnixNano, 10)},
			logstorage.Field{Name: eventFieldPrefix + otelpb.EventNameField + eventFieldSuffix, Value: event.Name},
			logstorage.Field{Name: eventFieldPrefix + otelpb.EventDroppedAttributesCountField + eventFieldSuffix, Value: strconv.FormatUint(uint64(event.DroppedAttributesCount), 10)},
		)
		// append event attributes
		fields = appendKeyValuesWithPrefixSuffix(fields, event.Attributes, "", eventFieldPrefix+otelpb.EventAttrPrefix, eventFieldSuffix)
	}

	for idx, link := range span.Links {
		linkFieldPrefix := otelpb.LinkPrefix
		linkFieldSuffix := ":" + strconv.Itoa(idx)
		fields = append(fields,
			logstorage.Field{Name: linkFieldPrefix + otelpb.LinkTraceIDField + linkFieldSuffix, Value: link.TraceID},
			logstorage.Field{Name: linkFieldPrefix + otelpb.LinkSpanIDField + linkFieldSuffix, Value: link.SpanID},
			logstorage.Field{Name: linkFieldPrefix + otelpb.LinkTraceStateField + linkFieldSuffix, Value: link.TraceState},
			logstorage.Field{Name: linkFieldPrefix + otelpb.LinkDroppedAttributesCountField + linkFieldSuffix, Value: strconv.FormatUint(uint64(link.DroppedAttributesCount), 10)},
			logstorage.Field{Name: linkFieldPrefix + otelpb.LinkFlagsField + linkFieldSuffix, Value: strconv.FormatUint(uint64(link.Flags), 10)},
		)

		// append link attributes
		fields = appendKeyValuesWithPrefixSuffix(fields, link.Attributes, "", linkFieldPrefix+otelpb.LinkAttrPrefix, linkFieldSuffix)
	}
	fields = append(fields,
		logstorage.Field{Name: "_msg", Value: msgFieldValue},
		// MUST: always place TraceIDField at the last. The Trace ID is required for data distribution.
		// Placing it at the last position helps netinsert to find it easily, without adding extra field to
		// *logstorage.InsertRow structure, which is required due to the sync between logstorage and VictoriaTraces.
		// todo: @jiekun the trace ID field MUST be the last field. add extra ways to secure it.
		logstorage.Field{Name: otelpb.TraceIDField, Value: span.TraceID},
	)

	// Create an entry in the trace-id-idx stream if this trace_id hasn't been seen before.
	// The index entry must be written first to ensure that an index always exists for the data.
	// During querying, if no index is found, the data must not exist.
	if !traceIDCache.Has([]byte(span.TraceID)) {
		lmp.AddRow(int64(span.StartTimeUnixNano), []logstorage.Field{
			{Name: otelpb.TraceIDIndexStreamName, Value: strconv.FormatUint(xxhash.Sum64String(span.TraceID)%otelpb.TraceIDIndexPartitionCount, 10)},
			{Name: "_msg", Value: msgFieldValue},
			// todo: @jiekun the trace ID field MUST be the last field. add extra ways to secure it.
			{Name: otelpb.TraceIDIndexFieldName, Value: span.TraceID},
		}, 1)
		traceIDCache.Set([]byte(span.TraceID), nil)
	}

	lmp.AddRow(int64(span.EndTimeUnixNano), fields, -1)

	return fields
}

func appendKeyValuesWithPrefix(fields []logstorage.Field, kvs []*otelpb.KeyValue, parentField, prefix string) []logstorage.Field {
	return appendKeyValuesWithPrefixSuffix(fields, kvs, parentField, prefix, "")
}

func appendKeyValuesWithPrefixSuffix(fields []logstorage.Field, kvs []*otelpb.KeyValue, parentField, prefix, suffix string) []logstorage.Field {
	for _, attr := range kvs {
		fieldName := attr.Key
		if parentField != "" {
			fieldName = parentField + "." + fieldName
		}

		if attr.Value.KeyValueList != nil {
			fields = appendKeyValuesWithPrefixSuffix(fields, attr.Value.KeyValueList.Values, fieldName, prefix, suffix)
			continue
		}

		v := attr.Value.FormatString(true)
		if len(v) == 0 {
			// VictoriaLogs does not support empty string as field value. set it to "-" to preserve the field.
			v = "-"
		}
		fields = append(fields, logstorage.Field{
			Name:  prefix + fieldName + suffix,
			Value: v,
		})
	}
	return fields
}

func PersistServiceGraph(ctx context.Context, tenantID logstorage.TenantID, commonFields []logstorage.Field, fields [][]logstorage.Field, timestamp time.Time) ([]logstorage.Field, error) {
	cp := insertutil.CommonParams{
		TenantID:   tenantID,
		TimeFields: []string{"_time"},
	}
	lmp := cp.NewLogMessageProcessor("internalinsert_servicegraph", false)
	commonFieldLen := len(commonFields)
	for _, row := range fields {
		commonFields = commonFields[:commonFieldLen]
		commonFields = append(commonFields, row...)
		commonFields = append(commonFields, logstorage.Field{Name: "_msg", Value: msgFieldValue})
		lmp.AddRow(timestamp.UnixNano(), commonFields, commonFieldLen)
	}
	lmp.MustClose()
	return commonFields, nil
}

func NewPushSpansCallbackFunc(lmp insertutil.LogMessageProcessor) func(timestamp int64, fields []logstorage.Field) {
	return func(timestamp int64, fields []logstorage.Field) {
		// search for traceID first. It should be at the tail of the fields.
		var traceID string
		for i := len(fields) - 1; i >= 0; i-- {
			if fields[i].Name == otelpb.TraceIDField {
				traceID = fields[i].Value
				break
			}
		}
		if traceID == "" {
			// not a valid trace
			logger.Errorf("cannot find traceID in span. skipping this span: %v", fields)
			return
		}

		if !traceIDCache.Has([]byte(traceID)) {
			// Create an entry in the trace-id-idx stream if this trace_id hasn't been seen before.
			// The index entry must be written first to ensure that an index always exists for the data.
			// During querying, if no index is found, the data must not exist.
			lmp.AddRow(int64(timestamp), []logstorage.Field{
				{Name: otelpb.TraceIDIndexStreamName, Value: strconv.FormatUint(xxhash.Sum64String(traceID)%otelpb.TraceIDIndexPartitionCount, 10)},
				{Name: "_msg", Value: msgFieldValue},
				// todo: @jiekun the trace ID field MUST be the last field. add extra ways to secure it.
				{Name: otelpb.TraceIDIndexFieldName, Value: traceID},
			}, 1)
			traceIDCache.Set([]byte(traceID), nil)
		}

		lmp.AddRow(timestamp, fields, -1)
	}
}
