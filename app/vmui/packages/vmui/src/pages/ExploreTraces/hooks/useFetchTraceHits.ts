import { useCallback, useMemo, useRef, useState } from "preact/compat";
import { getTraceHitsUrl } from "../../../api/traces";
import { ErrorTypes, TimeParams } from "../../../types";
import { TraceHits } from "../../../api/types";
import { getHitsTimeParams } from "../../../utils/traces";
import { TRACES_GROUP_BY, TRACES_LIMIT_HITS } from "../../../constants/traces";
import { isEmptyObject } from "../../../utils/object";
import { useEffect } from "react";
import { useTenant } from "../../../hooks/useTenant";

export const useFetchTraceHits = (server: string, query: string) => {
  const tenant = useTenant();
  const [traceHits, setTraceHits] = useState<TraceHits[]>([]);
  const [isLoading, setIsLoading] = useState<{[key: number]: boolean;}>([]);
  const [error, setError] = useState<ErrorTypes | string>();
  const abortControllerRef = useRef(new AbortController());

  const url = useMemo(() => getTraceHitsUrl(server), [server]);

  const getOptions = (query: string, period: TimeParams, signal: AbortSignal) => {
    const { start, end, step } = getHitsTimeParams(period);

    return {
      signal,
      method: "POST",
      headers: {
        ...tenant,
      },
      body: new URLSearchParams({
        query: query.trim(),
        step: `${step}ms`,
        start: start.toISOString(),
        end: end.toISOString(),
        fields_limit: `${TRACES_LIMIT_HITS}`,
        field: TRACES_GROUP_BY,
      })
    };
  };

  const fetchTraceHits = useCallback(async (period: TimeParams) => {
    abortControllerRef.current.abort();
    abortControllerRef.current = new AbortController();
    const { signal } = abortControllerRef.current;

    const id = Date.now();
    setIsLoading(prev => ({ ...prev, [id]: true }));
    setError(undefined);

    try {
      const options = getOptions(query, period, signal);
      const response = await fetch(url, options);

      if (!response.ok || !response.body) {
        const text = await response.text();
        setError(text);
        setTraceHits([]);
        setIsLoading(prev => ({ ...prev, [id]: false }));
        return;
      }

      const data = await response.json();
      const hits = data?.hits as TraceHits[];
      if (!hits) {
        const error = "Error: No 'hits' field in response";
        setError(error);
      }

      setTraceHits(hits.map(markIsOther).sort(sortHits));
    } catch (e) {
      if (e instanceof Error && e.name !== "AbortError") {
        setError(String(e));
        console.error(e);
        setTraceHits([]);
      }
    }
    setIsLoading(prev => ({ ...prev, [id]: false }));
  }, [url, query, tenant]);

  useEffect(() => {
    return () => {
      abortControllerRef.current.abort();
    };
  }, []);

  return {
    traceHits,
    isLoading: Object.values(isLoading).some(s => s),
    error,
    fetchTraceHits,
    abortController: abortControllerRef.current
  };
};

// Helper function to check if a hit is "other"
const markIsOther = (hit: TraceHits) => ({
  ...hit,
  _isOther: isEmptyObject(hit.fields)
});

// Comparison function for sorting hits
const sortHits = (a: TraceHits, b: TraceHits) => {
  if (a._isOther !== b._isOther) {
    return a._isOther ? -1 : 1; // "Other" hits first to avoid graph overlap
  }
  return b.total - a.total; // Sort remaining by total for better visibility
};
