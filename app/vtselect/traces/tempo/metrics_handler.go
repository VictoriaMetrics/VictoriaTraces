package tempo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/bits"
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
		allSeries, err = executeStatsQuery(ctx, cp, translation.baseQuery, translation.byFields, params, valueScale, translation.valueColumnLabelKey, translation.valueColumnLabels)
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
	otelpb.DurationField:               true,
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

// compareAttrResult holds per-attribute facet counts for baseline and selection.
//
// We don't bucket by time — facets returns one count per (attribute, value) over
// the whole window. The Drilldown comparison panel ranks by total magnitudes, not
// per-step shapes, so single-sample series is sufficient.
type compareAttrResult struct {
	attrName  string            // VT field name
	baseline  map[string]uint64 // value → hits in baseline window
	selection map[string]uint64 // value → hits in selection window

	// hitsBaseline / hitsSelection are sums across all values (i.e. total spans
	// where the attribute is observed). Used for volume-weighting in the final sort.
	hitsBaseline  uint64
	hitsSelection uint64
	// coverageShift = |selectionHits/selTotal - baselineHits/baseTotal|.
	coverageShift float64
}

const (
	// compareCoverageThreshold filters out attributes whose presence/absence
	// doesn't shift meaningfully between baseline and selection.
	compareCoverageThreshold = 0.001 // 0.1 percentage point
	// maxCompareAttributes caps pass-2 fan-out. Set to 16 to match the parallel
	// query semaphore — a single batch — instead of multiple queued waves.
	// Selection is volume-weighted (coverageShift × log(1 + max(hits))), so the
	// kept attributes are the ones that move at meaningful absolute volume.
	maxCompareAttributes = 16
)

// compareCandidateScore ranks compare attributes by combining how much their
// presence shifts between baseline and selection (coverageShift) with the
// log of the larger raw hit count. The log factor keeps the scale similar
// across orders of magnitude while suppressing rare-but-shifty noise.
func compareCandidateScore(hitsBaseline, hitsSelection uint64, coverageShift float64) float64 {
	h := hitsBaseline
	if hitsSelection > h {
		h = hitsSelection
	}
	return coverageShift * math.Log1p(float64(h))
}

// executeCompareQuery runs the compare() pipeline for a TraceQL metrics query.
//
// Strategy: two `| facets` queries (baseline + selection windows) in parallel +
// two `| stats count()` queries for total span counts (denominators). The facets
// pipe returns top values per field for ALL attributes in one go, replacing what
// used to be a 32-query fan-out (16 attrs × 2 windows).
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

	// Per-value cap: for the panel we want at least topN values per attribute,
	// but we ask facets for a bit more so we can intersect baseline and selection
	// and still have headroom after merging.
	valuesPerField := t.topN
	if valuesPerField <= 0 {
		valuesPerField = 10
	}
	if valuesPerField < 50 {
		valuesPerField = 50
	}

	// Run facets + totals for both windows in parallel.
	var (
		baseFacets                             facetResults
		selFacets                              facetResults
		baselineTotal                          uint64
		selectionTotal                         uint64
		errBase, errSel, errBaseTot, errSelTot error
		wg                                     sync.WaitGroup
	)
	wg.Add(4)
	go func() {
		defer wg.Done()
		baseFacets, errBase = runFacetsQuery(ctx, cp, t.baseFilter, params.start.UnixNano(), params.end.UnixNano(), valuesPerField)
	}()
	go func() {
		defer wg.Done()
		selFacets, errSel = runFacetsQuery(ctx, cp, selFilter, selStartNs, selEndNs, valuesPerField)
	}()
	go func() {
		defer wg.Done()
		baselineTotal, errBaseTot = runTotalCount(ctx, cp, t.baseFilter, params.start.UnixNano(), params.end.UnixNano())
	}()
	go func() {
		defer wg.Done()
		selectionTotal, errSelTot = runTotalCount(ctx, cp, selFilter, selStartNs, selEndNs)
	}()
	wg.Wait()
	for _, e := range []error{errBase, errSel, errBaseTot, errSelTot} {
		if e != nil {
			return nil, e
		}
	}

	// Build candidate list from union of facet field names. Apply exclusions and
	// coverage threshold; keep top-N by volume-weighted score.
	type candidate struct {
		attrName      string
		baseline      map[string]uint64
		selection     map[string]uint64
		hitsBaseline  uint64
		hitsSelection uint64
		coverageShift float64
	}
	seen := make(map[string]bool)
	var candidates []candidate
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
		baseValues := baseFacets[name]
		selValues := selFacets[name]
		if len(baseValues) == 0 && len(selValues) == 0 {
			return
		}
		var hb, hs uint64
		for _, c := range baseValues {
			hb += c
		}
		for _, c := range selValues {
			hs += c
		}
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
		candidates = append(candidates, candidate{
			attrName:      name,
			baseline:      baseValues,
			selection:     selValues,
			hitsBaseline:  hb,
			hitsSelection: hs,
			coverageShift: shift,
		})
	}
	for name := range baseFacets {
		consider(name)
	}
	for name := range selFacets {
		consider(name)
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		return compareCandidateScore(candidates[i].hitsBaseline, candidates[i].hitsSelection, candidates[i].coverageShift) >
			compareCandidateScore(candidates[j].hitsBaseline, candidates[j].hitsSelection, candidates[j].coverageShift)
	})
	if len(candidates) > maxCompareAttributes {
		candidates = candidates[:maxCompareAttributes]
	}

	results := make([]compareAttrResult, len(candidates))
	for i, c := range candidates {
		results[i] = compareAttrResult(c)
	}

	// Use the time range end as the single sample timestamp for emitted series.
	endTimestampMs := params.end.UnixNano() / 1e6
	return buildCompareSeries(results, t.topN, endTimestampMs, baselineTotal, selectionTotal), nil
}

// facetResults maps field_name → field_value → hits.
type facetResults map[string]map[string]uint64

// runFacetsQuery runs a `<filter> | facets <limit>` query and returns the field/value/hits map.
// maxFacetUniqueValuesPerField caps how many distinct values facets tracks per field
// before dropping the field entirely. Lower = faster (less hash table churn), at the
// cost of dropping high-cardinality fields. 500 is a conservative choice that keeps
// most useful comparison dimensions while pruning trace_id / span_id / http.url-style
// fields that aren't useful for compare.
const maxFacetUniqueValuesPerField = 500

func runFacetsQuery(ctx context.Context, cp *tracecommon.CommonParams, filterStr string, startNs, endNs int64, valuesPerField int) (facetResults, error) {
	if valuesPerField <= 0 {
		valuesPerField = 50
	}
	qStr := fmt.Sprintf("%s | facets %d max_values_per_field %d", filterStr, valuesPerField, maxFacetUniqueValuesPerField)
	q, err := logstorage.ParseQueryAtTimestamp(qStr, endNs)
	if err != nil {
		return nil, fmt.Errorf("cannot parse facets query [%s]: %w", qStr, err)
	}
	q.AddTimeFilter(startNs, endNs)

	results := make(facetResults)
	var mu sync.Mutex
	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		rowsCount := db.RowsCount()
		columns := db.GetColumns(false)
		var fnCol, fvCol, hitsCol *logstorage.BlockColumn
		for i := range columns {
			switch columns[i].Name {
			case "field_name":
				fnCol = &columns[i]
			case "field_value":
				fvCol = &columns[i]
			case "hits":
				hitsCol = &columns[i]
			}
		}
		if fnCol == nil || fvCol == nil || hitsCol == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		for i := 0; i < rowsCount; i++ {
			name := strings.Clone(fnCol.Values[i])
			value := strings.Clone(fvCol.Values[i])
			hits, _ := strconv.ParseUint(hitsCol.Values[i], 10, 64)
			m := results[name]
			if m == nil {
				m = make(map[string]uint64)
				results[name] = m
			}
			m[value] += hits
		}
	}

	cpCopy := *cp
	cpCopy.Query = q
	qctx := cpCopy.NewQueryContext(ctx)
	defer cpCopy.UpdatePerQueryStatsMetrics()
	if err := vtstorage.RunQuery(qctx, writeBlock); err != nil {
		return nil, err
	}
	return results, nil
}

// runTotalCount runs a `<filter> | stats count() as total` query and returns the total.
func runTotalCount(ctx context.Context, cp *tracecommon.CommonParams, filterStr string, startNs, endNs int64) (uint64, error) {
	qStr := filterStr + " | stats count() as total"
	q, err := logstorage.ParseQueryAtTimestamp(qStr, endNs)
	if err != nil {
		return 0, fmt.Errorf("cannot parse total query: %w", err)
	}
	q.AddTimeFilter(startNs, endNs)

	var total uint64
	var mu sync.Mutex
	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		rowsCount := db.RowsCount()
		columns := db.GetColumns(false)
		for _, c := range columns {
			if c.Name != "total" {
				continue
			}
			for i := 0; i < rowsCount; i++ {
				v, _ := strconv.ParseUint(c.Values[i], 10, 64)
				mu.Lock()
				total += v
				mu.Unlock()
			}
		}
	}
	cpCopy := *cp
	cpCopy.Query = q
	qctx := cpCopy.NewQueryContext(ctx)
	defer cpCopy.UpdatePerQueryStatsMetrics()
	if err := vtstorage.RunQuery(qctx, writeBlock); err != nil {
		return 0, err
	}
	return total, nil
}

// buildCompareSeries builds the Tempo compare response series from per-attribute results.
//
// Series carry a single sample at endTimestampMs since facets returns one count per
// (attribute, value) over the whole window — Drilldown's compare panel ranks by total
// magnitudes, not per-step shapes.
//
// baselineTotalSpans / selectionTotalSpans are the GLOBAL span counts (regardless of
// attribute presence). They're emitted as `baseline_total` / `selection_total` series
// so Drilldown computes percentages against a consistent denominator across all
// attributes — sparse attributes like has_error don't get truncated denominators.
func buildCompareSeries(results []compareAttrResult, topN int, endTimestampMs int64, baselineTotalSpans, selectionTotalSpans uint64) []tempoMetricsSeries {
	if topN <= 0 {
		topN = 10
	}

	// Order by the same volume-weighted score used during pre-rank so the emit order
	// matches the trim order. Drilldown re-sorts client-side, but keeping the orderings
	// consistent makes the response deterministic.
	type attrOrder struct {
		idx   int
		score float64
	}
	scores := make([]attrOrder, 0, len(results))
	for i, ar := range results {
		if len(ar.baseline) == 0 && len(ar.selection) == 0 {
			continue
		}
		scores = append(scores, attrOrder{
			idx:   i,
			score: compareCandidateScore(ar.hitsBaseline, ar.hitsSelection, ar.coverageShift),
		})
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
			total uint64
		}
		totals := make(map[string]uint64)
		for v, c := range ar.baseline {
			totals[v] += c
		}
		for v, c := range ar.selection {
			totals[v] += c
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

		// Emit per-value series for topN values.
		for _, vt := range ranked {
			allSeries = append(allSeries,
				makeCompareSeries("baseline", traceQLName, vt.value, ar.baseline[vt.value], endTimestampMs),
				makeCompareSeries("selection", traceQLName, vt.value, ar.selection[vt.value], endTimestampMs),
			)
		}

		// Emit total series — one per attribute. Uses the GLOBAL span counts so the
		// denominator is consistent across attributes (sparse attrs like has_error
		// don't get truncated totals from per-attribute facet sums).
		allSeries = append(allSeries,
			makeCompareSeriesTotals("baseline_total", traceQLName, baselineTotalSpans, endTimestampMs),
			makeCompareSeriesTotals("selection_total", traceQLName, selectionTotalSpans, endTimestampMs),
		)
	}

	return allSeries
}

func remapStatusValues(counts map[string]uint64) map[string]uint64 {
	result := make(map[string]uint64, len(counts))
	for v, c := range counts {
		result[traceql.StatusCodeToName(v)] = c
	}
	return result
}

func makeCompareSeries(metaType, attrName, attrValue string, count uint64, timestampMs int64) tempoMetricsSeries {
	return tempoMetricsSeries{
		Labels: []tempoLabel{
			{Key: "__meta_type", Value: metaType},
			{Key: attrName, Value: attrValue},
		},
		Samples: []tempoSample{{TimestampMs: timestampMs, Value: float64(count)}},
	}
}

func makeCompareSeriesTotals(metaType, attrName string, total uint64, timestampMs int64) tempoMetricsSeries {
	return tempoMetricsSeries{
		Labels: []tempoLabel{
			{Key: "__meta_type", Value: metaType},
			{Key: attrName, Value: ""},
		},
		Samples: []tempoSample{{TimestampMs: timestampMs, Value: float64(total)}},
	}
}

// executeStatsQuery runs a single LogsQL stats query and returns Tempo series.
// valueScale scales sample values (1 = no scaling, 1e-9 = ns → seconds for duration aggregations).
//
// When valueColumnLabelKey is non-empty, each non-label value column produces a
// separate Tempo series with an extra label {valueColumnLabelKey:
// valueColumnLabels[columnName]}. This is how Tempo distinguishes per-quantile
// series in `quantile_over_time(...)` responses.
func executeStatsQuery(ctx context.Context, cp *tracecommon.CommonParams, logsQLStr string, byFields []string, params *metricsQueryRangeParam, valueScale float64, valueColumnLabelKey string, valueColumnLabels map[string]string) ([]tempoMetricsSeries, error) {
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
						// Re-bin VictoriaLogs' fine log-scale buckets into Tempo's
						// Log2Bucketize layout (next power of 2 in nanoseconds). Multiple
						// vmranges that map to the same Tempo bin have their hits summed,
						// so the response matches Tempo's coarser histogram_over_time
						// shape used by Grafana's heatmap renderer.
						binHits := make(map[uint64]uint64, len(buckets))
						for _, bucket := range buckets {
							binNs := tempoBucketNs(bucket.VMRange)
							if binNs == 0 {
								// VictoriaLogs emits a "0" vmrange for sub-resolution samples.
								// Tempo has no equivalent bucket — drop them.
								continue
							}
							binHits[binNs] += bucket.Hits
						}
						for binNs, hits := range binHits {
							bucketLabels := make([]logstorage.Field, 0, len(labels)+1)
							bucketLabels = append(bucketLabels, filterByFields(labels, byFields)...)
							// Match Tempo's response shape: numeric `__bucket` label so
							// Grafana's Tempo datasource recognises the series as heatmap
							// rows. The encoder emits this label as doubleValue.
							bucketLabels = append(bucketLabels, logstorage.Field{
								Name:  histogramBucketLabelName,
								Value: strconv.FormatFloat(float64(binNs)/1e9, 'g', -1, 64),
							})
							bp := metricsStatsPoint{
								Timestamp: ts,
								Value:     strconv.FormatUint(hits, 10),
							}
							bucketKey := fmt.Sprintf("%d:%s:bin=%d", j, marshalLabels(labels), binNs)
							addPoint(bucketKey, bucketLabels, bp)
						}
						continue
					}
				}

				p := metricsStatsPoint{
					Timestamp: ts,
					Value:     v,
				}
				pointLabels := filterByFields(labels, byFields)
				if valueColumnLabelKey != "" {
					if labelValue, ok := valueColumnLabels[c.Name]; ok {
						extended := make([]logstorage.Field, 0, len(pointLabels)+1)
						extended = append(extended, pointLabels...)
						extended = append(extended, logstorage.Field{
							Name:  valueColumnLabelKey,
							Value: labelValue,
						})
						pointLabels = extended
					}
				}
				key := fmt.Sprintf("%d:%s", j, marshalLabels(labels))
				addPoint(key, pointLabels, p)
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

// histogramBucketLabelName is the label key VT uses for the per-bucket boundary
// in histogram_over_time responses. It matches Tempo's `__bucket` convention so
// Grafana's Tempo datasource renders the result as a heatmap. The label value
// is the geometric-mean bucket midpoint in seconds and is emitted as a numeric
// (doubleValue) JSON field by transformToTempoSeriesImpl.
const histogramBucketLabelName = "__bucket"

// tempoBucketNs returns the Tempo Log2Bucketize bin (next power of 2 in
// nanoseconds) that the vmrange's upper bound falls into. This matches
// Tempo's histogram_over_time response, which uses 2^k nanosecond bins
// instead of VictoriaLogs' ~18-per-decade log-scale ladder.
//
// vmrange may be either a single number ("0", "1e-5") representing a point
// bucket, or a "lo...hi" pair. The bin is computed from `hi` so that hits
// in [lo, hi] map to the smallest power-of-2 bin covering hi — the same
// rule Tempo applies to each span duration.
//
// Returns 0 for the underflow bucket (sub-nanosecond `hi`) so the caller
// can drop it. Tempo never emits a sub-microsecond bucket — these are
// degenerate zero-duration spans VictoriaLogs lumps into `"0...1e-09"`.
func tempoBucketNs(vmrange string) uint64 {
	hi := 0.0
	if parts := strings.SplitN(vmrange, "...", 2); len(parts) == 2 {
		hi, _ = strconv.ParseFloat(parts[1], 64)
	} else {
		hi, _ = strconv.ParseFloat(vmrange, 64)
	}
	if hi < 1 {
		return 0
	}
	n := uint64(math.Ceil(hi))
	if n <= 1 {
		return 1
	}
	return 1 << bits.Len64(n-1)
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
