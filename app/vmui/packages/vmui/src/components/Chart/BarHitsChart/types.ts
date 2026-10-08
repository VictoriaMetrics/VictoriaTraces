export enum GRAPH_STYLES {
  BAR = "Bars",
}

export interface GraphOptions {
  graphStyle: GRAPH_STYLES;
  stacked: boolean;
  fill: boolean;
}

export type HitsFilterMode = "include" | "exclude";

/** Called from the legend menu with the series value (e.g. a service name). */
export type ApplyHitsFilter = (value: string, mode: HitsFilterMode) => void;
