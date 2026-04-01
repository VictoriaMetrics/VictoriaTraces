package tempo

import (
	"sort"
	"strconv"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaTraces/lib/traceql"
)

// tempoMetricsSeries represents a single time series in the Tempo QueryRangeResponse format.
type tempoMetricsSeries struct {
	Labels  []tempoLabel
	Samples []tempoSample
}

// tempoLabel represents a label in the Tempo OTEL KeyValue format.
type tempoLabel struct {
	Key   string
	Value string
}

// tempoSample represents a single data point in a time series.
type tempoSample struct {
	TimestampMs int64
	Value       float64
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
func transformToTempoSeries(rows []*metricsStatsSeries) []tempoMetricsSeries {
	result := make([]tempoMetricsSeries, 0, len(rows))
	for _, ss := range rows {
		ts := tempoMetricsSeries{
			Labels: make([]tempoLabel, 0, len(ss.Labels)),
		}

		// Convert label field names back from VT internal names to TraceQL names.
		for _, label := range ss.Labels {
			ts.Labels = append(ts.Labels, tempoLabel{
				Key:   traceql.VTFieldToTraceQL(label.Name),
				Value: label.Value,
			})
		}

		// Convert data points.
		ts.Samples = make([]tempoSample, 0, len(ss.Points))
		for _, p := range ss.Points {
			value, _ := strconv.ParseFloat(p.Value, 64)
			ts.Samples = append(ts.Samples, tempoSample{
				TimestampMs: p.Timestamp / 1e6, // nanoseconds -> milliseconds
				Value:       value,
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
