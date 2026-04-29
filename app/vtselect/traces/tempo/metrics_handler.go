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
		valueScale := 1.0
		if translation.scaleDurationToSeconds {
			valueScale = 1e-9
		}
		allSeries, err = executeStatsQuery(ctx, cp, translation.baseQuery, translation.byFields, params, valueScale)
		if err != nil {
			httpserver.Errorf(w, r, "cannot execute query: %s", err)
			return
		}

		// Collect exemplars — sample trace IDs for clickable links in Grafana.
		exemplars, exemplarErr := collectExemplars(ctx, cp, translation.baseFilter, params.start.UnixNano(), params.end.UnixNano(), params.step, defaultMaxExemplars)
		if exemplarErr == nil && len(exemplars) > 0 {
			attachExemplarsToSeries(allSeries, exemplars)
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
	otelpb.TraceIDIndexHasRootSpan:        true,
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

	// hitsBaseline and hitsSelection are total span counts where the attribute is observed,
	// taken from the cheap GetFieldNames pre-rank. Used for volume-weighting in the final sort.
	hitsBaseline  uint64
	hitsSelection uint64
	// coverageShift = |selectionHits/selTotal - baselineHits/baseTotal|, captured from pass 1.
	coverageShift float64
}

const (
	// compareCoverageThreshold filters out attributes whose presence/absence
	// doesn't shift meaningfully between baseline and selection.
	compareCoverageThreshold = 0.001 // 0.1 percentage point
	// maxCompareAttributes caps pass-2 fan-out for pathological cases (hundreds of
	// shifting attributes). Sorted-by-coverage-shift, so the most signal is preserved.
	maxCompareAttributes = 100
)

// executeCompareQuery discovers attributes and runs per-attribute count queries for compare().
func executeCompareQuery(ctx context.Context, cp *tracecommon.CommonParams, t *metricsQueryTranslation, params *metricsQueryRangeParam) ([]tempoMetricsSeries, error) {
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

	// Pass 1 (cheap pre-rank): discover field names + per-attribute hit counts on both sides.
	baselineHits, baselineTotal, err := discoverFieldHits(ctx, cp, t.baseFilter, params.start.UnixNano(), params.end.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("cannot discover baseline field hits: %w", err)
	}
	selectionHits, selectionTotal, err := discoverFieldHits(ctx, cp, selFilter, selStartNs, selEndNs)
	if err != nil {
		return nil, fmt.Errorf("cannot discover selection field hits: %w", err)
	}

	// Filter attributes by coverage shift; exclude internal/excluded fields.
	type candidate struct {
		name          string
		hitsBaseline  uint64
		hitsSelection uint64
		coverageShift float64
	}
	var candidates []candidate
	seen := make(map[string]bool)
	consider := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		if compareExcludedFields[name] {
			return
		}
		if strings.HasPrefix(name, otelpb.EventPrefix) || strings.HasPrefix(name, otelpb.LinkPrefix) {
			return
		}
		hb := baselineHits[name]
		hs := selectionHits[name]
		var bCov, sCov float64
		if baselineTotal > 0 {
			bCov = float64(hb) / float64(baselineTotal)
		}
		if selectionTotal > 0 {
			sCov = float64(hs) / float64(selectionTotal)
		}
		shift := math.Abs(sCov - bCov)
		if shift <= compareCoverageThreshold {
			return
		}
		candidates = append(candidates, candidate{name, hb, hs, shift})
	}
	for name := range baselineHits {
		consider(name)
	}
	for name := range selectionHits {
		consider(name)
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// If too many candidates, keep top maxCompareAttributes by coverage shift.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].coverageShift > candidates[j].coverageShift
	})
	if len(candidates) > maxCompareAttributes {
		candidates = candidates[:maxCompareAttributes]
	}

	// Pass 2: per-attribute parallel queries on the filtered list.
	results := make([]compareAttrResult, len(candidates))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 16) // concurrency limit
	var queryErr error
	var queryErrMu sync.Mutex

	for i, c := range candidates {
		wg.Add(1)
		go func(i int, c candidate) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			attr := c.name
			quotedAttr := quoteLogsQLField(attr)
			ar := compareAttrResult{
				attrName:      attr,
				baseline:      make(map[string]map[int64]float64),
				selection:     make(map[string]map[int64]float64),
				hitsBaseline:  c.hitsBaseline,
				hitsSelection: c.hitsSelection,
				coverageShift: c.coverageShift,
			}

			// Filter out rows without this attribute — otherwise empty values dominate the grouping.
			nonEmptyFilter := quotedAttr + `:*`

			// Baseline: count per value over full time range.
			baseQ := t.baseFilter + " AND " + nonEmptyFilter + " | stats by (" + quotedAttr + ") count() as value"
			baseCounts, err := runCountQuery(ctx, cp, baseQ, attr, params.start.UnixNano(), params.end.UnixNano(), params.step)
			if err != nil {
				queryErrMu.Lock()
				queryErr = err
				queryErrMu.Unlock()
				return
			}
			ar.baseline = baseCounts

			// Selection: count per value over selection window.
			selQ := selFilter + " AND " + nonEmptyFilter + " | stats by (" + quotedAttr + ") count() as value"
			selCounts, err := runCountQuery(ctx, cp, selQ, attr, selStartNs, selEndNs, params.step)
			if err != nil {
				queryErrMu.Lock()
				queryErr = err
				queryErrMu.Unlock()
				return
			}
			ar.selection = selCounts

			results[i] = ar
		}(i, c)
	}
	wg.Wait()

	if queryErr != nil {
		return nil, queryErr
	}

	// Step 4: Build series with topN.
	return buildCompareSeries(results, t.topN), nil
}

// runCountQuery runs a `stats by (attr) count()` query and returns value → timestamp → count.
// discoverFieldHits returns a map of attribute_name → hits (number of spans where the attribute
// is observed) using vtstorage.GetFieldNames, plus the total span count matching the filter.
// The total is used as the denominator when computing coverage = hits / total per attribute.
func discoverFieldHits(ctx context.Context, cp *tracecommon.CommonParams, filterStr string, startNs, endNs int64) (map[string]uint64, uint64, error) {
	// Field hits.
	q, err := logstorage.ParseQueryAtTimestamp(filterStr, endNs)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot parse filter [%s]: %w", filterStr, err)
	}
	q.AddTimeFilter(startNs, endNs)
	cpFn := *cp
	cpFn.Query = q
	qctx := cpFn.NewQueryContext(ctx)
	fieldNames, err := vtstorage.GetFieldNames(qctx)
	cpFn.UpdatePerQueryStatsMetrics()
	if err != nil {
		return nil, 0, err
	}
	hits := make(map[string]uint64, len(fieldNames))
	for _, fn := range fieldNames {
		hits[fn.Value] = fn.Hits
	}

	// Total span count.
	totalQ, err := logstorage.ParseQueryAtTimestamp(filterStr+" | stats count() as total", endNs)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot parse total query: %w", err)
	}
	totalQ.AddTimeFilter(startNs, endNs)
	cpTot := *cp
	cpTot.Query = totalQ
	qctxTot := cpTot.NewQueryContext(ctx)
	defer cpTot.UpdatePerQueryStatsMetrics()

	var total uint64
	var totalMu sync.Mutex
	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		rowsCount := db.RowsCount()
		columns := db.GetColumns(false)
		for _, c := range columns {
			if c.Name != "total" {
				continue
			}
			for i := 0; i < rowsCount; i++ {
				v, _ := strconv.ParseUint(c.Values[i], 10, 64)
				totalMu.Lock()
				total += v
				totalMu.Unlock()
			}
		}
	}
	if err := vtstorage.RunQuery(qctxTot, writeBlock); err != nil {
		return nil, 0, err
	}
	return hits, total, nil
}

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

	// Order by coverage shift (already computed during pass-1 filtering). The UI
	// (e.g. Grafana Traces Drilldown) re-sorts attributes client-side anyway, so we
	// just return them in a deterministic order — no need for a custom score.
	type attrOrder struct {
		idx   int
		score float64
	}
	scores := make([]attrOrder, 0, len(results))
	for i, ar := range results {
		if len(ar.baseline) == 0 && len(ar.selection) == 0 {
			continue
		}
		scores = append(scores, attrOrder{idx: i, score: ar.coverageShift})
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	var allSeries []tempoMetricsSeries

	for _, s := range scores {
		ar := results[s.idx]

		traceQLName := traceql.VTFieldToTraceQL(ar.attrName)

		// Reverse-map known numeric values to human-readable names.
		if ar.attrName == otelpb.StatusCodeField {
			ar.baseline = remapStatusValues(ar.baseline)
			ar.selection = remapStatusValues(ar.selection)
		}

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

func remapStatusValues(counts map[string]map[int64]float64) map[string]map[int64]float64 {
	result := make(map[string]map[int64]float64, len(counts))
	for v, ts := range counts {
		result[traceql.StatusCodeToName(v)] = ts
	}
	return result
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
// valueScale scales sample values (1 = no scaling, 1e-9 = ns → seconds for duration aggregations).
func executeStatsQuery(ctx context.Context, cp *tracecommon.CommonParams, logsQLStr string, byFields []string, params *metricsQueryRangeParam, valueScale float64) ([]tempoMetricsSeries, error) {
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

	if valueScale == 0 {
		valueScale = 1
	}
	return transformToTempoSeriesScaled(rows, valueScale), nil
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

const defaultMaxExemplars = 100

// collectExemplars samples trace IDs from spans matching the filter for use as exemplars.
// It runs a lightweight query to get a spread of trace IDs across the time range.
func collectExemplars(ctx context.Context, cp *tracecommon.CommonParams, filterStr string, startNs, endNs, stepNs int64, maxExemplars int) ([]tempoExemplar, error) {
	if maxExemplars <= 0 {
		maxExemplars = defaultMaxExemplars
	}

	// Query: sample spans spread across the time range using time-bucketed sampling.
	// Use uniq_values to get one trace_id per time bucket.
	bucketCount := maxExemplars
	bucketSize := (endNs - startNs) / int64(bucketCount)
	if bucketSize < 1e9 {
		bucketSize = 1e9 // minimum 1 second buckets
	}
	bucketSizeStr := strconv.FormatFloat(float64(bucketSize)/1e9, 'f', -1, 64) + "s"

	qStr := fmt.Sprintf("%s | stats by (_time:%s) any(%s) as tid, any(%s) as sid, any(%s) as dur",
		filterStr, bucketSizeStr, otelpb.TraceIDField, otelpb.SpanIDField, otelpb.DurationField)
	q, err := logstorage.ParseQueryAtTimestamp(qStr, endNs)
	if err != nil {
		return nil, err
	}
	q.AddTimeFilter(startNs, endNs)

	type rawExemplar struct {
		traceID  string
		spanID   string
		duration float64
		tsNs     int64
	}

	var exemplars []rawExemplar
	var mu sync.Mutex
	seen := make(map[string]bool) // dedup by trace_id

	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		rowsCount := db.RowsCount()
		columns := db.GetColumns(false)

		for i := range rowsCount {
			var traceID, spanID string
			var duration float64
			tsNs := q.GetTimestamp()

			for _, c := range columns {
				switch c.Name {
				case "tid":
					traceID = strings.Clone(c.Values[i])
				case "sid":
					spanID = strings.Clone(c.Values[i])
				case "dur":
					duration, _ = strconv.ParseFloat(c.Values[i], 64)
				case "_time":
					if nsec, ok := logstorage.TryParseTimestampRFC3339Nano(c.Values[i]); ok {
						tsNs = nsec
					}
				}
			}

			if traceID == "" {
				continue
			}

			mu.Lock()
			if !seen[traceID] && len(exemplars) < maxExemplars {
				seen[traceID] = true
				exemplars = append(exemplars, rawExemplar{
					traceID:  traceID,
					spanID:   spanID,
					duration: duration,
					tsNs:     tsNs,
				})
			}
			mu.Unlock()
		}
	}

	cpCopy := *cp
	cpCopy.Query = q
	qctx := cpCopy.NewQueryContext(ctx)
	defer cpCopy.UpdatePerQueryStatsMetrics()

	if err := vtstorage.RunQuery(qctx, writeBlock); err != nil {
		return nil, err
	}

	result := make([]tempoExemplar, len(exemplars))
	for i, e := range exemplars {
		result[i] = tempoExemplar{
			TraceID:     e.traceID,
			SpanID:      e.spanID,
			TimestampMs: e.tsNs / 1e6,
			Value:       e.duration / 1e9, // span duration in seconds
		}
	}
	return result, nil
}

// attachExemplarsToSeries distributes exemplars across series and sets each exemplar's
// value to the corresponding metric sample value so dots appear on the chart line.
func attachExemplarsToSeries(series []tempoMetricsSeries, exemplars []tempoExemplar) {
	if len(series) == 0 || len(exemplars) == 0 {
		return
	}

	// For single series (no by-clause), attach all exemplars.
	if len(series) == 1 {
		snapExemplarValues(exemplars, series[0].Samples)
		series[0].Exemplars = exemplars
		return
	}

	// For multiple series, distribute exemplars round-robin.
	for i := range exemplars {
		idx := i % len(series)
		series[idx].Exemplars = append(series[idx].Exemplars, exemplars[i])
	}
	// Snap values for each series.
	for i := range series {
		snapExemplarValues(series[i].Exemplars, series[i].Samples)
	}
}

// snapExemplarValues sets each exemplar's value to the nearest sample's value
// so exemplar dots appear on the chart line instead of at the bottom.
func snapExemplarValues(exemplars []tempoExemplar, samples []tempoSample) {
	if len(samples) == 0 {
		return
	}
	for i := range exemplars {
		bestIdx := 0
		bestDist := abs64(exemplars[i].TimestampMs - samples[0].TimestampMs)
		for j := 1; j < len(samples); j++ {
			d := abs64(exemplars[i].TimestampMs - samples[j].TimestampMs)
			if d < bestDist {
				bestDist = d
				bestIdx = j
			}
		}
		exemplars[i].Value = samples[bestIdx].Value
	}
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
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
