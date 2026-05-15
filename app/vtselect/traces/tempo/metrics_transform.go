package tempo

import (
	"sort"
	"strconv"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/traceql"
)

// tempoMetricsSeries represents a single time series in the Tempo QueryRangeResponse format.
type tempoMetricsSeries struct {
	Labels    []tempoLabel
	Samples   []tempoSample
	Exemplars []tempoExemplar
}

// tempoExemplar represents an exemplar linking a metric data point to a specific trace.
type tempoExemplar struct {
	TraceID     string
	SpanID      string
	TimestampMs int64
	Value       float64 // span duration in seconds, or NaN
}

// tempoLabel represents a label in the Tempo OTEL KeyValue format.
//
// Numeric marks the label as carrying a numeric (doubleValue) instead of a
// stringValue. Tempo emits numeric labels for histogram bucket boundaries
// (`__bucket`) and quantile percentiles (`p`); Grafana's Tempo datasource
// stringifies stringValue labels with JSON quoting, which breaks heatmap
// binning when the value is meant to be ordinal.
type tempoLabel struct {
	Key     string
	Value   string
	Numeric bool
}

// tempoSample represents a single data point in a time series.
type tempoSample struct {
	TimestampMs int64
	Value       float64
}

// isNumericLabel reports whether the label's value should be encoded as a
// JSON doubleValue in the Tempo OTEL KeyValue response. Tempo emits histogram
// boundaries (`__bucket`) and quantile percentiles (`p`) as numeric labels;
// stringValue labels are JSON-quoted by Grafana's Tempo datasource, which
// breaks heatmap binning and percentile ordering.
func isNumericLabel(name string) bool {
	return name == "__bucket" || name == "p"
}

// metricsStatsSeries mirrors the statsSeries from logsql but is local to avoid cross-package dependency.
type metricsStatsSeries struct {
	key    string
	Labels []logstorage.Field
	Points []metricsStatsPoint
}

// metricsStatsPoint represents a single data point collected from VictoriaLogs stats query.
type metricsStatsPoint struct {
	Timestamp int64  // nanoseconds
	Value     string // string representation of the value
}

// transformToTempoSeries converts collected stats results to Tempo series format.
// valueScale is a multiplier applied to each sample value (use 1 for no scaling,
// or 1e-9 to convert nanosecond durations to seconds).
func transformToTempoSeriesScaled(rows []*metricsStatsSeries, valueScale float64) []tempoMetricsSeries {
	return transformToTempoSeriesImpl(rows, valueScale)
}

func transformToTempoSeriesImpl(rows []*metricsStatsSeries, valueScale float64) []tempoMetricsSeries {
	result := make([]tempoMetricsSeries, 0, len(rows))
	for _, ss := range rows {
		ts := tempoMetricsSeries{
			Labels: make([]tempoLabel, 0, len(ss.Labels)),
		}

		// Convert label field names back from VT internal names to TraceQL names.
		for _, label := range ss.Labels {
			ts.Labels = append(ts.Labels, tempoLabel{
				Key:     traceql.VTFieldToTraceQL(label.Name),
				Value:   label.Value,
				Numeric: isNumericLabel(label.Name),
			})
		}

		// Convert data points.
		ts.Samples = make([]tempoSample, 0, len(ss.Points))
		for _, p := range ss.Points {
			value, _ := strconv.ParseFloat(p.Value, 64)
			ts.Samples = append(ts.Samples, tempoSample{
				TimestampMs: p.Timestamp / 1e6, // nanoseconds -> milliseconds
				Value:       value * valueScale,
			})
		}

		// Sort samples by timestamp.
		sort.Slice(ts.Samples, func(i, j int) bool {
			return ts.Samples[i].TimestampMs < ts.Samples[j].TimestampMs
		})

		result = append(result, ts)
	}
	return result
}
