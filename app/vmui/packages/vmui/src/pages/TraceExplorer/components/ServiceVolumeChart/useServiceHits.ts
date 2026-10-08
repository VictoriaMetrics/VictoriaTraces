import { useCallback, useEffect, useMemo, useRef, useState } from "preact/compat";
import { getLogsqlHitsUrl } from "../../../../api/logsql";
import { LogHits } from "../../../../api/types";
import { TimeParams } from "../../../../types";
import { useAppState } from "../../../../state/common/StateContext";
import { useTenant } from "../../../../hooks/useTenant";
import { buildServiceHitsParams, hitsToAlignedData, normalizeHits, RawLogHits } from "./hitsQuery";

interface HitsResult {
  hits: LogHits[];
  /** Period the hits were requested for; bucket positions and the chart axes depend on it */
  period: TimeParams | null;
}

const EMPTY_RESULT: HitsResult = { hits: [], period: null };

export function useServiceHits() {
  const { serverUrl } = useAppState();
  const tenant = useTenant();

  const [result, setResult] = useState<HitsResult>(EMPTY_RESULT);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string>();
  const abortControllerRef = useRef(new AbortController());

  const data = useMemo(
    () => (result.period ? hitsToAlignedData(result.hits, result.period.start, result.period.end) : hitsToAlignedData([], 0n, 0n)),
    [result]
  );

  useEffect(() => () => abortControllerRef.current.abort(), []);

  const fetchHits = useCallback(async (
    query: string, startNs: bigint, endNs: bigint, extraClauses: string[] = []
  ) => {
    const trimmed = query.trim();
    if (!trimmed) return;

    abortControllerRef.current.abort();
    const controller = new AbortController();
    abortControllerRef.current = controller;

    setIsLoading(true);
    setError(undefined);

    try {
      const response = await fetch(getLogsqlHitsUrl(serverUrl), {
        signal: controller.signal,
        method: "POST",
        headers: { ...tenant },
        body: buildServiceHitsParams(trimmed, startNs, endNs, extraClauses),
      });

      const text = await response.text();
      if (!response.ok) throw new Error(text);

      const payload = JSON.parse(text) as { hits?: RawLogHits[] };
      if (!payload.hits) throw new Error("No 'hits' field in response");

      setResult({ hits: normalizeHits(payload.hits), period: { start: startNs, end: endNs } });
    } catch (e) {
      if (controller.signal.aborted) return;
      setError(e instanceof Error ? e.message : String(e));
      setResult({ hits: [], period: { start: startNs, end: endNs } });
    } finally {
      if (abortControllerRef.current === controller) setIsLoading(false);
    }
  }, [serverUrl, tenant]);

  return { hits: result.hits, data, period: result.period, isLoading, error, fetchHits };
}
