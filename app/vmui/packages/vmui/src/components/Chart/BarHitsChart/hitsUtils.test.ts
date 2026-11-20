import { describe, it, expect } from "vitest";
import { toggleHiddenLabels } from "./hitsUtils";
import stack from "../../../utils/uplot/stack";
import { AlignedData } from "uplot";

const LABELS = ["a", "b", "c"];

describe("BarHitsChart/hitsUtils", () => {
  describe("toggleHiddenLabels", () => {
    it("isolates the clicked series on a plain click", () => {
      expect(toggleHiddenLabels(LABELS, new Set(), "b", false)).toEqual(new Set(["a", "c"]));
    });

    it("shows all series back when the clicked one is already the only visible", () => {
      expect(toggleHiddenLabels(LABELS, new Set(["a", "c"]), "b", false)).toEqual(new Set());
    });

    it("switches isolation to another series", () => {
      expect(toggleHiddenLabels(LABELS, new Set(["a", "c"]), "a", false)).toEqual(new Set(["b", "c"]));
    });

    it("toggles a single series on a multi-select click", () => {
      expect(toggleHiddenLabels(LABELS, new Set(), "b", true)).toEqual(new Set(["b"]));
      expect(toggleHiddenLabels(LABELS, new Set(["b", "c"]), "b", true)).toEqual(new Set(["c"]));
    });
  });

  describe("stacking with hidden series", () => {
    it("does not accumulate hidden series into the visible ones", () => {
      const data: AlignedData = [[1, 2], [10, 1], [5, 2]];
      const { data: stacked, bands } = stack(data, i => i === 1);

      expect(stacked[2]).toEqual([5, 2]);
      expect(bands).toEqual([]);
    });
  });
});
