import { useCallback, useEffect, useRef, useState } from "preact/compat";
import dayjs from "dayjs";
import { getLogsqlQueryUrl } from "../../../api/logsql";
import { parseLineToJSON } from "../../../utils/json";
import { useAppState } from "../../../state/common/StateContext";
import { useTenant } from "../../../hooks/useTenant";

export type TraceSummary = {
  traceID: string;
  service: string;
  operation: string;
  duration: number;
  startTime: number;
  spanCount: number;
  errorCount: number;
};

// OTel status code, as encoded by the LogsQL `status_code` field: 0=UNSET, 1=OK, 2=ERROR.
export const ERROR_STATUS_CODE = "2";

export interface SpanLogLine {
  _time: string;
  name: string;
  trace_id: string;
  span_id: string;
  parent_span_id?: string;
  duration: string;
  start_time_unix_nano: string;
  end_time_unix_nano: string;
  kind?: string;
  status_code?: string;
  "resource_attr:service.name"?: string;
  [key: string]: string | undefined;
}

function groupIntoTraces(spans: SpanLogLine[]): { traces: TraceSummary[]; spansByTraceId: Map<string, SpanLogLine[]> } {
  const spansByTraceId = new Map<string, SpanLogLine[]>();
  spans.forEach(span => {
    if (!span?.trace_id) return;
    const list = spansByTraceId.get(span.trace_id) || [];
    list.push(span);
    spansByTraceId.set(span.trace_id, list);
  });

  const traces = Array.from(spansByTraceId.entries()).map(([traceID, traceSpans]): TraceSummary => {
    const root = traceSpans.find(s => !s.parent_span_id) ||
      traceSpans.reduce((a, b) => (Number(a.start_time_unix_nano) <= Number(b.start_time_unix_nano) ? a : b));

    return {
      traceID,
      service: root["resource_attr:service.name"] || "",
      operation: root.name || "",
      duration: Math.round((Number(root.duration) || 0) / 1000),
      startTime: Math.round((Number(root.start_time_unix_nano) || 0) / 1000),
      spanCount: traceSpans.length,
      errorCount: traceSpans.filter(s => s.status_code === ERROR_STATUS_CODE).length,
    };
  });

  return { traces, spansByTraceId };
}

export function useLogsqlTracesSearch() {
  const { serverUrl } = useAppState();
  const tenant = useTenant();

  const [traces, setTraces] = useState<TraceSummary[]>([]);
  const [spansByTraceId, setSpansByTraceId] = useState<Map<string, SpanLogLine[]>>(() => new Map());
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string>();
  const abortControllerRef = useRef(new AbortController());

  useEffect(() => () => abortControllerRef.current.abort(), []);

  const search = useCallback(async (query: string, startNs: bigint, endNs: bigint, limit: number) => {
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
        body: new URLSearchParams({
          query: query.trim(),
          limit: `${limit}`,
          start: startIso,
          end: endIso,
        }),
      });

      const text = await response.text();
      if (!response.ok) {
        setError(text);
        setTraces([]);
        setSpansByTraceId(new Map());
        return;
      }

      const lines = text.split("\n").map(parseLineToJSON).filter(Boolean) as SpanLogLine[];
      const grouped = groupIntoTraces(lines);
      setTraces(grouped.traces);
      setSpansByTraceId(grouped.spansByTraceId);
    } catch (e) {
      if (e instanceof Error && e.name !== "AbortError") {
        setError(String(e));
        setTraces([]);
        setSpansByTraceId(new Map());
        console.error(e);
      }
    } finally {
      if (abortControllerRef.current === controller) {
        setIsLoading(false);
      }
    }
  }, [serverUrl, tenant]);

  return { traces, spansByTraceId, isLoading, error, search };
}
