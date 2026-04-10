package tempo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/cespare/xxhash/v2"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/tracecommon"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage"
	vtstoragecommon "github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage/common"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/traceql"
)

// Q: Why the query part is separated from `app/vtselect/traces/query`
//
// A: The Tempo API in VictoriaTraces is experimental, and we observed some unstable structure in the query and response.
//
// e.g.
// The `/api/search` API returns `spanSet` and `spanSets` fields with identical data but in different structure at the same time,
// while `spanSet` is not mentioned on doc.
//
// To avoid polluting the stable Jaeger API implementation, we have temporarily placed all related queries under the tempo directory.
// But they could be ported back to `app/vtselect/traces/query` and unified with queries already there in the future.

// GetTraceList returns multiple traceIDs and spans of them in []*Row format.
// It searches for traceIDs first, and then search for the spans of these traceIDs.
// To not miss any spans on the edge, it extends both the start time and end time
// by *traceMaxDurationWindow.
//
// e.g.:
// 1. input time range: [00:00, 09:00]
// 2. found 20 trace id, and adjust time range to: [08:00, 09:00]
// 3. find spans on time range: [08:00-traceMaxDurationWindow, 09:00+traceMaxDurationWindow]
func GetTraceList(ctx context.Context, cp *tracecommon.CommonParams, filterQuery *traceql.Query, start, end time.Time, limit int64) ([]string, []*tracecommon.Row, error) {
	currentTime := time.Now()

	// query 1: * AND filter_conditions | last 1 by (_time) partition by (trace_id) | fields _time, trace_id | sort by (_time) desc
	traceIDs, startTime, err := getTraceIDList(ctx, cp, filterQuery, start, end, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("get trace id error: %w", err)
	}
	if len(traceIDs) == 0 {
		return nil, nil, nil
	}

	// query 2: trace_id:in(traceID, traceID, ...)
	qStr := fmt.Sprintf(otelpb.TraceIDField+":in(%s)", strings.Join(traceIDs, ","))
	q, err := logstorage.ParseQueryAtTimestamp(qStr, currentTime.UnixNano())
	if err != nil {
		return nil, nil, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}

	// adjust start time and end time with max duration window to make sure all spans are included.
	q.AddTimeFilter(startTime.Add(-*tracecommon.TraceMaxDurationWindow).UnixNano(), end.Add(*tracecommon.TraceMaxDurationWindow).UnixNano())

	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	cp.Query = q
	qctx := cp.NewQueryContext(ctxWithCancel)
	defer cp.UpdatePerQueryStatsMetrics()

	// search for trace spans and write to `rows []*Row`
	var rowsLock sync.Mutex
	var rows []*tracecommon.Row
	var missingTimeColumn atomic.Bool
	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		if missingTimeColumn.Load() {
			return
		}

		columns := db.GetColumns(false)
		clonedColumnNames := make([]string, len(columns))
		for i, c := range columns {
			clonedColumnNames[i] = strings.Clone(c.Name)
		}

		timestamps, ok := db.GetTimestamps(nil)
		if !ok {
			missingTimeColumn.Store(true)
			cancel()
			return
		}

		for i, timestamp := range timestamps {
			fields := make([]logstorage.Field, 0, len(columns))
			for j := range columns {
				// column could be empty if this span does not contain such field.
				// only append non-empty columns.
				if columns[j].Values[i] != "" {
					fields = append(fields, logstorage.Field{Name: clonedColumnNames[j], Value: strings.Clone(columns[j].Values[i])})
				}
			}

			rowsLock.Lock()
			rows = append(rows, &tracecommon.Row{
				Timestamp: timestamp,
				Fields:    fields,
			})
			rowsLock.Unlock()
		}
	}

	if err = vtstorage.RunQuery(qctx, writeBlock); err != nil {
		return nil, nil, err
	}
	if missingTimeColumn.Load() {
		return nil, nil, fmt.Errorf("missing _time column in the result for the query [%s]", q)
	}
	return traceIDs, rows, nil
}

func getTraceIDList(ctx context.Context, cp *tracecommon.CommonParams, filterQuery *traceql.Query, start, end time.Time, limit int64) ([]string, time.Time, error) {
	qStr := `{trace_id_idx_stream=""} AND ` + filterQuery.String() + ` | last 1 by (_time) partition by (` + otelpb.TraceIDField + ") | fields _time, " + otelpb.TraceIDField + " | sort by (_time) desc"

	q, err := logstorage.ParseQueryAtTimestamp(qStr, end.UnixNano())
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}
	q.AddPipeOffsetLimit(0, uint64(limit))

	// adjust the max start time, because fresh traces may not be completed.
	// they should wait for *latencyOffset before being visible. currently hardcoded as 1m.
	maxStartTime := time.Now().Add(-*tracecommon.LatencyOffset)
	if end.After(maxStartTime) {
		end = maxStartTime
	}
	traceIDs, maxStartTime, err := findTraceIDsSplitTimeRange(ctx, q, cp, start, end, limit)
	if err != nil {
		return nil, time.Time{}, err
	}

	return traceIDs, maxStartTime, nil
}

// GetTrace returns all spans of a trace in []*Row format.
// It searches in the index stream for start_time and end_time.
// If found:
// - search for span in time range [start_time, end_time].
func GetTrace(ctx context.Context, cp *tracecommon.CommonParams, traceID string, start, end time.Time) ([]*tracecommon.Row, error) {
	currentTime := time.Now()

	// possible partition
	// query: {trace_id_idx="xx"} AND trace_id:traceID
	qStr := fmt.Sprintf(
		`{%s="%d"} AND %s:=%q | stats min(_time) _time, min(%s) %s, max(%s) %s`,
		otelpb.TraceIDIndexStreamName,
		xxhash.Sum64String(traceID)%otelpb.TraceIDIndexPartitionCount,
		otelpb.TraceIDIndexFieldName,
		traceID,
		otelpb.TraceIDIndexStartTimeFieldName, otelpb.TraceIDIndexStartTimeFieldName,
		otelpb.TraceIDIndexEndTimeFieldName, otelpb.TraceIDIndexEndTimeFieldName,
	)
	q, err := logstorage.ParseQueryAtTimestamp(qStr, currentTime.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal query=%q: %w", qStr, err)
	}
	q.AddPipeOffsetLimit(0, 10)
	traceStartTime, traceEndTime, err := findTraceIDTimeSplitTimeRange(ctx, q, cp, start, end)
	if err != nil && errors.Is(err, vtstoragecommon.ErrOutOfRetention) {
		// no hit in the retention period, simply returns empty.
		return nil, nil
	}
	if err != nil {
		// something wrong when trying to find the trace_id's start and end time.
		return nil, fmt.Errorf("cannot find trace_id %q start time: %s", traceID, err)
	}

	// trace start time found, search in [trace start time, trace start time + *traceMaxDurationWindow] time range.
	return findSpansByTraceIDAndTime(ctx, cp, traceID, traceStartTime, traceEndTime)
}

// findTraceIDsSplitTimeRange try to search from the nearest time range of the end time.
// if the result already met requirement of `limit`, return.
// otherwise, amplify the time range to 5x and search again, until the start time exceed the input.
func findTraceIDsSplitTimeRange(ctx context.Context, q *logstorage.Query, cp *tracecommon.CommonParams, startTime, endTime time.Time, limit int64) ([]string, time.Time, error) {
	currentTime := time.Now()

	step := time.Minute
	currentStartTime := endTime.Add(-step)

	var traceIDListLock sync.Mutex
	var startTimeLock sync.Mutex
	traceIDList := make([]string, 0, limit)
	maxStartTimeStr := endTime.Format(time.RFC3339)

	cp.Query = q
	qctx := cp.NewQueryContext(ctx)
	defer cp.UpdatePerQueryStatsMetrics()

	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		columns := db.GetColumns(false)
		clonedColumnNames := make([]string, len(columns))
		for i, c := range columns {
			clonedColumnNames[i] = strings.Clone(c.Name)
		}
		for i := range clonedColumnNames {
			switch clonedColumnNames[i] {
			case "trace_id":
				traceIDListLock.Lock()
				for _, v := range columns[i].Values {
					traceIDList = append(traceIDList, strings.Clone(v))
				}
				traceIDListLock.Unlock()
			case "_time":
				startTimeLock.Lock()
				for _, v := range columns[i].Values {
					if v < maxStartTimeStr {
						maxStartTimeStr = strings.Clone(v)
					}
				}
				startTimeLock.Unlock()
			}
		}
	}

	for currentStartTime.After(startTime) {
		qClone := q.CloneWithTimeFilter(currentTime.UnixNano(), currentStartTime.UnixNano(), endTime.UnixNano())
		qctx = qctx.WithQuery(qClone)
		if err := vtstorage.RunQuery(qctx, writeBlock); err != nil {
			if errors.Is(err, vtstoragecommon.ErrOutOfRetention) {
				return nil, time.Time{}, nil
			}
			return nil, time.Time{}, err
		}

		// found enough trace_id, return directly
		if len(traceIDList) == int(limit) {
			maxStartTime, err := time.Parse(time.RFC3339, maxStartTimeStr)
			if err != nil {
				return nil, maxStartTime, err
			}
			return checkTraceIDList(traceIDList), maxStartTime, nil
		}

		// not enough trace_id, clear the result, extend the time range and try again.
		traceIDList = traceIDList[:0]
		step *= 5
		currentStartTime = currentStartTime.Add(-step)
	}

	// one last try with input time range
	if currentStartTime.Before(startTime) {
		currentStartTime = startTime
	}

	qClone := q.CloneWithTimeFilter(currentTime.UnixNano(), currentStartTime.UnixNano(), endTime.UnixNano())
	qctx = qctx.WithQuery(qClone)
	if err := vtstorage.RunQuery(qctx, writeBlock); err != nil {
		return nil, time.Time{}, err
	}

	maxStartTime, err := time.Parse(time.RFC3339, maxStartTimeStr)
	if err != nil {
		return nil, maxStartTime, err
	}

	return checkTraceIDList(traceIDList), maxStartTime, nil
}

// findTraceIDTimeSplitTimeRange try to search from {trace_id_idx_stream="xx"} stream, which contains
// the trace_id and start/end time of this trace. It returns the time range of the trace if found.
//
// If the span with this trace_id never reach VictoriaTraces, the index search will go through the whole time range within
// the retention period, and returns an ErrOutOfRetention.
func findTraceIDTimeSplitTimeRange(ctx context.Context, q *logstorage.Query, cp *tracecommon.CommonParams, start, end time.Time) (time.Time, time.Time, error) {
	var (
		valueLock                              sync.Mutex
		traceIDStartTimeStr, traceIDEndTimeStr string
		// for compatible with old data
		timeStr string
	)

	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	cp.Query = q
	qctx := cp.NewQueryContext(ctxWithCancel)
	defer cp.UpdatePerQueryStatsMetrics()

	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		rowsCount := db.RowsCount()
		if rowsCount == 0 {
			return
		}

		if rowsCount > 1 {
			logger.Errorf("BUG: unexpected rowCount during trace ID index search. query: %s", q.String())
		}

		columns := db.GetColumns(false)
		clonedColumnNames := make([]string, len(columns))
		for i, c := range columns {
			clonedColumnNames[i] = strings.Clone(c.Name)
		}

		// There should be only a few lines in result, so it's safe to lock the whole block.
		valueLock.Lock()
		defer valueLock.Unlock()

		for _, c := range columns {
			switch c.Name {
			case "_time":
				timeStr = c.Values[len(c.Values)-1]
			case otelpb.TraceIDIndexStartTimeFieldName:
				for _, v := range c.Values {
					if traceIDStartTimeStr == "" || traceIDStartTimeStr > v {
						traceIDStartTimeStr = strings.Clone(v)
					}
				}
			case otelpb.TraceIDIndexEndTimeFieldName:
				for _, v := range c.Values {
					if traceIDEndTimeStr == "" || traceIDEndTimeStr < v {
						traceIDEndTimeStr = strings.Clone(v)
					}
				}
			}
		}
	}

	currentTime := time.Now()
	startTime := currentTime.Add(-*tracecommon.TraceSearchStep)
	if !start.IsZero() {
		startTime = start
	}
	endTime := currentTime
	if !end.IsZero() {
		endTime = end
	}

	for startTime.UnixNano() > 0 {
		qq := q.CloneWithTimeFilter(currentTime.UnixNano(), startTime.UnixNano(), endTime.UnixNano())
		qctx = qctx.WithQuery(qq)

		if err := vtstorage.RunQuery(qctx, writeBlock); err != nil {
			// this could be either a ErrOutOfRetention, or a real error.
			return time.Time{}, time.Time{}, err
		}

		// no hit in this time range
		if timeStr == "" {
			if start.IsZero() {
				// continue the loop if no time range
				endTime = startTime
				startTime = startTime.Add(-*tracecommon.TraceSearchStep)
				continue
			} else {
				// the time range is manually configured, stop here if result is empty
				return time.Time{}, time.Time{}, vtstoragecommon.ErrOutOfRetention
			}
		}

		// found result.
		if traceIDStartTimeStr == "" || traceIDEndTimeStr == "" {
			// this could be the old format index, which records trace ID and the approximate timestamp only.
			// to transform this into new format (start time & end time), use [t-traceWindow, t+traceWindow].
			// this code should be deprecated in the future.
			timestamp, _ := time.Parse(time.RFC3339, timeStr)
			return timestamp.Add(-*tracecommon.TraceMaxDurationWindow), timestamp.Add(*tracecommon.TraceMaxDurationWindow), nil
		}

		traceIDStartTime, _ := strconv.ParseInt(traceIDStartTimeStr, 10, 64)
		traceIDEndTime, _ := strconv.ParseInt(traceIDEndTimeStr, 10, 64)

		return time.Unix(traceIDStartTime/1e9, traceIDStartTime%1e9), time.Unix(traceIDEndTime/1e9, traceIDEndTime%1e9), nil
	}
	return time.Time{}, time.Time{}, vtstoragecommon.ErrOutOfRetention
}

// findSpansByTraceIDAndTime search for spans in given time range.
func findSpansByTraceIDAndTime(ctx context.Context, cp *tracecommon.CommonParams, traceID string, startTime, endTime time.Time) ([]*tracecommon.Row, error) {
	// query: trace_id:traceID
	qStr := fmt.Sprintf(otelpb.TraceIDField+": %q", traceID)
	q, err := logstorage.ParseQueryAtTimestamp(qStr, endTime.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}
	ctxWithCancel, cancel := context.WithCancel(ctx)
	cp.Query = q
	qctx := cp.NewQueryContext(ctxWithCancel)
	defer cp.UpdatePerQueryStatsMetrics()

	// search for trace spans and write to `rows []*Row`
	var rowsLock sync.Mutex
	var rows []*tracecommon.Row
	var missingTimeColumn atomic.Bool
	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		if missingTimeColumn.Load() {
			return
		}

		columns := db.GetColumns(false)
		clonedColumnNames := make([]string, len(columns))
		for i, c := range columns {
			clonedColumnNames[i] = strings.Clone(c.Name)
		}

		timestamps, ok := db.GetTimestamps(nil)
		if !ok {
			missingTimeColumn.Store(true)
			cancel()
			return
		}

		for i, timestamp := range timestamps {
			fields := make([]logstorage.Field, 0, len(columns))
			for j := range columns {
				// column could be empty if this span does not contain such field.
				// only append non-empty columns.
				if columns[j].Values[i] != "" {
					fields = append(fields, logstorage.Field{
						Name:  clonedColumnNames[j],
						Value: strings.Clone(columns[j].Values[i]),
					})
				}
			}

			rowsLock.Lock()
			rows = append(rows, &tracecommon.Row{
				Timestamp: timestamp,
				Fields:    fields,
			})
			rowsLock.Unlock()
		}
	}

	qq := q.CloneWithTimeFilter(endTime.UnixNano(), startTime.UnixNano(), endTime.UnixNano())
	qctx = qctx.WithQuery(qq)
	if err = vtstorage.RunQuery(qctx, writeBlock); err != nil {
		return nil, err
	}
	if missingTimeColumn.Load() {
		return nil, fmt.Errorf("missing _time column in the result for the query [%s]", qq)
	}
	return rows, nil
}

// checkTraceIDList removes invalid `trace_id`. It helps prevent query injection.
func checkTraceIDList(traceIDList []string) []string {
	result := make([]string, 0, len(traceIDList))
	for i := range traceIDList {
		if tracecommon.TraceIDRegex.MatchString(traceIDList[i]) {
			result = append(result, traceIDList[i])
		}
	}
	return result
}
