import { describe, it, expect } from "vitest";
import {
  columnToTimeRangeNs,
  columnToTimeRangeUs,
  computeHeatmapTimeStepNs,
  durationUsToRow,
  rowToDurationRangeUs,
  timeUsToColumn,
} from "./computeHeatmapGrid";

describe("TracesHeatmap/computeHeatmapGrid", () => {
  describe("rowToDurationRangeUs", () => {
    const boundaries = [10, 100, 1000];

    it("is open-ended below zero for row 0", () => {
      expect(rowToDurationRangeUs(0, boundaries)).toEqual([0, 10]);
    });

    it("covers the gap between consecutive boundaries for middle rows", () => {
      expect(rowToDurationRangeUs(1, boundaries)).toEqual([10, 100]);
      expect(rowToDurationRangeUs(2, boundaries)).toEqual([100, 1000]);
    });

    it("is open-ended above the last boundary for the last row", () => {
      expect(rowToDurationRangeUs(3, boundaries)).toEqual([1000, Infinity]);
    });
  });

  describe("columnToTimeRangeUs", () => {
    it("splits the period into equal, contiguous column ranges", () => {
      const periodStartUs = 1_000;
      const periodEndUs = 11_000;
      const columns = 5;

      let previousHigh = periodStartUs;
      for (let col = 0; col < columns; col++) {
        const [low, high] = columnToTimeRangeUs(col, periodStartUs, periodEndUs, columns);
        expect(low).toBe(previousHigh);
        expect(high - low).toBeCloseTo((periodEndUs - periodStartUs) / columns);
        previousHigh = high;
      }
      expect(previousHigh).toBeCloseTo(periodEndUs);
    });
  });

  describe("columnToTimeRangeNs", () => {
    it("splits the period into equal, contiguous column ranges with exact integer bounds", () => {
      const periodStartNs = 1_000_000n;
      const periodEndNs = 11_000_000n;
      const columns = 5;

      let previousHigh = periodStartNs;
      for (let col = 0; col < columns; col++) {
        const [low, high] = columnToTimeRangeNs(col, periodStartNs, periodEndNs, columns);
        expect(low).toBe(previousHigh);
        expect(high - low).toBe((periodEndNs - periodStartNs) / BigInt(columns));
        previousHigh = high;
      }
      expect(previousHigh).toBe(periodEndNs);
    });

    it("caps the last column's upper edge to periodEndNs even when the span doesn't divide evenly", () => {
      const periodStartNs = 0n;
      const periodEndNs = 1_000_007n; // not a multiple of 3
      const [, high] = columnToTimeRangeNs(2, periodStartNs, periodEndNs, 3);
      expect(high).toBe(periodEndNs);
    });

    it("never produces a zero-width step", () => {
      const step = computeHeatmapTimeStepNs(0n, 0n, 96);
      expect(step).toBe(1n);
    });
  });

  describe("durationUsToRow", () => {
    const boundaries = [10, 100, 1000];

    it("is the inverse of rowToDurationRangeUs at each boundary", () => {
      for (let row = 0; row <= boundaries.length; row++) {
        const [low] = rowToDurationRangeUs(row, boundaries);
        expect(durationUsToRow(low, boundaries)).toBe(row);
      }
    });

    it("places a value strictly below the first boundary in row 0", () => {
      expect(durationUsToRow(5, boundaries)).toBe(0);
    });

    it("places a value at or above the last boundary in the last row", () => {
      expect(durationUsToRow(1000, boundaries)).toBe(3);
      expect(durationUsToRow(1_000_000, boundaries)).toBe(3);
    });
  });

  describe("timeUsToColumn", () => {
    it("is the inverse of columnToTimeRangeUs for each column's own low edge", () => {
      const periodStartUs = 1_000;
      const periodEndUs = 11_000;
      const columns = 5;

      for (let col = 0; col < columns; col++) {
        const [low] = columnToTimeRangeUs(col, periodStartUs, periodEndUs, columns);
        expect(timeUsToColumn(low, periodStartUs, periodEndUs, columns)).toBe(col);
      }
    });

    it("clamps to the first column for a time before periodStartUs", () => {
      expect(timeUsToColumn(-500, 1_000, 11_000, 5)).toBe(0);
    });

    it("clamps to the last column for a time at or after periodEndUs", () => {
      expect(timeUsToColumn(11_000, 1_000, 11_000, 5)).toBe(4);
      expect(timeUsToColumn(50_000, 1_000, 11_000, 5)).toBe(4);
    });
  });
});
