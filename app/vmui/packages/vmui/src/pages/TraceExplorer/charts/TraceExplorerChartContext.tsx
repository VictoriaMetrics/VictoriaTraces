import { createContext } from "preact";
import { useContext } from "preact/compat";
import { HeatmapSelectionRange, HighlightedTrace } from "../components/TracesHeatmap/TracesHeatmap";
import { HitsFilterMode } from "../../../components/Chart/BarHitsChart/types";

/** Page state the heatmap is coupled to: table preview selection, duration bounds and the highlighted row. */
export interface HeatmapChartBindings {
  minDurationUs: number;
  maxDurationUs: number;
  highlightedTrace: HighlightedTrace | null;
  onSelectionChange: (selection: HeatmapSelectionRange | null) => void;
  onCommitSelection: (selection: HeatmapSelectionRange) => void;
}

export interface TraceExplorerChartContextValue {
  heatmap: HeatmapChartBindings;
  /** Adds (`include`) an extra-filter chip or appends a negative clause (`exclude`) to the query and re-runs it. */
  applyFieldFilter: (field: string, value: string, mode: HitsFilterMode) => void;
}

const TraceExplorerChartContext = createContext<TraceExplorerChartContextValue | null>(null);

export const TraceExplorerChartProvider = TraceExplorerChartContext.Provider;

export function useTraceExplorerChartContext(): TraceExplorerChartContextValue {
  // eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
  const value = useContext(TraceExplorerChartContext);
  if (!value) throw new Error("useTraceExplorerChartContext must be used within TraceExplorerChartProvider");
  return value;
}
