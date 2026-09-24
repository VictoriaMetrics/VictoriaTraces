import { useCallback, useEffect, useMemo, useRef, useState } from "preact/compat";
import { getLogsqlQueryUrl } from "../../../../api/logsql";
import { parseLineToJSON } from "../../../../utils/json";
import { nanosToIsoString } from "../../../../utils/time";
import { useAppState } from "../../../../state/common/StateContext";
import { useTenant } from "../../../../hooks/useTenant";
import {
  HeatmapStatsRow,
  buildHeatmapErrorsQuery,
  buildHeatmapTracesQuery,
  getMaxCount,
  makeEmptyMatrix,
  parseHeatmapRows,
} from "./heatmapQuery";

export type { HeatmapGrid } from "./heatmapQuery";

const EMPTY_MATRIX = makeEmptyMatrix();

export function useHeatmapAggregation() {
  const { serverUrl } = useAppState();
  const tenant = useTenant();

  const [counts, setCounts] = useState<number[][]>(EMPTY_MATRIX);
  const [errors, setErrors] = useState<number[][]>(EMPTY_MATRIX);
  const [isLoading, setIsLoading] = useState(false);
  const [isErrorsLoading, setIsErrorsLoading] = useState(false);
  const [error, setError] = useState<string>();
  const abortControllerRef = useRef(new AbortController());

  const maxCount = useMemo(() => getMaxCount(counts), [counts]);
  const grid = useMemo(() => ({ counts, errors, maxCount }), [counts, errors, maxCount]);

  useEffect(() => () => abortControllerRef.current.abort(), []);

  const fetchHeatmap = useCallback(async (
    query: string, startNs: bigint, endNs: bigint, extraFilters: string[] = []
  ) => {
    const trimmed = query.trim();
    if (!trimmed) return;

    abortControllerRef.current.abort();
    const controller = new AbortController();
    abortControllerRef.current = controller;
    const { signal } = controller;

    setIsLoading(true);
    setIsErrorsLoading(true);
    setError(undefined);
    setErrors(EMPTY_MATRIX);

    const load = async (
      heatmapQuery: string,
      setMatrix: (matrix: number[][]) => void,
      setLoading: (isLoading: boolean) => void,
    ) => {
      try {
        // No `limit` here: the table's row limit must not truncate the set stats is computed over.
        const response = await fetch(getLogsqlQueryUrl(serverUrl), {
          signal,
          method: "POST",
          headers: {
            ...tenant,
            Accept: "application/stream+json",
          },
          body: new URLSearchParams({
            query: heatmapQuery,
            start: nanosToIsoString(startNs),
            end: nanosToIsoString(endNs),
          }),
        });

        const text = await response.text();
        if (!response.ok) throw new Error(text);

        const rows = text.split("\n").map(parseLineToJSON).filter(Boolean) as HeatmapStatsRow[];
        setMatrix(parseHeatmapRows(rows, startNs, endNs));
      } catch (e) {
        if (signal.aborted) return;
        setError(e instanceof Error ? e.message : String(e));
        setMatrix(EMPTY_MATRIX);
      } finally {
        if (abortControllerRef.current === controller) setLoading(false);
      }
    };

    await Promise.all([
      load(buildHeatmapTracesQuery(trimmed, extraFilters, startNs, endNs), setCounts, setIsLoading),
      load(buildHeatmapErrorsQuery(trimmed, extraFilters, startNs, endNs), setErrors, setIsErrorsLoading),
    ]);
  }, [serverUrl, tenant]);

  return { grid, isLoading, isErrorsLoading, error, fetchHeatmap };
}
