package insertutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/metrics"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// traceSpanProcessor is a wrapper logMessageProcessor.
type traceSpanProcessor struct {
	lmp *logMessageProcessor
}

// NewTraceProcessor returns new TraceSpansProcessor for the given cp.
//
// MustClose() must be called on the returned TraceSpansProcessor when it is no longer needed.
func (cp *CommonParams) NewTraceProcessor(protocolName string, isStreamMode bool) LogMessageProcessor {
	lr := logstorage.GetLogRows(cp.StreamFields, cp.IgnoreFields, cp.DecolorizeFields, cp.ExtraFields, *defaultMsgValue)
	rowsIngestedTotal := metrics.GetOrCreateCounter(fmt.Sprintf("vt_rows_ingested_total{type=%q}", protocolName))
	bytesIngestedTotal := metrics.GetOrCreateCounter(fmt.Sprintf("vt_bytes_ingested_total{type=%q}", protocolName))
	flushDuration := metrics.GetOrCreateSummary(fmt.Sprintf("vt_insert_flush_duration_seconds{type=%q}", protocolName))
	tsp := &traceSpanProcessor{
		lmp: &logMessageProcessor{
			cp: cp,
			lr: lr,

			rowsIngestedTotal:  rowsIngestedTotal,
			bytesIngestedTotal: bytesIngestedTotal,
			flushDuration:      flushDuration,

			stopCh: make(chan struct{}),
		},
	}

	if isStreamMode {
		tsp.lmp.initPeriodicFlush()
	}

	messageProcessorCount.Add(1)
	return tsp
}

// The following methods are for external data ingestion. They could be called from OTLP handlers.

// AddRow adds new log message to lmp with the given timestamp and fields.
// It also creates index if the current process is a storage node (VictoriaTraces Single-node).
//
// If streamFields is non-nil, then it is used as log stream fields instead of the pre-configured stream fields.
func (tsp *traceSpanProcessor) AddRow(timestamp int64, fields []logstorage.Field, streamFieldsLen int) {
	if logRowsStorage.IsLocalStorage() {
		if !tsp.pushTraceToIndexQueue(tsp.lmp.cp.TenantID, fields) {
			// This should not happen because:
			// 1. AddRow is called by HTTP handler.
			// 2. During shutdown, main HTTP server should be closed first, and then the index worker.
			// 3. `false` only return when index worker is exited.
			metrics.GetOrCreateCounter("vt_traceid_index_push_error_total").Inc()
			logger.Errorf("cannot push index for a trace to the queue: %v", fields)
			return
		}
	}
	tsp.lmp.AddRow(timestamp, fields, streamFieldsLen)
}

func (tsp *traceSpanProcessor) MustClose() {
	tsp.lmp.MustClose()
}

// pushTraceToIndexQueue is for trace from LogMessageProcessor. It adds trace ID, startTimeNano, endTimeNano of the span to the FIFO queue.
// Each item in the queue will be popped after certain interval, and carries the min(startTimeNano), max(endTimeNano) of this trace ID.
func (tsp *traceSpanProcessor) pushTraceToIndexQueue(tenant logstorage.TenantID, fields []logstorage.Field) bool {
	var (
		traceID            string
		startTime, endTime int64
		err                error
	)

	i := len(fields) - 1
	// find trace ID in reverse order.
	for ; i >= 0; i-- {
		if fields[i].Name == otelpb.TraceIDField {
			traceID = strings.Clone(fields[i].Value)
			break
		}
	}

	if traceID == "" {
		return false
	}

	// find endTimeNano of the span in reverse order, it should be right before trace ID field.
	for i = i - 1; i >= 0; i-- {
		if fields[i].Name == otelpb.EndTimeUnixNanoField {
			endTime, err = strconv.ParseInt(fields[i].Value, 10, 64)
			if err != nil {
				logger.Errorf("cannot parse endTime %s for traceID %q: %v", fields[i].Value, traceID, err)
				return false
			}
			break
		}
	}

	// find startTimeNano of the span in reverse order, it should be right before endTimeNano field.
	for i = i - 1; i >= 0; i-- {
		if fields[i].Name == otelpb.StartTimeUnixNanoField {
			startTime, err = strconv.ParseInt(fields[i].Value, 10, 64)
			if err != nil {
				logger.Errorf("cannot parse startTime %s for traceID %q: %v", fields[i].Value, traceID, err)
				return false
			}
			break
		}
	}

	// to secure an index entry will be created even if the span does not have startTimeNano and endTimeNano.
	if startTime == 0 {
		startTime = time.Now().UnixNano()
	}
	if endTime == 0 {
		endTime = time.Now().UnixNano()
	}

	return pushIndexToQueue(tenant, traceID, startTime, endTime)
}

// The following methods are for native data ingestion protocol. They must be called from `internalinsert`.

// AddInsertRow is a wrapper function of logMessageProcessor.AddInsertRow.
// while processing trace spans as logs, traceSpanProcessor also create trace ID index for each new trace ID.
func (tsp *traceSpanProcessor) AddInsertRow(r *logstorage.InsertRow) {
	// create index <traceID, startTimeNano, endTimeNano> if the current process is a storage node (VictoriaTraces single-node, or vtstorage).
	if logRowsStorage.IsLocalStorage() && !tsp.pushNativeRowToIndexQueue(r) {
		metrics.GetOrCreateCounter("vt_traceid_index_push_error_total").Inc()
		logger.Errorf("cannot push index for a native insert trace to the queue: %v", r.Fields)
		return
	}
	tsp.lmp.AddInsertRow(r)
}

// pushNativeRowToIndexQueue is for native data ingestion protocol. It adds trace ID, startTimeNano, endTimeNano of the span to the FIFO queue.
// Each item in the queue will be popped after certain interval, and carries the min(startTimeNano), max(endTimeNano) of this trace ID.
func (tsp *traceSpanProcessor) pushNativeRowToIndexQueue(r *logstorage.InsertRow) bool {
	var (
		traceID            string
		startTime, endTime int64
		err                error
	)

	i := len(r.Fields) - 1
	// find trace ID in reverse order.
	for ; i >= 0; i-- {
		if r.Fields[i].Name == otelpb.TraceIDField {
			traceID = strings.Clone(r.Fields[i].Value)
			break
		}
	}

	if traceID == "" {
		logger.Errorf("cannot push index for a trace to the queue: cannot find the trace ID of an insert row: %v", r)
		return false
	}

	// find endTimeNano of the span in reverse order, it should be right before trace ID field.
	for i = i - 1; i >= 0; i-- {
		if r.Fields[i].Name == otelpb.EndTimeUnixNanoField {
			endTime, err = strconv.ParseInt(r.Fields[i].Value, 10, 64)
			if err != nil {
				logger.Errorf("cannot parse endTime %s for traceID %q: %v", r.Fields[i].Value, traceID, err)
				return false
			}
			break
		}
	}

	// find startTimeNano of the span in reverse order, it should be right before endTimeNano field.
	for i = i - 1; i >= 0; i-- {
		if r.Fields[i].Name == otelpb.StartTimeUnixNanoField {
			startTime, err = strconv.ParseInt(r.Fields[i].Value, 10, 64)
			if err != nil {
				logger.Errorf("cannot parse startTime %s for traceID %q: %v", r.Fields[i].Value, traceID, err)
				return false
			}
			break
		}
	}

	// to secure an index entry will be created even if the span does not have startTimeNano and endTimeNano.
	if startTime == 0 {
		startTime = time.Now().UnixNano()
	}
	if endTime == 0 {
		endTime = time.Now().UnixNano()
	}

	return pushIndexToQueue(r.TenantID, traceID, startTime, endTime)
}
