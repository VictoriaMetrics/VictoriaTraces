import { describe, it, expect } from "vitest";
import { AlignedData } from "uplot";
import stack from "./stack";

describe("uplot/stack", () => {
  it("accumulates series and pairs each one with the next above it", () => {
    const data: AlignedData = [[1, 2], [10, 1], [5, 2]];
    const { data: stacked, bands } = stack(data, () => false);

    expect(stacked[1]).toEqual([10, 1]);
    expect(stacked[2]).toEqual([15, 3]);
    expect(bands).toEqual([{ series: [2, 1] }]);
  });

  it("does not accumulate omitted series into the others", () => {
    const data: AlignedData = [[1, 2], [10, 1], [5, 2]];
    const { data: stacked, bands } = stack(data, i => i === 1);

    expect(stacked[2]).toEqual([5, 2]);
    expect(bands).toEqual([]);
  });

  it("builds bands across an omitted series in the middle", () => {
    const data: AlignedData = [[1], [1], [2], [4]];
    const { data: stacked, bands } = stack(data, i => i === 2);

    expect(stacked[3]).toEqual([5]);
    expect(bands).toEqual([{ series: [3, 1] }]);
  });
});
