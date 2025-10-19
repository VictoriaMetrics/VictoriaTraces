package query

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/cespare/xxhash/v2"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage"
	vtstoragecommon "github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage/common"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

var (
	traceMaxDurationWindow = flag.Duration("search.traceMaxDurationWindow", 45*time.Second, "The window of searching for the rest trace spans after finding one span."+
		"It allows extending the search start time and end time by -search.traceMaxDurationWindow to make sure all spans are included."+
		"It affects both Jaeger's /api/traces and /api/traces/<trace_id> APIs.")
	traceServiceAndSpanNameLookbehind = flag.Duration("search.traceServiceAndSpanNameLookbehind", 3*24*time.Hour, "The time range of searching for service name and span name. "+
		"It affects Jaeger's /api/services and /api/services/*/operations APIs.")
	traceSearchStep = flag.Duration("search.traceSearchStep", 24*time.Hour, "Splits the [0, now] time range into many small time ranges by -search.traceSearchStep "+
		"when searching for spans by trace_id. Once it finds spans in a time range, it performs an additional search according to -search.traceMaxDurationWindow and then stops. "+
		"It affects Jaeger's /api/traces/<trace_id> API.")
	traceMaxServiceNameList = flag.Uint64("search.traceMaxServiceNameList", 1000, "The maximum number of service name can return in a get service name request. "+
		"This limit affects Jaeger's /api/services API.")
	traceMaxSpanNameList = flag.Uint64("search.traceMaxSpanNameList", 1000, "The maximum number of span name can return in a get span name request. "+
		"This limit affects Jaeger's /api/services/*/operations API.")
)

var (
	traceIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-.:]*$`)
)

// CommonParams common query params that shared by all requests.
type CommonParams struct {
	TenantIDs []logstorage.TenantID
	Query     *logstorage.Query

	// qs contains execution statistics for the Query.
	qs logstorage.QueryStats
}

func (cp *CommonParams) NewQueryContext(ctx context.Context) *logstorage.QueryContext {
	return logstorage.NewQueryContext(ctx, &cp.qs, cp.TenantIDs, cp.Query, false)
}

func (cp *CommonParams) UpdatePerQueryStatsMetrics() {
	vtstorage.UpdatePerQueryStatsMetrics(&cp.qs)
}

// GetCommonParams get common params from request for all traces query APIs.
func GetCommonParams(r *http.Request) (*CommonParams, error) {
	tenantID, err := logstorage.GetTenantIDFromRequest(r)
	if err != nil {
		return nil, fmt.Errorf("cannot obtain tenantID: %w", err)
	}
	tenantIDs := []logstorage.TenantID{tenantID}
	cp := &CommonParams{
		TenantIDs: tenantIDs,
	}
	return cp, nil
}

// TraceQueryParam is the parameters for querying a batch of traces.
type TraceQueryParam struct {
	ServiceName  string
	SpanName     string
	Attributes   map[string]string
	StartTimeMin time.Time
	StartTimeMax time.Time
	DurationMin  time.Duration
	DurationMax  time.Duration
	Limit        int
}

// Row represent the query result of a trace span.
type Row struct {
	Timestamp int64
	Fields    []logstorage.Field
}

// GetServiceNameList returns all unique service names within *traceServiceAndSpanNameLookbehind window.
// todo: cache of recent result.
func GetServiceNameList(ctx context.Context, cp *CommonParams) ([]string, error) {
	currentTime := time.Now()

	// query: _time:[start, end] *
	qStr := "*"
	q, err := logstorage.ParseQueryAtTimestamp(qStr, currentTime.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}
	q.AddTimeFilter(currentTime.Add(-*traceServiceAndSpanNameLookbehind).UnixNano(), currentTime.UnixNano())

	cp.Query = q
	qctx := cp.NewQueryContext(ctx)
	defer cp.UpdatePerQueryStatsMetrics()

	serviceHits, err := vtstorage.GetStreamFieldValues(qctx, otelpb.ResourceAttrServiceName, *traceMaxServiceNameList)
	if err != nil {
		return nil, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}

	serviceList := make([]string, 0, len(serviceHits))
	for i := range serviceHits {
		serviceList = append(serviceList, serviceHits[i].Value)
	}
	return serviceList, nil
}

// GetSpanNameList returns all unique span names for a service within *traceServiceAndSpanNameLookbehind window.
// todo: cache of recent result.
func GetSpanNameList(ctx context.Context, cp *CommonParams, serviceName string) ([]string, error) {
	currentTime := time.Now()

	// query: _time:[start, end] {"resource_attr:service.name"=serviceName}
	qStr := fmt.Sprintf("_stream:{%s=%q}", otelpb.ResourceAttrServiceName, serviceName)
	q, err := logstorage.ParseQueryAtTimestamp(qStr, currentTime.Unix())
	if err != nil {
		return nil, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}
	q.AddTimeFilter(currentTime.Add(-*traceServiceAndSpanNameLookbehind).UnixNano(), currentTime.UnixNano())

	cp.Query = q
	qctx := cp.NewQueryContext(ctx)
	defer cp.UpdatePerQueryStatsMetrics()

	spanNameHits, err := vtstorage.GetStreamFieldValues(qctx, otelpb.NameField, *traceMaxSpanNameList)
	if err != nil {
		return nil, fmt.Errorf("get span name hits error: %s", err)
	}

	spanNameList := make([]string, 0, len(spanNameHits))
	for i := range spanNameHits {
		spanNameList = append(spanNameList, spanNameHits[i].Value)
	}
	return spanNameList, nil
}

// GetTrace returns all spans of a trace in []*Row format.
// It search in the index stream for the approximate timestamp.
// If found:
// - search for span in time range [aTimestamp-traceMaxDurationWindow, aTimestamp+traceMaxDurationWindow].
// If not found:
// - search span by step via findSpansByTraceID.
//
// todo in-memory cache of hot traces.
func GetTrace(ctx context.Context, cp *CommonParams, traceID string) ([]*Row, error) {
	currentTime := time.Now()

	// possible partition
	// query: {trace_id_idx="xx"} AND trace_id:traceID
	qStr := fmt.Sprintf(`{%s="%d"} AND %s:=%q | fields _time`, otelpb.TraceIDIndexStreamName, xxhash.Sum64String(traceID)%otelpb.TraceIDIndexPartitionCount, otelpb.TraceIDIndexFieldName, traceID)
	q, err := logstorage.ParseQueryAtTimestamp(qStr, currentTime.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal query=%q: %w", qStr, err)
	}
	q.AddPipeOffsetLimit(0, 1)
	traceTimestamp, err := findTraceIDTimeSplitTimeRange(ctx, q, cp)
	if err != nil && errors.Is(err, vtstoragecommon.ErrOutOfRetention) {
		// no hit in the retention period, simply returns empty.
		return nil, nil
	}
	if err != nil {
		// something wrong when trying to find the trace_id's start time.
		return nil, fmt.Errorf("cannot find trace_id %q start time: %s", traceID, err)
	}

	// trace start time found, search in [trace start time, trace start time + *traceMaxDurationWindow] time range.
	return findSpansByTraceIDAndTime(ctx, cp, traceID, traceTimestamp.Add(-*traceMaxDurationWindow), traceTimestamp.Add(*traceMaxDurationWindow))
}

// GetTraceList returns multiple traceIDs and spans of them in []*Row format.
// It search for traceIDs first, and then search for the spans of these traceIDs.
// To not miss any spans on the edge, it extends both the start time and end time
// by *traceMaxDurationWindow.
//
// e.g.:
// 1. input time range: [00:00, 09:00]
// 2. found 20 trace id, and adjust time range to: [08:00, 09:00]
// 3. find spans on time range: [08:00-traceMaxDurationWindow, 09:00+traceMaxDurationWindow]
func GetTraceList(ctx context.Context, cp *CommonParams, param *TraceQueryParam) ([]string, []*Row, error) {
	currentTime := time.Now()

	// query 1: * AND filter_conditions | last 1 by (_time) partition by (trace_id) | fields _time, trace_id | sort by (_time) desc
	traceIDs, startTime, err := getTraceIDList(ctx, cp, param)
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
	q.AddTimeFilter(startTime.Add(-*traceMaxDurationWindow).UnixNano(), param.StartTimeMax.Add(*traceMaxDurationWindow).UnixNano())

	ctxWithCancel, cancel := context.WithCancel(ctx)
	cp.Query = q
	qctx := cp.NewQueryContext(ctxWithCancel)
	defer cp.UpdatePerQueryStatsMetrics()

	// search for trace spans and write to `rows []*Row`
	var rowsLock sync.Mutex
	var rows []*Row
	var missingTimeColumn atomic.Bool
	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		if missingTimeColumn.Load() {
			return
		}

		columns := db.Columns
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
			rows = append(rows, &Row{
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

// getTraceIDList returns traceIDs according to the search params.
// It also returns the earliest start time of these traces, to help reducing the time range for spans search.
func getTraceIDList(ctx context.Context, cp *CommonParams, param *TraceQueryParam) ([]string, time.Time, error) {
	currentTime := time.Now()
	// query: * AND <filter> | last 1 by (_time) partition by (trace_id) | fields _time, trace_id | sort by (_time) desc
	qStr := "* "
	if param.ServiceName != "" {
		qStr += fmt.Sprintf("AND _stream:{"+otelpb.ResourceAttrServiceName+"=%q} ", param.ServiceName)
	}
	if param.SpanName != "" {
		qStr += fmt.Sprintf("AND _stream:{"+otelpb.NameField+"=%q} ", param.SpanName)
	}
	if len(param.Attributes) > 0 {
		for k, v := range param.Attributes {
			qStr += fmt.Sprintf(`AND %q:=%q `, k, v)
		}
	}
	if param.DurationMin > 0 {
		qStr += fmt.Sprintf("AND "+otelpb.DurationField+":>%d ", param.DurationMin.Nanoseconds())
	}
	if param.DurationMax > 0 {
		qStr += fmt.Sprintf("AND duration:<%d ", param.DurationMax.Nanoseconds())
	}
	qStr += " | last 1 by (_time) partition by (" + otelpb.TraceIDField + ") | fields _time, " + otelpb.TraceIDField + " | sort by (_time) desc"

	q, err := logstorage.ParseQueryAtTimestamp(qStr, currentTime.UnixNano())
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}
	q.AddPipeOffsetLimit(0, uint64(param.Limit))

	traceIDs, maxStartTime, err := findTraceIDsSplitTimeRange(ctx, q, cp, param.StartTimeMin, param.StartTimeMax, param.Limit)
	if err != nil {
		return nil, time.Time{}, err
	}

	return traceIDs, maxStartTime, nil
}

// findTraceIDsSplitTimeRange try to search from the nearest time range of the end time.
// if the result already met requirement of `limit`, return.
// otherwise, amplify the time range to 5x and search again, until the start time exceed the input.
func findTraceIDsSplitTimeRange(ctx context.Context, q *logstorage.Query, cp *CommonParams, startTime, endTime time.Time, limit int) ([]string, time.Time, error) {
	currentTime := time.Now()

	step := time.Minute
	currentStartTime := endTime.Add(-step)

	var traceIDListLock sync.Mutex
	traceIDList := make([]string, 0, limit)
	maxStartTimeStr := endTime.Format(time.RFC3339)

	cp.Query = q
	qctx := cp.NewQueryContext(ctx)
	defer cp.UpdatePerQueryStatsMetrics()

	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		columns := db.Columns
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
				for _, v := range columns[i].Values {
					if v < maxStartTimeStr {
						maxStartTimeStr = strings.Clone(v)
					}
				}
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
		if len(traceIDList) == limit {
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
// the trace_id and the rough start time of this trace. It returns the start time of the trace if found.
//
// If the span with this trace_id never reach VictoriaTraces, the search will to through the whole time range within
// the retention period, and returns an ErrOutOfRetention.
func findTraceIDTimeSplitTimeRange(ctx context.Context, q *logstorage.Query, cp *CommonParams) (time.Time, error) {
	traceIDStartTimeInt := int64(0)
	var missingTimeColumn atomic.Bool

	ctxWithCancel, cancel := context.WithCancel(ctx)
	cp.Query = q
	qctx := cp.NewQueryContext(ctxWithCancel)
	defer cp.UpdatePerQueryStatsMetrics()

	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		if missingTimeColumn.Load() {
			return
		}

		columns := db.Columns
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
		if len(timestamps) > 0 {
			traceIDStartTimeInt = timestamps[0]
		}
	}

	currentTime := time.Now()
	startTime := currentTime.Add(-*traceSearchStep)
	endTime := currentTime
	for startTime.UnixNano() > 0 {
		qq := q.CloneWithTimeFilter(currentTime.UnixNano(), startTime.UnixNano(), endTime.UnixNano())
		qctx = qctx.WithQuery(qq)

		if err := vtstorage.RunQuery(qctx, writeBlock); err != nil {
			// this could be either a ErrOutOfRetention, or a real error.
			return time.Time{}, err
		}

		if missingTimeColumn.Load() {
			return time.Time{}, fmt.Errorf("missing _time column in the result for the query [%s]", qq)
		}

		// no hit in this time range, continue with step.
		if traceIDStartTimeInt == 0 {
			endTime = startTime
			startTime = startTime.Add(-*traceSearchStep)
			continue
		}

		// found result, perform extra search for traceMaxDurationWindow and then break.
		return time.Unix(traceIDStartTimeInt/1e9, traceIDStartTimeInt%1e9), nil
	}
	return time.Time{}, vtstoragecommon.ErrOutOfRetention
}

// findSpansByTraceIDAndTime search for spans in given time range.
func findSpansByTraceIDAndTime(ctx context.Context, cp *CommonParams, traceID string, startTime, endTime time.Time) ([]*Row, error) {
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
	var rows []*Row
	var missingTimeColumn atomic.Bool
	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		if missingTimeColumn.Load() {
			return
		}

		columns := db.Columns
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
			rows = append(rows, &Row{
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
		if traceIDRegex.MatchString(traceIDList[i]) {
			result = append(result, traceIDList[i])
		}
	}
	return result
}

type ServiceGraphQueryParameters struct {
	EndTs    time.Time
	Lookback time.Duration
}

// GetServiceGraphList returns service dependencies graph edges (parent, child, callCount) in []*Row format.
//
// TODO: currently this function can only handle request from Jaeger dependencies API. Since Tempo provides similar service graph
// feature, it would be great to add support for Tempo service graph API as well.
func GetServiceGraphList(ctx context.Context, cp *CommonParams, param *ServiceGraphQueryParameters) ([]*Row, error) {
	// {trace_service_graph_stream="-"} | fields parent, child, callCount | stats by (parent, child) sum(callCount) as callCount
	qStr := fmt.Sprintf(`{%s="-"} | fields %s, %s, %s | stats by (%s, %s) sum(%s) as %s`,
		otelpb.ServiceGraphStreamName,
		otelpb.ServiceGraphParentFieldName,
		otelpb.ServiceGraphChildFieldName,
		otelpb.ServiceGraphCallCountFieldName,
		otelpb.ServiceGraphParentFieldName,
		otelpb.ServiceGraphChildFieldName,
		otelpb.ServiceGraphCallCountFieldName,
		otelpb.ServiceGraphCallCountFieldName,
	)
	startTime := param.EndTs.Add(-param.Lookback).UnixNano()
	endTime := param.EndTs.UnixNano()
	q, err := logstorage.ParseQueryAtTimestamp(qStr, endTime)
	if err != nil {
		return nil, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}
	q.AddTimeFilter(startTime, endTime)

	cp.Query = q
	qctx := cp.NewQueryContext(ctx)

	var rowsLock sync.Mutex
	var rows []*Row
	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		columns := db.Columns
		if len(columns) == 0 {
			return
		}
		clonedColumnNames := make([]string, len(columns))
		valuesCount := 0
		for i, c := range columns {
			clonedColumnNames[i] = strings.Clone(c.Name)
			if len(c.Values) > valuesCount {
				valuesCount = len(c.Values)
			}
		}
		if valuesCount == 0 {
			return
		}
		for i := 0; i < valuesCount; i++ {
			fields := make([]logstorage.Field, 0, len(columns))
			for j := range columns {
				fields = append(
					fields,
					logstorage.Field{
						Name:  clonedColumnNames[j],
						Value: strings.Clone(columns[j].Values[i]),
					},
				)
			}
			rowsLock.Lock()
			rows = append(rows, &Row{
				Fields: fields,
			})
			rowsLock.Unlock()
		}
	}

	if err = vtstorage.RunQuery(qctx, writeBlock); err != nil {
		return nil, err
	}

	return rows, nil
}

// GetServiceGraphTimeRange is an internal function used by service graph background task.
// It calculates the service graph relation within the time range in (parent, child, callCount) format for specific tenant.
func GetServiceGraphTimeRange(ctx context.Context, tenantID logstorage.TenantID, startTime, endTime time.Time, limit uint64) ([][]logstorage.Field, error) {
	cp := &CommonParams{
		TenantIDs: []logstorage.TenantID{tenantID},
	}

	// (NOT parent_span_id:"") AND (kind:~"2|5")  | fields parent_span_id, resource_attr:service.name | rename parent_span_id as span_id, resource_attr:service.name as child
	qStrChildSpans := fmt.Sprintf(
		`(NOT %s:"") AND (%s:~"%d|%d")  | fields %s, %s | rename %s as %s, %s as %s`,
		otelpb.ParentSpanIDField, // parent span id not empty means this span is a child span.
		otelpb.KindField,         // only server(2) and consumer(5) span could be used as a child. It helps reduce the spans it needs to fetch.
		otelpb.SpanKind(2),
		otelpb.SpanKind(5),
		otelpb.ParentSpanIDField,
		otelpb.ResourceAttrServiceName,
		otelpb.ParentSpanIDField,
		otelpb.SpanIDField,
		otelpb.ResourceAttrServiceName,
		otelpb.ServiceGraphChildFieldName,
	)
	// (NOT span_id:"") AND (kind:~"3|4")  | fields span_id, resource_attr:service.name | rename resource_attr:service.name as parent
	qStrParentSpans := fmt.Sprintf(
		`(NOT %s:"") AND (%s:~"%d|%d") | fields %s, %s | rename %s as %s`,
		otelpb.SpanIDField, // Any span could be a parent span, as long as it has a span ID.
		otelpb.KindField,   // only client(3) and producer(4) span could be used as a parent. It helps reduce the spans it needs to fetch.
		otelpb.SpanKind(3),
		otelpb.SpanKind(4),
		otelpb.SpanIDField,
		otelpb.ResourceAttrServiceName,
		otelpb.ResourceAttrServiceName,
		otelpb.ServiceGraphParentFieldName,
	)
	// join by span_id
	qStr := fmt.Sprintf(
		`%s | join by (%s) (%s) inner | NOT %s:eq_field(%s) | stats by (%s, %s) count() %s`,
		qStrChildSpans,
		otelpb.SpanIDField,
		qStrParentSpans,
		otelpb.ServiceGraphParentFieldName,
		otelpb.ServiceGraphChildFieldName,
		otelpb.ServiceGraphParentFieldName,
		otelpb.ServiceGraphChildFieldName,
		otelpb.ServiceGraphCallCountFieldName,
	)

	q, err := logstorage.ParseQueryAtTimestamp(qStr, endTime.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}
	q.AddTimeFilter(startTime.UnixNano(), endTime.UnixNano())
	q.AddPipeOffsetLimit(0, limit)

	cp.Query = q
	qctx := cp.NewQueryContext(ctx)
	defer cp.UpdatePerQueryStatsMetrics()

	var rowsLock sync.Mutex
	var rows [][]logstorage.Field
	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		columns := db.Columns
		if len(columns) == 0 {
			return
		}
		clonedColumnNames := make([]string, len(columns))
		valuesCount := 0
		for i, c := range columns {
			clonedColumnNames[i] = strings.Clone(c.Name)
			if len(c.Values) > valuesCount {
				valuesCount = len(c.Values)
			}
		}
		if valuesCount == 0 {
			return
		}
		for i := 0; i < valuesCount; i++ {
			fields := make([]logstorage.Field, 0, len(columns))
			for j := range clonedColumnNames {
				fields = append(
					fields,
					logstorage.Field{
						Name:  clonedColumnNames[j],
						Value: strings.Clone(columns[j].Values[i]),
					},
				)
			}
			rowsLock.Lock()
			rows = append(rows, fields)
			rowsLock.Unlock()
		}
	}

	if err = vtstorage.RunQuery(qctx, writeBlock); err != nil {
		return nil, fmt.Errorf("cannot execute query [%s]: %s", qStr, err)
	}

	return rows, nil
}
