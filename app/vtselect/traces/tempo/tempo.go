package tempo

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/bytebufferpool"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/timeutil"
	"github.com/VictoriaMetrics/metrics"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/tracecommon"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/traceql"
)

var (
	tempoSearchTagsRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/tempo/api/v2/search/tags"}`)
	tempoSearchTagsDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/tempo/api/v2/search/tags"}`)

	tempoSearchTagValuesRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/tempo/api/v2/search/tag/*/values"}`)
	tempoSearchTagValuesDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/tempo/api/v2/search/tag/*/values"}`)

	tempoSearchRequests = metrics.NewCounter(`vt_http_requests_total{path="/select/tempo/api/search"}`)
	tempoSearchDuration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/tempo/api/search"}`)

	tempoQueryV2Requests = metrics.NewCounter(`vt_http_requests_total{path="/select/tempo/api/v2/traces/*"}`)
	tempoQueryV2Duration = metrics.NewSummary(`vt_http_request_duration_seconds{path="/select/tempo/api/v2/traces/*"}`)
)

var (
	defaultNoopFilter, _ = traceql.ParseQuery("{}")
)

func RequestHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	httpserver.EnableCORS(w, r)
	startTime := time.Now()
	path := r.URL.Path
	if path == "/select/tempo/api/echo" {
		// mainly for datasource creation health check.
		_, _ = w.Write([]byte("echo"))
		return true
	} else if path == "/select/tempo/api/v2/search/tags" {
		tempoSearchTagsRequests.Inc()
		processSearchTagsRequest(ctx, w, r)
		tempoSearchTagsDuration.UpdateDuration(startTime)
		return true
	} else if strings.HasPrefix(path, "/select/tempo/api/v2/search/tag/") && strings.HasSuffix(path, "/values") {
		tempoSearchTagValuesRequests.Inc()
		processSearchTagValuesRequest(ctx, w, r)
		tempoSearchTagValuesDuration.UpdateDuration(startTime)
		return true
	} else if path == "/select/tempo/api/search" {
		tempoSearchRequests.Inc()
		processSearchRequest(ctx, w, r)
		tempoSearchDuration.UpdateDuration(startTime)
		return true
	} else if strings.HasPrefix(path, "/select/tempo/api/v2/traces/") && len(path) > len("/select/tempo/api/v2/traces/") {
		tempoQueryV2Requests.Inc()
		processQueryV2Request(ctx, w, r)
		tempoQueryV2Duration.UpdateDuration(startTime)
		return true
	}
	return false
}

// processSearchTagsRequest handle the Tempo /api/v2/search/tags API request.
func processSearchTagsRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cp, err := tracecommon.GetCommonParams(r)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}

	params, err := parseTempoAPIParam(ctx, r, true)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}

	q := r.URL.Query()
	scope := q.Get("scope")

	result, err := searchTags(ctx, cp, params.q, scope, params.start.UnixNano(), params.end.UnixNano(), params.limit)
	if err != nil {
		httpserver.Errorf(w, r, "cannot get services list: %s", err)
		return
	}

	// Write results
	w.Header().Set("Content-Type", "application/json")
	WriteSearchTagsResponse(w, result.resourceTagList, result.spanTagList, result.eventTagList, result.linkTagList, result.instrumentationScopeTagList)
}

// processSearchTagValuesRequest handle the Tempo /api/v2/search/tag/*/values API request.
func processSearchTagValuesRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cp, err := tracecommon.GetCommonParams(r)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}

	// extract the `tag` name.
	// the path must be like `/select/tempo/api/v2/search/tag/<tag>/values`.
	u := r.URL.Path[len("/select/tempo/api/v2/search/tag/"):]

	// check for invalid path: /select/tempo/api/v2/search/tag/values
	if !strings.Contains(u, "/") {
		httpserver.Errorf(w, r, "incorrect query path [%s]", r.URL.Path)
		return
	}
	tagName := u[:len(u)-len("/values")]
	if len(tagName) == 0 {
		httpserver.Errorf(w, r, "incorrect query path [%s]", r.URL.Path)
		return
	}

	params, err := parseTempoAPIParam(ctx, r, true)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}

	// let's start from basic fields: service name, span name and status first.
	var mappedTagName string
	switch tagName {
	case "service.name", ".service.name": // it's not documented why a `.` could be the prefix. for compatible, add this special case for now.
		mappedTagName = otelpb.ResourceAttrServiceName
	case "status":
		mappedTagName = otelpb.StatusCodeField
	case "name", ".name":
		mappedTagName = otelpb.NameField
	default:
		if strings.HasPrefix(tagName, "resource.") {
			mappedTagName = otelpb.ResourceAttrPrefix + tagName[len("resource."):]
		} else if strings.HasPrefix(tagName, "span.") {
			mappedTagName = otelpb.SpanAttrPrefixField + tagName[len("span."):]
		} else if strings.HasPrefix(tagName, "event.") {
			mappedTagName = otelpb.EventPrefix + otelpb.EventAttrPrefix + tagName[len("event."):]
		} else {
			mappedTagName = tagName
		}
	}

	result, err := searchTagValues(ctx, cp, params.q, mappedTagName, params.start.UnixNano(), params.end.UnixNano(), params.limit)
	if err != nil {
		httpserver.Errorf(w, r, "cannot get tag values: %s", err)
		return
	}

	// Write results
	w.Header().Set("Content-Type", "application/json")
	WriteSearchTagValuesResponse(w, result)
}

// processSearchRequest handle the Tempo /api/v1/search API request.
func processSearchRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cp, err := tracecommon.GetCommonParams(r)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}

	params, err := parseTempoAPIParam(ctx, r, true)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}

	result, err := searchTraces(ctx, cp, params.q, params.start, params.end, params.limit)
	if err != nil {
		httpserver.Errorf(w, r, "cannot get traces list: %s", err)
		return
	}

	// Write results
	w.Header().Set("Content-Type", "application/json")
	WriteSearchResponse(w, result)
}

// processQueryV2Request handle the Tempo /api/v2/traces/<traceid> API request.
func processQueryV2Request(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cp, err := tracecommon.GetCommonParams(r)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}

	params, err := parseTempoAPIParam(ctx, r, false)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}

	// extract the `trace_id`.
	// the path must be like `/select/tempo/api/v2/traces/<trace_id>`.
	traceID := r.URL.Path[len("/select/tempo/api/v2/traces/"):]
	if len(traceID) == 0 {
		httpserver.Errorf(w, r, "incorrect query path [%s]", r.URL.Path)
		return
	}

	rows, err := GetTrace(ctx, cp, traceID, params.start, params.end)
	if err != nil {
		httpserver.Errorf(w, r, "cannot get traces list: %s", err)
		return
	}

	resourceSpans, err := rowsToResourceSpans(rows)
	if err != nil {
		httpserver.Errorf(w, r, "cannot parse rows into resource spans: %s", err)
		return
	}

	resp := otelpb.TempoTraceByIDResponse{
		Trace: otelpb.TempoTrace{
			ResourceSpan: resourceSpans,
		},
	}

	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)

	b.B = resp.MarshalProtobuf(b.B)

	// Write results
	w.Header().Set("Content-Type", "application/protobuf")
	w.Header().Set("Content-Length", strconv.Itoa(b.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b.Bytes())
}

type searchTagResult struct {
	resourceTagList, spanTagList, eventTagList, linkTagList, instrumentationScopeTagList []string
}

func searchTags(ctx context.Context, cp *tracecommon.CommonParams, traceQLStr string, scope string, start, end, limit int64) (*searchTagResult, error) {
	// transform traceQL into LogsQL as filter. It should contain filter only without any pipe.
	filterQuery, err := traceql.ParseQuery(traceQLStr)
	if err != nil {
		filterQuery = defaultNoopFilter
	}

	// exclude queries that contains pipe(s).
	if filterQuery.HasPipe() {
		return nil, fmt.Errorf("cannot use query pipes in search tag values API: %s", traceQLStr)
	}

	scopes := ``
	pipeLimit := limit
	switch scope {
	case "instrumentation":
		scopes = fmt.Sprintf(`| filter name:"%s:"*`, otelpb.InstrumentationScopeAttrPrefix)
	case "resource":
		scopes = fmt.Sprintf(`| filter name:"%s:"*`, otelpb.ResourceAttrPrefix)
	case "span":
		scopes = fmt.Sprintf(`| filter name:"%s:"*`, otelpb.SpanAttrPrefixField)
	case "event":
		//scopes = fmt.Sprintf(`| filter name:"%s:"*`, otelpb.EventPrefix+otelpb.EventAttrPrefix)
		return nil, errors.New("scope: event is not supported yet")
	case "link":
		return nil, errors.New("scope: link is not supported yet")
		//scopes = fmt.Sprintf(`| filter name:"%s:"*`, otelpb.LinkPrefix+otelpb.LinkAttrPrefix)
	case "intrinsic":
		return nil, errors.New("scope: intrinsic is not supported yet")
	case "", "all":
		// todo: this does not align with the doc, but user usually don't expect a result fully match the limit
		// because they're not really looking for a specific tag name when no scope argument is used. it's likely
		// to be an initial request when loading possible option on the search page.
		// so returning "something" should be enough.
		pipeLimit = limit * 3
	default:
		return nil, fmt.Errorf("unsupported scope: %s", scope)
	}

	qStr := fmt.Sprintf(`%s | field_names %s | uniq by (name)`,
		filterQuery.String(), scopes,
	)

	q, err := logstorage.ParseQueryAtTimestamp(qStr, time.Now().UnixNano())
	if err != nil {
		return nil, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}
	q.AddTimeFilter(start, end)
	q.AddPipeOffsetLimit(0, uint64(pipeLimit))

	fieldNames, err := singleFieldQueryHelper(ctx, q, cp, pipeLimit)
	if err != nil {
		return nil, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}

	result := &searchTagResult{
		resourceTagList:             []string{},
		spanTagList:                 []string{},
		instrumentationScopeTagList: []string{},
		eventTagList:                []string{},
		linkTagList:                 []string{},
	}
	for i := range fieldNames {
		if strings.HasPrefix(fieldNames[i], otelpb.SpanAttrPrefixField) {
			result.spanTagList = appendNoExceedN(result.spanTagList, fieldNames[i][len(otelpb.SpanAttrPrefixField):], limit)
		} else if strings.HasPrefix(fieldNames[i], otelpb.ResourceAttrPrefix) {
			result.resourceTagList = appendNoExceedN(result.resourceTagList, fieldNames[i][len(otelpb.ResourceAttrPrefix):], limit)
		} else if strings.HasPrefix(fieldNames[i], otelpb.InstrumentationScopeAttrPrefix) {
			result.instrumentationScopeTagList = appendNoExceedN(result.instrumentationScopeTagList, fieldNames[i][len(otelpb.InstrumentationScopeAttrPrefix):], limit)
		} else {
			// strings.HasPrefix(fieldNames[i], otelpb.LinkPrefix+otelpb.LinkAttrPrefix) || strings.HasPrefix(fieldNames[i], otelpb.EventPrefix+otelpb.EventAttrPrefix)
			//lIdx := strings.LastIndex(fieldNames[i], ":")
			//result.linkTagList = appendNoExceedN(result.linkTagList, fieldNames[i][len(otelpb.LinkPrefix+otelpb.LinkAttrPrefix):lIdx], limit)

			//lIdx := strings.LastIndex(fieldNames[i], ":")
			//result.eventTagList = appendNoExceedN(result.eventTagList, fieldNames[i][len(otelpb.EventPrefix+otelpb.EventAttrPrefix):lIdx], limit)
			// todo wait until LogsQL support search across fields.
			continue

		}
	}
	return result, nil
}

func appendNoExceedN(s []string, item string, n int64) []string {
	if len(s) >= int(n) {
		return s
	}
	return append(s, item)
}

func searchTagValues(ctx context.Context, cp *tracecommon.CommonParams, traceQLStr, tagName string, start, end, limit int64) ([]string, error) {
	// transform traceQL into LogsQL as filter. It should contain filter only without any pipe.
	filterQuery, err := traceql.ParseQuery(traceQLStr)
	if err != nil {
		filterQuery = defaultNoopFilter
	}

	// exclude queries that contains pipe(s).
	if filterQuery.HasPipe() {
		return nil, fmt.Errorf("cannot use query pipes in search tag values API: %s", traceQLStr)
	}

	qStr := fmt.Sprintf(`%s | fields %q | field_values %q | fields %q`,
		filterQuery.String(), tagName, tagName, tagName,
	)

	q, err := logstorage.ParseQueryAtTimestamp(qStr, time.Now().UnixNano())
	if err != nil {
		return nil, fmt.Errorf("cannot parse query [%s]: %s", qStr, err)
	}
	q.AddTimeFilter(start, end)
	q.AddPipeOffsetLimit(0, uint64(limit))

	return singleFieldQueryHelper(ctx, q, cp, limit)
}

// singleFieldQueryHelper execute queries which contains only a single field in response, and return as []string.
// it's useful for queries looking for `field_name`s or `field_value`s.
func singleFieldQueryHelper(ctx context.Context, q *logstorage.Query, cp *tracecommon.CommonParams, limit int64) ([]string, error) {
	resultList := make([]string, 0, limit)
	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		columns := db.Columns
		if len(columns) != 1 {
			logger.Panicf("BUG: unexpected column(s) returned for singleFieldQueryHelper: %v", columns)
		}

		for _, v := range columns[0].Values {
			if v != "" {
				resultList = append(resultList, strings.Clone(v))
			}
		}
	}

	cp.Query = q
	qctx := cp.NewQueryContext(ctx)
	defer cp.UpdatePerQueryStatsMetrics()

	if err := vtstorage.RunQuery(qctx, writeBlock); err != nil {
		return nil, err
	}

	return resultList, nil
}

func searchTraces(ctx context.Context, cp *tracecommon.CommonParams, traceQLStr string, start, end time.Time, limit int64) ([]traceSummary, error) {
	// transform traceQL into LogsQL as filter. It should contain filter only without any pipe.
	filterQuery, err := traceql.ParseQuery(traceQLStr)
	if err != nil {
		return nil, err
	}

	// exclude queries that contains pipe(s).
	if filterQuery.HasPipe() {
		return nil, fmt.Errorf("cannot use query pipes in search tag values API: %s", traceQLStr)
	}

	_, rows, err := GetTraceList(ctx, cp, filterQuery, start, end, limit)
	if err != nil {
		return nil, err
	}

	result, err := summarySearchTracesResult(ctx, rows, limit)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type traceSummary struct {
	traceID           string
	rootServiceName   string
	rootTraceName     string
	startTimeUnixNano int64
	endTimeUnixNano   int64
}

func summarySearchTracesResult(ctx context.Context, rows []*tracecommon.Row, limit int64) ([]traceSummary, error) {
	traceMap := make(map[string]traceSummary)

	for _, row := range rows {
		var traceID, serviceName, spanName, parentSpanID string
		var startTimeUnixNano, endTimeUnixNano int64
		var err error
		for _, field := range row.Fields {
			switch field.Name {
			case otelpb.ResourceAttrServiceName:
				serviceName = field.Value
			case otelpb.TraceIDField:
				traceID = field.Value
			case otelpb.NameField:
				spanName = field.Value
			case otelpb.ParentSpanIDField:
				parentSpanID = field.Value
			case otelpb.StartTimeUnixNanoField:
				startTimeUnixNano, err = strconv.ParseInt(field.Value, 10, 64)
				if err != nil {
					return nil, err
				}
			case otelpb.EndTimeUnixNanoField:
				endTimeUnixNano, err = strconv.ParseInt(field.Value, 10, 64)
				if err != nil {
					return nil, err
				}
			default:
				continue
			}
		}

		if traceID == "" {
			return nil, fmt.Errorf("trace ID found for a span %v", row)
		}

		// get the summary for this trace
		summary, ok := traceMap[traceID]
		if !ok {
			summary = traceSummary{
				startTimeUnixNano: math.MaxInt64,
				rootServiceName:   "<root span not yet received>",
			}
			traceMap[traceID] = summary
		}

		summary.traceID = traceID
		summary.startTimeUnixNano = min(summary.startTimeUnixNano, startTimeUnixNano)
		summary.endTimeUnixNano = max(summary.endTimeUnixNano, endTimeUnixNano)
		// if it's the root span
		if parentSpanID == "" {
			summary.rootServiceName = serviceName
			summary.rootTraceName = spanName
		}
		// summary is not a pointer so it must be put back to the map.
		traceMap[traceID] = summary
	}

	resultList := make([]traceSummary, 0, len(traceMap))
	for _, summary := range traceMap {
		resultList = append(resultList, summary)
	}
	return resultList, nil
}

type commonAPIParam struct {
	q     string
	start time.Time
	end   time.Time
	limit int64
}

// parseTempoAPIParam parse Tempo request.
func parseTempoAPIParam(_ context.Context, r *http.Request, allowDefaultTime bool) (*commonAPIParam, error) {
	// default params
	p := &commonAPIParam{
		q:     "{}",
		start: time.Time{},
		end:   time.Time{},
		limit: 100,
	}

	if allowDefaultTime {
		p.start = time.Now().Add(-10 * time.Minute)
		p.end = time.Now()
	}

	q := r.URL.Query()

	start := q.Get("start")
	if start != "" {
		ts, ok := timeutil.TryParseUnixTimestamp(start)
		if !ok {
			return nil, fmt.Errorf("cannot parse start timestamp: %s", start)
		}
		p.start = time.Unix(ts/1e9, 0)
	}
	end := q.Get("end")
	if end != "" {
		ts, ok := timeutil.TryParseUnixTimestamp(end)
		if !ok {
			return nil, fmt.Errorf("cannot parse end timestamp: %s", start)
		}
		p.end = time.Unix(ts/1e9, 0)
	}
	if p.start.After(p.end) {
		p.start = p.end.Add(-10 * time.Minute)
	}

	limit := q.Get("limit")
	if limit != "" {
		l, err := strconv.ParseInt(limit, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot parse limit: %s", limit)
		}
		// Let's limit this to [0, 1000] to prevent users from specifying an excessively large value.
		p.limit = max(0, min(1000, l))
	}

	p.q = q.Get("q")

	return p, nil
}
