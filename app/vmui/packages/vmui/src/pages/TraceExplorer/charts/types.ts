import { FC } from "preact/compat";

/**
 * Snapshot of a committed query run. Chart panels load their data from it, so they always
 * fetch what the user ran, not whatever the URL state holds at render time.
 */
export interface TraceChartRun {
  /** Grows with every committed run, including re-runs of the same query. */
  id: number;
  query: string;
  startNs: bigint;
  endNs: bigint;
  extraClauses: string[];
}

export interface TraceChartPanelProps {
  /** `null` until the page commits its first run. */
  run: TraceChartRun | null;
  isActive: boolean;
}

export interface TraceChartDefinition {
  /** Value of the `chart` URL parameter. */
  id: string;
  /** Tab label. */
  label: string;
  Component: FC<TraceChartPanelProps>;
}
