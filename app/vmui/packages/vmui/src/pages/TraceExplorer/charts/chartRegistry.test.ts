import { describe, it, expect } from "vitest";
import { resolveChartId } from "./chartRegistry";
import { TraceChartDefinition } from "./types";

const Noop = () => null;

const CHARTS: TraceChartDefinition[] = [
  { id: "heatmap", label: "Heatmap", Component: Noop },
  { id: "volume", label: "Volume by service", Component: Noop },
];

describe("TraceExplorer/charts/resolveChartId", () => {
  it("returns a known chart id as is", () => {
    expect(resolveChartId("volume", CHARTS)).toBe("volume");
  });

  it("falls back to the first registered chart for an unknown or missing value", () => {
    expect(resolveChartId("nope", CHARTS)).toBe("heatmap");
    expect(resolveChartId(null, CHARTS)).toBe("heatmap");
  });
});
