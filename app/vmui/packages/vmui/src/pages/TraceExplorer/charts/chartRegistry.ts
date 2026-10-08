import { TraceChartDefinition } from "./types";

export const CHART_MODE_URL_PARAM = "chart";

/** Maps a `chart` URL value to a registered chart id; unknown values fall back to the first (default) chart. */
export function resolveChartId(value: string | null, charts: TraceChartDefinition[]): string {
  const known = charts.find(chart => chart.id === value);
  return (known ?? charts[0]).id;
}
