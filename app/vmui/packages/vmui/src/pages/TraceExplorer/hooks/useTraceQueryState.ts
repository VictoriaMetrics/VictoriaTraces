import { useCallback } from "preact/compat";
import { useSearchParams } from "react-router-dom";

export const TRACE_QUERY_URL_PARAMS = {
  QUERY: "logsql_query",
  TRACE_ID: "trace_id",
} as const;

const DEFAULT_QUERY = "*";

export const useTraceQueryState = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const query = searchParams.get(TRACE_QUERY_URL_PARAMS.QUERY) || DEFAULT_QUERY;
  const traceId = searchParams.get(TRACE_QUERY_URL_PARAMS.TRACE_ID) || "";

  const setParam = useCallback((key: string, value: string | null) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev);
      if (value) next.set(key, value);
      else next.delete(key);
      return next;
    });
  }, [setSearchParams]);

  const setQuery = useCallback((value: string) => {
    setParam(TRACE_QUERY_URL_PARAMS.QUERY, value || null);
  }, [setParam]);

  const setTraceId = useCallback((value: string) => {
    setParam(TRACE_QUERY_URL_PARAMS.TRACE_ID, value || null);
  }, [setParam]);

  return {
    query,
    setQuery,
    traceId,
    setTraceId,
  };
};
