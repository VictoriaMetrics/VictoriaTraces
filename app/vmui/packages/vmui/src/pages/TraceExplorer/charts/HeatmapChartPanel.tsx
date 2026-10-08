import { FC, useCallback } from "preact/compat";
import TracesHeatmap from "../components/TracesHeatmap";
import { useHeatmapAggregation } from "../components/TracesHeatmap/useHeatmapAggregation";
import { useTimePeriod } from "../hooks/useTimePeriod";
import { useTraceExplorerChartContext } from "./TraceExplorerChartContext";
import { useChartRun } from "./useChartRun";
import { TraceChartPanelProps, TraceChartRun } from "./types";

const HeatmapChartPanel: FC<TraceChartPanelProps> = ({ run, isActive }) => {
  const { grid, period: gridPeriod, isLoading, isErrorsLoading, error, fetchHeatmap } = useHeatmapAggregation();
  const { heatmap } = useTraceExplorerChartContext();
  const { period: livePeriod } = useTimePeriod();

  const load = useCallback((chartRun: TraceChartRun) => {
    fetchHeatmap(chartRun.query, chartRun.startNs, chartRun.endNs, chartRun.extraClauses);
  }, [fetchHeatmap]);
  useChartRun(isActive, run, load);

  const periodStart = gridPeriod?.start ?? livePeriod.start;
  const periodEnd = gridPeriod?.end ?? livePeriod.end;

  return (
    <TracesHeatmap
      grid={grid}
      isLoading={isLoading}
      isErrorsLoading={isErrorsLoading}
      error={error}
      periodStart={periodStart}
      periodEnd={periodEnd}
      minDurationUs={heatmap.minDurationUs}
      maxDurationUs={heatmap.maxDurationUs}
      highlightedTrace={heatmap.highlightedTrace}
      onSelectionChange={heatmap.onSelectionChange}
      onCommitSelection={heatmap.onCommitSelection}
    />
  );
};

export default HeatmapChartPanel;
