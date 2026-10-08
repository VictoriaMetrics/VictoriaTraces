import { describe, it, expect } from "vitest";
import {
  buildServiceHitsQuery,
  getHitsStepNs,
  hitsToAlignedData,
  normalizeHits,
} from "./hitsQuery";
import { HITS_BARS_VIEW } from "../../../../constants/logs";

const SERVICE_FILTER = "\"resource_attr:service.name\":*";

describe("ServiceVolumeChart/hitsQuery", () => {
  describe("buildServiceHitsQuery", () => {
    it("only keeps rows with a service name for an unfiltered query", () => {
      expect(buildServiceHitsQuery(" * ", [])).toBe(SERVICE_FILTER);
      expect(buildServiceHitsQuery("", [])).toBe(SERVICE_FILTER);
    });

    it("combines user filters and extra filters, dropping user pipes", () => {
      const query = buildServiceHitsQuery("a OR b | stats count()", ["x:in(\"1\")"]);
      expect(query).toBe(`(a OR b) AND (x:in("1")) AND ${SERVICE_FILTER}`);
    });
  });

  describe("getHitsStepNs", () => {
    it("splits the period into HITS_BARS_VIEW buckets, rounding up", () => {
      const endNs = BigInt(HITS_BARS_VIEW) * 1_000_000_000n + 1n;
      expect(getHitsStepNs(0n, endNs)).toBe(1_000_000_001n);
    });

    it("never goes below one millisecond", () => {
      expect(getHitsStepNs(0n, 10n)).toBe(1_000_000n);
    });
  });

  describe("normalizeHits", () => {
    it("marks hits without fields as other and sorts other first, then by total desc", () => {
      const hits = normalizeHits([
        { fields: { svc: "b" }, timestamps: [], values: [], total: 5 },
        { fields: {}, timestamps: [], values: [], total: 1 },
        { fields: { svc: "a" }, timestamps: [], values: [], total: 9 },
      ]);
      expect(hits.map(h => [h._isOther, h.total])).toEqual([[true, 1], [false, 9], [false, 5]]);
    });
  });

  describe("hitsToAlignedData", () => {
    const stepNs = 10_000_000_000n; // 10s per bucket
    const startNs = 1_005_000_000_000n; // epoch + 1005s, not aligned to the step
    const endNs = startNs + BigInt(HITS_BARS_VIEW) * stepNs;

    it("returns empty axes when there are no hits", () => {
      expect(hitsToAlignedData([], startNs, endNs)).toEqual([[], []]);
    });

    it("builds an epoch-aligned x axis over the period and maps values by bucket, leaving gaps null", () => {
      const hit = {
        fields: { svc: "a" },
        timestamps: ["1970-01-01T00:16:50Z", "1970-01-01T00:17:10Z"], // 1010s, 1030s
        values: [3, 7],
        total: 10,
        _isOther: false,
      };

      const [x, y] = hitsToAlignedData([hit], startNs, endNs);

      expect(x.length).toBe(HITS_BARS_VIEW + 1);
      expect(x[0]).toBe(1000);
      expect(x[1]).toBe(1010);
      expect(x[x.length - 1]).toBe(1000 + HITS_BARS_VIEW * 10);
      expect(y.slice(0, 4)).toEqual([null, 3, null, 7]);
    });
  });
});
