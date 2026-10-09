import HeatmapChartPanel from "./HeatmapChartPanel";
import ServiceVolumeChartPanel from "./ServiceVolumeChartPanel";
import { TraceChartDefinition } from "./types";

export type { TraceChartDefinition, TraceChartPanelProps, TraceChartRun } from "./types";
export { CHART_MODE_URL_PARAM, resolveChartId } from "./chartRegistry";

/** Charts shown above the traces table, in tab order. The first one is the default. */
export const TRACE_CHARTS: TraceChartDefinition[] = [
  { id: "heatmap", label: "Heatmap", Component: HeatmapChartPanel },
  { id: "volume", label: "Volume by service", Component: ServiceVolumeChartPanel },
];
