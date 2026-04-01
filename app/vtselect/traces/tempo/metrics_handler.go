package tempo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/timeutil"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/tracecommon"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/traceql"
)

type metricsQueryRangeParam struct {
	q     string
	start time.Time
	end   time.Time
	step  int64 // nanoseconds
}

// processMetricsQueryRangeRequest handles the Tempo /api/metrics/query_range API request.
func processMetricsQueryRangeRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cp, err := tracecommon.GetCommonParams(r)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}

	params, err := parseMetricsQueryRangeParams(r)
	if err != nil {
		httpserver.Errorf(w, r, "incorrect query params: %s", err)
		return
	}

	translation, err := translateMetricsQuery(params.q, params.end.UnixNano())
	if err != nil {
		httpserver.Errorf(w, r, "cannot translate metrics query: %s", err)
		return
	}

	var allSeries []tempoMetricsSeries

	if translation.isCompare {
		allSeries, err = executeCompareQuery(ctx, cp, translation, params)
		if err != nil {
			httpserver.Errorf(w, r, "cannot execute compare query: %s", err)
			return
		}
	} else {
		allSeries, err = executeStatsQuery(ctx, cp, translation.baseQuery, translation.byFields, params)
		if err != nil {
			httpserver.Errorf(w, r, "cannot execute query: %s", err)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	WriteMetricsQueryRangeResponse(w, allSeries)
}

// Fields to exclude from compare attribute discovery.
var compareExcludedFields = map[string]bool{
	otelpb.TraceIDField:                true,
	otelpb.SpanIDField:                 true,
	otelpb.ParentSpanIDField:           true,
	otelpb.StartTimeUnixNanoField:      true,
	otelpb.EndTimeUnixNanoField:        true,
	otelpb.FlagsField:                  true,
	otelpb.TraceStateField:             true,
	otelpb.DroppedAttributesCountField: true,
	otelpb.DroppedEventsCountField:     true,
	otelpb.DroppedLinksCountField:      true,
	otelpb.DurationField: true,
	// internal index fields
	otelpb.TraceIDIndexStreamName:         true,
	otelpb.TraceIDIndexFieldName:          true,
	otelpb.TraceIDIndexStartTimeFieldName: true,
	otelpb.TraceIDIndexEndTimeFieldName:   true,
	// service graph fields
	otelpb.ServiceGraphStreamName:         true,
	otelpb.ServiceGraphParentFieldName:    true,
	otelpb.ServiceGraphChildFieldName:     true,
	otelpb.ServiceGraphCallCountFieldName: true,
	// VictoriaLogs internals
	"_msg": true, "_time": true, "_stream": true, "_stream_id": true,
}

// compareAttrResult holds per-attribute counts for baseline and selection.
type compareAttrResult struct {
	attrName  string                       // VT field name
	baseline  map[string]map[int64]float64 // value → timestamp → count
	selection map[string]map[int64]float64
}

// executeCompareQuery discovers attributes and runs per-attribute count queries for compare().
func executeCompareQuery(ctx context.Context, cp *tracecommon.CommonParams, t *metricsQueryTranslation, params *metricsQueryRangeParam) ([]tempoMetricsSeries, error) {
	// Step 1: Discover attributes via GetFieldNames (it handles discovery internally).
	q, err := logstorage.ParseQueryAtTimestamp(t.baseFilter, params.end.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("cannot parse field_names query: %w", err)
	}
	q.AddTimeFilter(params.start.UnixNano(), params.end.UnixNano())

	cpDiscover := *cp
	cpDiscover.Query = q
	qctx := cpDiscover.NewQueryContext(ctx)
	fieldNames, err := vtstorage.GetFieldNames(qctx)
	cpDiscover.UpdatePerQueryStatsMetrics()
	if err != nil {
		return nil, fmt.Errorf("cannot discover field names: %w", err)
	}

	// Step 2: Filter excluded fields.
	var attrs []string
	for _, fn := range fieldNames {
		if compareExcludedFields[fn.Value] {
			continue
		}
		// Skip event/link sub-fields.
		if strings.HasPrefix(fn.Value, otelpb.EventPrefix) || strings.HasPrefix(fn.Value, otelpb.LinkPrefix) {
			continue
		}
		attrs = append(attrs, fn.Value)
	}

	if len(attrs) == 0 {
		return nil, nil
	}

	// Step 3: Per-attribute parallel queries.
	results := make([]compareAttrResult, len(attrs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 16) // concurrency limit
	var queryErr error
	var queryErrMu sync.Mutex

	// Build the selection filter (base AND compare filter).
	selFilter := t.baseFilter
	if t.compareFilter != "" && t.compareFilter != "*" {
		if selFilter == "*" {
			selFilter = t.compareFilter
		} else {
			selFilter = selFilter + " AND " + t.compareFilter
		}
	}

	// Determine selection time range.
	selStartNs := params.start.UnixNano()
	selEndNs := params.end.UnixNano()
	if t.selectionStartNs > 0 && t.selectionEndNs > 0 {
		selStartNs = t.selectionStartNs
		selEndNs = t.selectionEndNs
	}

	for i, attr := range attrs {
		wg.Add(1)
		go func(i int, attr string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			quotedAttr := quoteLogsQLField(attr)
			ar := compareAttrResult{
				attrName:  attr,
				baseline:  make(map[string]map[int64]float64),
				selection: make(map[string]map[int64]float64),
			}

			// Baseline: count per value over full time range.
			baseQ := t.baseFilter + " | stats by (" + quotedAttr + ") count() as value"
			baseCounts, err := runCountQuery(ctx, cp, baseQ, attr, params.start.UnixNano(), params.end.UnixNano(), params.step)
			if err != nil {
				queryErrMu.Lock()
				queryErr = err
				queryErrMu.Unlock()
				return
			}
			ar.baseline = baseCounts

			// Selection: count per value over selection window.
			selQ := selFilter + " | stats by (" + quotedAttr + ") count() as value"
			selCounts, err := runCountQuery(ctx, cp, selQ, attr, selStartNs, selEndNs, params.step)
			if err != nil {
				queryErrMu.Lock()
				queryErr = err
				queryErrMu.Unlock()
				return
			}
			ar.selection = selCounts

			results[i] = ar
		}(i, attr)
	}
	wg.Wait()

	if queryErr != nil {
		return nil, queryErr
	}

	// Step 4: Build series with topN.
	return buildCompareSeries(results, t.topN), nil
}

// runCountQuery runs a `stats by (attr) count()` query and returns value → timestamp → count.
func runCountQuery(ctx context.Context, cp *tracecommon.CommonParams, logsQLStr, attrName string, startNs, endNs, stepNs int64) (map[string]map[int64]float64, error) {
	q, err := logstorage.ParseQueryAtTimestamp(logsQLStr, endNs)
	if err != nil {
		return nil, fmt.Errorf("cannot parse query [%s]: %w", logsQLStr, err)
	}
	q.AddTimeFilter(startNs, endNs)

	labelFields, err := q.GetStatsLabelsAddGroupingByTime(stepNs, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare stats query: %w", err)
	}

	counts := make(map[string]map[int64]float64)
	var mu sync.Mutex

	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		rowsCount := db.RowsCount()
		columns := db.GetColumns(false)

		for i := range rowsCount {
			ts := q.GetTimestamp()
			var attrValue string

			for _, c := range columns {
				if c.Name == "_time" {
					nsec, ok := logstorage.TryParseTimestampRFC3339Nano(c.Values[i])
					if ok {
						ts = nsec
					}
					continue
				}
				if slices.Contains(labelFields, c.Name) && c.Name == attrName {
					attrValue = strings.Clone(c.Values[i])
				}
			}

			for _, c := range columns {
				if slices.Contains(labelFields, c.Name) || c.Name == "_time" {
					continue
				}
				v, _ := strconv.ParseFloat(c.Values[i], 64)
				mu.Lock()
				if counts[attrValue] == nil {
					counts[attrValue] = make(map[int64]float64)
				}
				counts[attrValue][ts] = v
				mu.Unlock()
			}
		}
	}

	cpCopy := *cp
	cpCopy.Query = q
	qctx := cpCopy.NewQueryContext(ctx)
	defer cpCopy.UpdatePerQueryStatsMetrics()

	if err := vtstorage.RunQuery(qctx, writeBlock); err != nil {
		return nil, err
	}
	return counts, nil
}

// buildCompareSeries builds the Tempo compare response series from per-attribute results.
func buildCompareSeries(results []compareAttrResult, topN int) []tempoMetricsSeries {
	if topN <= 0 {
		topN = 10
	}

	// Compute divergence per attribute and sort by it descending.
	type attrDivergence struct {
		idx        int
		divergence float64
	}
	divergences := make([]attrDivergence, 0, len(results))
	for i, ar := range results {
		if len(ar.baseline) == 0 && len(ar.selection) == 0 {
			continue
		}
		divergences = append(divergences, attrDivergence{
			idx:        i,
			divergence: computeDivergence(ar),
		})
	}
	sort.Slice(divergences, func(i, j int) bool {
		return divergences[i].divergence > divergences[j].divergence
	})

	var allSeries []tempoMetricsSeries

	for _, ad := range divergences {
		ar := results[ad.idx]

		traceQLName := traceql.VTFieldToTraceQL(ar.attrName)

		// Rank values by total count (baseline + selection combined).
		type valueTotal struct {
			value string
			total float64
		}
		totals := make(map[string]float64)
		for v, tsCounts := range ar.baseline {
			for _, c := range tsCounts {
				totals[v] += c
			}
		}
		for v, tsCounts := range ar.selection {
			for _, c := range tsCounts {
				totals[v] += c
			}
		}

		ranked := make([]valueTotal, 0, len(totals))
		for v, t := range totals {
			ranked = append(ranked, valueTotal{v, t})
		}
		sort.Slice(ranked, func(i, j int) bool {
			return ranked[i].total > ranked[j].total
		})
		if len(ranked) > topN {
			ranked = ranked[:topN]
		}

		// Collect all timestamps across baseline and selection.
		tsSet := make(map[int64]bool)
		for _, tsCounts := range ar.baseline {
			for ts := range tsCounts {
				tsSet[ts] = true
			}
		}
		for _, tsCounts := range ar.selection {
			for ts := range tsCounts {
				tsSet[ts] = true
			}
		}
		timestamps := make([]int64, 0, len(tsSet))
		for ts := range tsSet {
			timestamps = append(timestamps, ts)
		}
		sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })

		// Compute totals per timestamp (across ALL values, not just topN).
		baseTotalByTs := make(map[int64]float64)
		selTotalByTs := make(map[int64]float64)
		for _, tsCounts := range ar.baseline {
			for ts, c := range tsCounts {
				baseTotalByTs[ts] += c
			}
		}
		for _, tsCounts := range ar.selection {
			for ts, c := range tsCounts {
				selTotalByTs[ts] += c
			}
		}

		// Emit per-value series for topN values.
		for _, vt := range ranked {
			allSeries = append(allSeries,
				makeCompareSeries("baseline", traceQLName, vt.value, ar.baseline[vt.value], timestamps),
				makeCompareSeries("selection", traceQLName, vt.value, ar.selection[vt.value], timestamps),
			)
		}

		// Emit total series (one per attribute, not per value).
		allSeries = append(allSeries,
			makeCompareSeriesTotals("baseline_total", traceQLName, baseTotalByTs, timestamps),
			makeCompareSeriesTotals("selection_total", traceQLName, selTotalByTs, timestamps),
		)
	}

	return allSeries
}

// computeDivergence computes the total variation distance between the baseline and selection
// distributions for a single attribute. Higher values mean the attribute's distribution changed
// more between baseline and selection — making it more interesting for root-cause analysis.
func computeDivergence(ar compareAttrResult) float64 {
	// Sum total counts across all timestamps.
	baseTotal := 0.0
	selTotal := 0.0
	baseCounts := make(map[string]float64)
	selCounts := make(map[string]float64)

	for v, tsCounts := range ar.baseline {
		for _, c := range tsCounts {
			baseCounts[v] += c
			baseTotal += c
		}
	}
	for v, tsCounts := range ar.selection {
		for _, c := range tsCounts {
			selCounts[v] += c
			selTotal += c
		}
	}

	if baseTotal == 0 || selTotal == 0 {
		return 0
	}

	// Total variation distance: sum of |p_sel(v) - p_base(v)| over all values.
	allValues := make(map[string]bool)
	for v := range baseCounts {
		allValues[v] = true
	}
	for v := range selCounts {
		allValues[v] = true
	}

	divergence := 0.0
	for v := range allValues {
		baseProp := baseCounts[v] / baseTotal
		selProp := selCounts[v] / selTotal
		divergence += math.Abs(selProp - baseProp)
	}
	return divergence
}

func makeCompareSeries(metaType, attrName, attrValue string, tsCounts map[int64]float64, timestamps []int64) tempoMetricsSeries {
	samples := make([]tempoSample, len(timestamps))
	for i, ts := range timestamps {
		samples[i] = tempoSample{
			TimestampMs: ts / 1e6,
			Value:       tsCounts[ts], // 0 if missing
		}
	}
	return tempoMetricsSeries{
		Labels: []tempoLabel{
			{Key: "__meta_type", Value: metaType},
			{Key: attrName, Value: attrValue},
		},
		Samples: samples,
	}
}

func makeCompareSeriesTotals(metaType, attrName string, totalByTs map[int64]float64, timestamps []int64) tempoMetricsSeries {
	samples := make([]tempoSample, len(timestamps))
	for i, ts := range timestamps {
		samples[i] = tempoSample{
			TimestampMs: ts / 1e6,
			Value:       totalByTs[ts],
		}
	}
	return tempoMetricsSeries{
		Labels: []tempoLabel{
			{Key: "__meta_type", Value: metaType},
			{Key: attrName, Value: ""},
		},
		Samples: samples,
	}
}

// executeStatsQuery runs a single LogsQL stats query and returns Tempo series.
func executeStatsQuery(ctx context.Context, cp *tracecommon.CommonParams, logsQLStr string, byFields []string, params *metricsQueryRangeParam) ([]tempoMetricsSeries, error) {
	q, err := logstorage.ParseQueryAtTimestamp(logsQLStr, params.end.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("cannot parse query [%s]: %s", logsQLStr, err)
	}
	q.AddTimeFilter(params.start.UnixNano(), params.end.UnixNano())

	labelFields, err := q.GetStatsLabelsAddGroupingByTime(params.step, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare stats query: %s", err)
	}

	m := make(map[string]*metricsStatsSeries)
	var mLock sync.Mutex

	addPoint := func(key string, labels []logstorage.Field, p metricsStatsPoint) {
		mLock.Lock()
		ss := m[key]
		if ss == nil {
			ss = &metricsStatsSeries{
				key:    key,
				Labels: labels,
			}
			m[key] = ss
		}
		ss.Points = append(ss.Points, p)
		mLock.Unlock()
	}

	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		rowsCount := db.RowsCount()
		columns := db.GetColumns(false)

		clonedColumnNames := make([]string, len(columns))
		for i, c := range columns {
			clonedColumnNames[i] = strings.Clone(c.Name)
		}

		for i := range rowsCount {
			ts := q.GetTimestamp()
			labels := make([]logstorage.Field, 0, len(labelFields))

			for j, c := range columns {
				if c.Name == "_time" {
					nsec, ok := logstorage.TryParseTimestampRFC3339Nano(c.Values[i])
					if ok {
						ts = nsec
						continue
					}
				}
				if slices.Contains(labelFields, c.Name) {
					labels = append(labels, logstorage.Field{
						Name:  clonedColumnNames[j],
						Value: strings.Clone(c.Values[i]),
					})
				}
			}

			for j, c := range columns {
				if slices.Contains(labelFields, c.Name) || c.Name == "_time" {
					continue
				}

				v := strings.Clone(c.Values[i])

				// Special case: histogram() returns JSON bucket arrays.
				if v == "[]" || strings.HasPrefix(v, `[{"vmrange":"`) {
					var buckets []histogramBucket
					if err := json.Unmarshal([]byte(v), &buckets); err == nil {
						for _, bucket := range buckets {
							bucketLabels := make([]logstorage.Field, 0, len(labels)+1)
							bucketLabels = append(bucketLabels, filterByFields(labels, byFields)...)
							bucketLabels = append(bucketLabels, logstorage.Field{
								Name:  "duration",
								Value: vmrangeToSeconds(bucket.VMRange),
							})
							bp := metricsStatsPoint{
								Timestamp: ts,
								Value:     strconv.FormatUint(bucket.Hits, 10),
							}
							bucketKey := fmt.Sprintf("%d:%s:vmrange=%s", j, marshalLabels(labels), bucket.VMRange)
							addPoint(bucketKey, bucketLabels, bp)
						}
						continue
					}
				}

				p := metricsStatsPoint{
					Timestamp: ts,
					Value:     v,
				}
				key := fmt.Sprintf("%d:%s", j, marshalLabels(labels))
				addPoint(key, filterByFields(labels, byFields), p)
			}
		}
	}

	cpCopy := *cp
	cpCopy.Query = q
	qctx := cpCopy.NewQueryContext(ctx)
	defer cpCopy.UpdatePerQueryStatsMetrics()

	if err := vtstorage.RunQuery(qctx, writeBlock); err != nil {
		return nil, fmt.Errorf("cannot execute query [%s]: %s", logsQLStr, err)
	}

	rows := make([]*metricsStatsSeries, 0, len(m))
	for _, ss := range m {
		rows = append(rows, ss)
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].key < rows[j].key
	})

	return transformToTempoSeries(rows), nil
}

// parseMetricsQueryRangeParams parses query parameters for the metrics/query_range endpoint.
func parseMetricsQueryRangeParams(r *http.Request) (*metricsQueryRangeParam, error) {
	qp := r.URL.Query()

	p := &metricsQueryRangeParam{
		end: time.Now(),
	}
	p.start = p.end.Add(-1 * time.Hour)

	p.q = qp.Get("q")
	if p.q == "" {
		p.q = qp.Get("query")
	}
	if p.q == "" {
		return nil, fmt.Errorf("missing required parameter: q")
	}

	since := qp.Get("since")
	if since != "" {
		d, err := time.ParseDuration(since)
		if err != nil {
			return nil, fmt.Errorf("cannot parse 'since': %s", err)
		}
		p.start = p.end.Add(-d)
	}

	startStr := qp.Get("start")
	if startStr != "" {
		ts, ok := timeutil.TryParseUnixTimestamp(startStr)
		if !ok {
			return nil, fmt.Errorf("cannot parse 'start': %s", startStr)
		}
		p.start = time.Unix(ts/1e9, ts%1e9)
	}

	endStr := qp.Get("end")
	if endStr != "" {
		ts, ok := timeutil.TryParseUnixTimestamp(endStr)
		if !ok {
			return nil, fmt.Errorf("cannot parse 'end': %s", endStr)
		}
		p.end = time.Unix(ts/1e9, ts%1e9)
	}

	if p.start.After(p.end) {
		p.start = p.end.Add(-1 * time.Hour)
	}

	stepStr := qp.Get("step")
	if stepStr != "" {
		d, err := time.ParseDuration(stepStr)
		if err != nil {
			return nil, fmt.Errorf("cannot parse 'step': %s", err)
		}
		p.step = d.Nanoseconds()
	}

	if p.step <= 0 {
		rangeNs := p.end.Sub(p.start).Nanoseconds()
		p.step = rangeNs / 100
		minStep := int64(time.Second)
		if p.step < minStep {
			p.step = minStep
		}
	}

	return p, nil
}

type histogramBucket struct {
	VMRange string `json:"vmrange"`
	Hits    uint64 `json:"hits"`
}

// vmrangeToSeconds converts a vmrange (nanosecond boundaries) to its geometric mean in seconds.
// This matches Tempo's convention: Log2Bucketize(durationNanos) / time.Second
// e.g., "5.995e+08...6.813e+08" → "0.639" (seconds)
func vmrangeToSeconds(vmrange string) string {
	parts := strings.SplitN(vmrange, "...", 2)
	if len(parts) != 2 {
		return vmrange
	}
	lo, errLo := strconv.ParseFloat(parts[0], 64)
	hi, errHi := strconv.ParseFloat(parts[1], 64)
	if errLo != nil || errHi != nil {
		return vmrange
	}
	mid := math.Sqrt(lo * hi)
	seconds := mid / 1e9
	return strconv.FormatFloat(seconds, 'g', -1, 64)
}

// vmrangeToNanos converts a vmrange to its geometric mean as a nanosecond integer string.
// This matches what Grafana's Drilldown expects for constructing compare filters.
// e.g., "5.995e+08...6.813e+08" → "639000000"
func vmrangeToNanos(vmrange string) string {
	parts := strings.SplitN(vmrange, "...", 2)
	if len(parts) != 2 {
		return vmrange
	}
	lo, errLo := strconv.ParseFloat(parts[0], 64)
	hi, errHi := strconv.ParseFloat(parts[1], 64)
	if errLo != nil || errHi != nil {
		return vmrange
	}
	mid := math.Sqrt(lo * hi)
	return strconv.FormatInt(int64(mid), 10)
}

func humanizeVMRange(vmrange string) string {
	parts := strings.SplitN(vmrange, "...", 2)
	if len(parts) != 2 {
		return vmrange
	}
	lo, errLo := strconv.ParseFloat(parts[0], 64)
	hi, errHi := strconv.ParseFloat(parts[1], 64)
	if errLo != nil || errHi != nil {
		return vmrange
	}
	mid := math.Sqrt(lo * hi)
	return formatDurationNs(mid)
}

func formatDurationNs(ns float64) string {
	switch {
	case ns >= 60e9:
		return fmt.Sprintf("%.3g mins", ns/60e9)
	case ns >= 1e9:
		return fmt.Sprintf("%.3g s", ns/1e9)
	case ns >= 1e6:
		return fmt.Sprintf("%.3g ms", ns/1e6)
	case ns >= 1e3:
		return fmt.Sprintf("%.3g µs", ns/1e3)
	default:
		return fmt.Sprintf("%.3g ns", ns)
	}
}

func marshalLabels(labels []logstorage.Field) string {
	var sb strings.Builder
	for i, l := range labels {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(l.Name)
		sb.WriteByte('=')
		sb.WriteString(l.Value)
	}
	return sb.String()
}

func filterByFields(labels []logstorage.Field, byFields []string) []logstorage.Field {
	if len(byFields) == 0 {
		return labels
	}
	result := make([]logstorage.Field, 0, len(byFields))
	for _, l := range labels {
		if slices.Contains(byFields, l.Name) {
			result = append(result, l)
		}
	}
	return result
}
