import { describe, it, expect } from "vitest";
import {
  buildHeatmapErrorsQuery,
  buildHeatmapTracesQuery,
  makeEmptyMatrix,
  getMaxCount,
  parseHeatmapRows,
  HeatmapStatsRow,
} from "./heatmapQuery";
import { columnToTimeRangeNs } from "./computeHeatmapGrid";
import { HEATMAP_DURATION_BUCKETS, HEATMAP_TIME_BUCKETS } from "./constants";

const INDEX = "{trace_id_idx_stream!=\"\"}";

describe("TracesHeatmap/heatmapQuery", () => {
  const startNs = 1_000_000_000n;
  const endNs = startNs + BigInt(HEATMAP_TIME_BUCKETS) * 10_000_000_000n; // 10s per column

  describe("buildHeatmapTracesQuery", () => {
    it("reads the trace index directly for an unfiltered query", () => {
      const query = buildHeatmapTracesQuery(" * ", [], startNs, endNs);
      expect(query.startsWith(`${INDEX} | math `)).toBe(true);
      expect(query).not.toContain("join");
      expect(query).toContain(`, ${HEATMAP_DURATION_BUCKETS - 1}), 0) as band | stats`);
      expect(query.endsWith("| stats by (_time:10000000000ns offset 1000000000ns, band) count() as count")).toBe(true);
    });

    it("joins traces matching the span filter and extra filters, dropping user pipes", () => {
      const query = buildHeatmapTracesQuery("a OR b | stats count()", ["x:in(\"1\")"], startNs, endNs);
      expect(query.startsWith(
        `${INDEX} | join by (trace_id_idx) ((a OR b) AND (x:in("1")) | uniq by (trace_id) `
        + "| rename trace_id as trace_id_idx) inner | math "
      )).toBe(true);
      expect(query).not.toContain("stats count()");
    });

    it("falls back to a minimal step for a zero-width period", () => {
      expect(buildHeatmapTracesQuery("*", [], 100n, 100n)).toContain("stats by (_time:1ns offset 0ns, band)");
    });
  });

  describe("buildHeatmapErrorsQuery", () => {
    it("joins traces with error spans", () => {
      expect(buildHeatmapErrorsQuery("*", [], startNs, endNs))
        .toContain(`${INDEX} | join by (trace_id_idx) ((status_code:="2") | uniq by (trace_id)`);
    });

    it("combines the span filter with the error status", () => {
      expect(buildHeatmapErrorsQuery("service:foo", [], startNs, endNs))
        .toContain("join by (trace_id_idx) ((service:foo) AND (status_code:=\"2\") | uniq by (trace_id)");
    });
  });

  describe("parseHeatmapRows", () => {
    it("puts each row into the column of its bucket start and the row of its band", () => {
      const [lowNs] = columnToTimeRangeNs(10, startNs, endNs, HEATMAP_TIME_BUCKETS);
      const time = new Date(Number(lowNs / 1_000_000n)).toISOString();
      const rows: HeatmapStatsRow[] = [
        { _time: time, band: "3", count: "5" },
        { _time: time, band: "4", count: "2" },
      ];

      const matrix = parseHeatmapRows(rows, startNs, endNs);
      expect(matrix[10][3]).toBe(5);
      expect(matrix[10][4]).toBe(2);
      expect(matrix[9][3]).toBe(0);
    });

    it("keeps nanosecond precision of bucket starts", () => {
      const periodStartNs = 1_700_000_000_123_456_789n;
      const periodEndNs = periodStartNs + BigInt(HEATMAP_TIME_BUCKETS) * 1_000_000n;
      const rows: HeatmapStatsRow[] = [{ _time: "2023-11-14T22:13:20.124456789Z", band: "0", count: "1" }];

      expect(parseHeatmapRows(rows, periodStartNs, periodEndNs)[1][0]).toBe(1);
    });

    it("clamps out-of-range bucket times into the first/last column", () => {
      const rows: HeatmapStatsRow[] = [
        { _time: new Date(0).toISOString(), band: "0", count: "1" },
        { _time: new Date(Number(endNs / 1_000_000n) + 1_000).toISOString(), band: "0", count: "1" },
      ];
      const matrix = parseHeatmapRows(rows, startNs, endNs);
      expect(matrix[0][0]).toBe(1);
      expect(matrix[HEATMAP_TIME_BUCKETS - 1][0]).toBe(1);
    });

    it("ignores invalid rows and returns an empty matrix for a zero-width period", () => {
      const time = new Date(Number(startNs / 1_000_000n)).toISOString();
      const rows: HeatmapStatsRow[] = [
        { _time: "not-a-date", band: "0", count: "1" },
        { _time: time, band: String(HEATMAP_DURATION_BUCKETS), count: "1" },
        { _time: time, band: "0", count: "abc" },
      ];
      expect(parseHeatmapRows(rows, startNs, endNs)).toEqual(makeEmptyMatrix());
      expect(parseHeatmapRows(rows, 0n, 0n)).toEqual(makeEmptyMatrix());
    });
  });

  describe("getMaxCount", () => {
    it("returns the highest trace count", () => {
      const counts = makeEmptyMatrix();
      counts[2][1] = 7;
      counts[5][0] = 3;
      expect(getMaxCount(counts)).toBe(7);
    });
  });
});
