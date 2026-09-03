import { describe, it, expect } from "vitest";
import { buildHeatmapStatsQuery, parseHeatmapRows, HeatmapStatsRow } from "./heatmapQuery";
import { columnToTimeRangeNs } from "./computeHeatmapGrid";
import { HEATMAP_DURATION_BAND_BOUNDARIES_US, HEATMAP_DURATION_BUCKETS, HEATMAP_TIME_BUCKETS } from "./constants";

describe("TracesHeatmap/heatmapQuery", () => {
  describe("buildHeatmapStatsQuery", () => {
    it("appends a stats pipe bucketing by _time and by duration bands, without a limit", () => {
      const startNs = 0n;
      const endNs = BigInt(HEATMAP_TIME_BUCKETS) * 1_000_000_000n; // 1s per column
      const query = buildHeatmapStatsQuery("service:foo", startNs, endNs);

      expect(query.startsWith("service:foo | stats by (trace_id) ")).toBe(true);
      expect(query).toContain(" | stats by (_time:1000000000ns) ");
      expect(query).not.toMatch(/\blimit\b/i);

      const firstBoundaryNs = HEATMAP_DURATION_BAND_BOUNDARIES_US[0] * 1000;
      const lastBoundaryNs = HEATMAP_DURATION_BAND_BOUNDARIES_US[HEATMAP_DURATION_BAND_BOUNDARIES_US.length - 1] * 1000;
      expect(query).toContain(`count() if (duration:<${firstBoundaryNs}) b0`);
      expect(query).toContain(`count() if (duration:>=${lastBoundaryNs}) b${HEATMAP_DURATION_BAND_BOUNDARIES_US.length}`);
      // every field name b0..bN must appear exactly once
      for (let i = 0; i < HEATMAP_DURATION_BUCKETS; i++) {
        expect(query).toContain(`b${i}`);
      }
    });

    it("trims the filter query and falls back to a minimal step for a zero-width period", () => {
      const query = buildHeatmapStatsQuery("  error  ", 100n, 100n);
      expect(query.startsWith("error | stats by (trace_id) ")).toBe(true);
      expect(query).toContain(" | stats by (_time:1ns) ");
    });
  });

  describe("parseHeatmapRows", () => {
    it("buckets rows into the column matching their _time and sums counts per band", () => {
      const periodStartNs = 0n;
      const columnWidthNs = 1_000_000_000n; // 1s per column, so ms-precision _time round-trips exactly
      const periodEndNs = columnWidthNs * BigInt(HEATMAP_TIME_BUCKETS);
      // land squarely in column 10
      const rowTimeMs = Number(periodStartNs + 10n * columnWidthNs + columnWidthNs / 2n) / 1_000_000;

      const rows: HeatmapStatsRow[] = [
        { _time: new Date(rowTimeMs).toISOString(), b0: "3", b1: "5" },
      ];

      const grid = parseHeatmapRows(rows, periodStartNs, periodEndNs);
      expect(grid.counts[10][0]).toBe(3);
      expect(grid.counts[10][1]).toBe(5);
      expect(grid.maxCount).toBe(5);
    });

    it("clamps out-of-range bucket times into the first/last column", () => {
      const periodEndNs = 1_000_000_000n * BigInt(HEATMAP_TIME_BUCKETS);
      const rows: HeatmapStatsRow[] = [
        { _time: new Date(-1_000).toISOString(), b0: "1" },
        { _time: new Date(Number(periodEndNs / 1_000_000n) + 1_000).toISOString(), b0: "1" },
      ];
      const grid = parseHeatmapRows(rows, 0n, periodEndNs);
      expect(grid.counts[0][0]).toBe(1);
      expect(grid.counts[HEATMAP_TIME_BUCKETS - 1][0]).toBe(1);
    });

    it("ignores rows with an unparseable _time and returns an empty grid for a zero-width period", () => {
      const rows: HeatmapStatsRow[] = [{ _time: "not-a-date", b0: "1" }];
      expect(parseHeatmapRows(rows, 0n, 100_000_000n).maxCount).toBe(0);
      expect(parseHeatmapRows(rows, 0n, 0n).maxCount).toBe(0);
    });

    it("assigns a row landing exactly on a column boundary to the column columnToTimeRangeNs reports for it, even when periodStartNs isn't step-aligned", () => {
      // A real `start` timestamp is essentially never an exact multiple of the step - this
      // is what previously let a selection built from columnToTimeRangeUs (floating-point,
      // re-derived independently of this function) omit rows sitting right at a boundary.
      const periodStartNs = 1_234_000_000n;
      const stepNs = 10_000_000n; // 10ms
      const periodEndNs = periodStartNs + stepNs * BigInt(HEATMAP_TIME_BUCKETS);

      const [lowNs] = columnToTimeRangeNs(50, periodStartNs, periodEndNs, HEATMAP_TIME_BUCKETS);
      const row: HeatmapStatsRow = { _time: new Date(Number(lowNs / 1_000_000n)).toISOString(), b0: "1" };

      const grid = parseHeatmapRows([row], periodStartNs, periodEndNs);
      expect(grid.counts[50][0]).toBe(1);
      expect(grid.counts[49][0]).toBe(0);
    });
  });
});
