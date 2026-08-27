import { useCallback, useEffect, useRef, useState } from "preact/compat";
import dayjs from "dayjs";
import { getLogsqlQueryUrl } from "../../../../api/logsql";
import { parseLineToJSON } from "../../../../utils/json";
import { useAppState } from "../../../../state/common/StateContext";
import { useTenant } from "../../../../hooks/useTenant";
import { HeatmapGrid, HeatmapStatsRow, buildHeatmapStatsQuery, makeEmptyGrid, parseHeatmapRows } from "./heatmapQuery";

export type { HeatmapGrid } from "./heatmapQuery";

export function useHeatmapAggregation() {
  const { serverUrl } = useAppState();
  const tenant = useTenant();

  const [grid, setGrid] = useState<HeatmapGrid>(makeEmptyGrid);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string>();
  const abortControllerRef = useRef(new AbortController());

  useEffect(() => () => abortControllerRef.current.abort(), []);

  const fetchHeatmap = useCallback(async (query: string, startNs: bigint, endNs: bigint) => {
    const trimmed = query.trim();
    if (!trimmed) return;

    abortControllerRef.current.abort();
    const controller = new AbortController();
    abortControllerRef.current = controller;
    const { signal } = controller;

    setIsLoading(true);
    setError(undefined);

    try {
      const url = getLogsqlQueryUrl(serverUrl);
      const startIso = dayjs(Number(startNs / 1_000_000n)).toISOString();
      const endIso = dayjs(Number(endNs / 1_000_000n)).toISOString();

      const response = await fetch(url, {
        signal,
        method: "POST",
        headers: {
          ...tenant,
          Accept: "application/stream+json",
        },
        // No `limit` here: this is a full-range aggregation via `stats`, and the
        // table's row limit must not truncate the set stats is computed over.
        body: new URLSearchParams({
          query: buildHeatmapStatsQuery(trimmed, startNs, endNs),
          start: startIso,
          end: endIso,
        }),
      });

      const text = await response.text();
      if (!response.ok) {
        setError(text);
        setGrid(makeEmptyGrid());
        return;
      }

      const rows = text.split("\n").map(parseLineToJSON).filter(Boolean) as HeatmapStatsRow[];
      setGrid(parseHeatmapRows(rows, startNs, endNs));
    } catch (e) {
      if (e instanceof Error && e.name !== "AbortError") {
        setError(String(e));
        setGrid(makeEmptyGrid());
        console.error(e);
      }
    } finally {
      if (abortControllerRef.current === controller) {
        setIsLoading(false);
      }
    }
  }, [serverUrl, tenant]);

  return { grid, isLoading, error, fetchHeatmap };
}
