import { describe, it, expect } from "vitest";
import { getHitKey, pruneHiddenSeries, sortLogHits, toggleHiddenSeries } from "./hitsUtils";

const KEYS = ["a", "b", "c"];

describe("BarHitsChart/hitsUtils", () => {
  describe("toggleHiddenSeries", () => {
    it("isolates the clicked series on a plain click", () => {
      expect(toggleHiddenSeries(KEYS, new Set(), "b", false)).toEqual(new Set(["a", "c"]));
    });

    it("shows all series back when the clicked one is already the only visible", () => {
      expect(toggleHiddenSeries(KEYS, new Set(["a", "c"]), "b", false)).toEqual(new Set());
    });

    it("switches isolation to another series", () => {
      expect(toggleHiddenSeries(KEYS, new Set(["a", "c"]), "a", false)).toEqual(new Set(["b", "c"]));
    });

    it("toggles a single series on a multi-select click", () => {
      expect(toggleHiddenSeries(KEYS, new Set(), "b", true)).toEqual(new Set(["b"]));
      expect(toggleHiddenSeries(KEYS, new Set(["b", "c"]), "b", true)).toEqual(new Set(["c"]));
    });
  });

  describe("getHitKey", () => {
    it("tells the aggregate series from a real one labelled \"other\"", () => {
      expect(getHitKey({ fields: {} })).not.toBe(getHitKey({ fields: { service: "other" } }));
    });
  });

  describe("pruneHiddenSeries", () => {
    it("drops keys of series that are no longer present", () => {
      expect(pruneHiddenSeries(new Set(["a", "gone"]), KEYS)).toEqual(new Set(["a"]));
    });

    it("returns the same set when nothing is dropped", () => {
      const hidden = new Set(["a"]);
      expect(pruneHiddenSeries(hidden, KEYS)).toBe(hidden);
    });
  });

  describe("sortLogHits", () => {
    it("keeps only the aggregate series last, not a real one labelled \"other\"", () => {
      const items = [
        { label: "other", isOther: true, total: 100 },
        { label: "other", isOther: false, total: 1 },
        { label: "a", isOther: false, total: 5 },
      ];
      expect(items.sort(sortLogHits("total")).map(item => item.total)).toEqual([5, 1, 100]);
    });
  });
});
