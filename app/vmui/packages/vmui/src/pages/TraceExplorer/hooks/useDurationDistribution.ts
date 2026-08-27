import { useEffect, useRef, useState } from "preact/compat";
import dayjs from "dayjs";
import { getLogsqlQueryUrl } from "../../../api/logsql";
import { parseLineToJSON } from "../../../utils/json";
import { useAppState } from "../../../state/common/StateContext";
import { useTenant } from "../../../hooks/useTenant";
import { OPERATION_FIELD, quoteLogsqlValue } from "../utils";

export const DURATION_HISTOGRAM_BUCKETS = 40;

export interface DurationRange {
  total: number;
  minDurationNs: number;
  maxDurationNs: number;
}

export interface DurationRangeStatsRow {
  total: string;
  min_duration: string;
  max_duration: string;
}

export interface DurationDistribution {
  total: number;
  minDurationNs: number;
  maxDurationNs: number;
  minDurationUs: number;
  maxDurationUs: number;
  bucketWidthUs: number;
  bucketCounts: number[];
}

export interface DurationHistogramStatsRow {
  [bucketField: string]: string;
}

// Collapses to one row per trace_id first, so a trace with more than one matching span
// (retries, recursive calls) is counted once rather than once per span.
const PER_TRACE_DURATION_PIPE = "stats by (trace_id) min(duration) as duration";

export function buildDurationRangeQuery(operation: string): string {
  return `${OPERATION_FIELD}:${quoteLogsqlValue(operation)} | ${PER_TRACE_DURATION_PIPE}` +
    " | stats count() as total, min(duration) as min_duration, max(duration) as max_duration";
}

export function parseDurationRange(rows: DurationRangeStatsRow[]): DurationRange | undefined {
  const row = rows[0];
  if (!row) return undefined;

  const total = Number(row.total) || 0;
  if (!total) return undefined;

  return {
    total,
    minDurationNs: Number(row.min_duration) || 0,
    maxDurationNs: Number(row.max_duration) || 0,
  };
}

function bucketBoundaryNs(minDurationNs: number, bucketWidthNs: number, bucketIndex: number): number {
  return Math.round(minDurationNs + bucketIndex * bucketWidthNs);
}

/** [low, high) bounds of bucket `i`, in nanoseconds - the exact same rounding used to build
 * the histogram query below, so a client-side duration always maps to the same bucket the
 * server counted it into. The last bucket's upper edge is nudged past maxDurationNs since it
 * would otherwise be exclusive and drop a trace whose duration lands exactly on it. */
export function durationBucketRangeNs(
  minDurationNs: number,
  maxDurationNs: number,
  bucketIndex: number
): [number, number] {
  const bucketWidthNs = Math.max(1, maxDurationNs - minDurationNs) / DURATION_HISTOGRAM_BUCKETS;
  const low = bucketBoundaryNs(minDurationNs, bucketWidthNs, bucketIndex);
  const high = bucketIndex === DURATION_HISTOGRAM_BUCKETS - 1
    ? maxDurationNs + 1
    : bucketBoundaryNs(minDurationNs, bucketWidthNs, bucketIndex + 1);
  return [low, high];
}

export function findDurationBucketIndex(durationNs: number, minDurationNs: number, maxDurationNs: number): number {
  for (let i = 0; i < DURATION_HISTOGRAM_BUCKETS; i++) {
    const [low, high] = durationBucketRangeNs(minDurationNs, maxDurationNs, i);
    if (durationNs >= low && durationNs < high) return i;
  }
  return durationNs < minDurationNs ? 0 : DURATION_HISTOGRAM_BUCKETS - 1;
}

export function buildDurationHistogramQuery(operation: string, minDurationNs: number, maxDurationNs: number): string {
  const clauses = Array.from({ length: DURATION_HISTOGRAM_BUCKETS }, (_, i) => {
    const [low, high] = durationBucketRangeNs(minDurationNs, maxDurationNs, i);
    return `count() if (duration:>=${low} AND duration:<${high}) b${i}`;
  });

  return `${OPERATION_FIELD}:${quoteLogsqlValue(operation)} | ${PER_TRACE_DURATION_PIPE} | stats ${clauses.join(", ")}`;
}

export function parseDurationHistogram(
  rows: DurationHistogramStatsRow[],
  range: DurationRange
): DurationDistribution {
  const row = rows[0];
  const bucketCounts = Array.from(
    { length: DURATION_HISTOGRAM_BUCKETS },
    (_, i) => (row ? Number(row[`b${i}`]) || 0 : 0)
  );

  return {
    total: range.total,
    minDurationNs: range.minDurationNs,
    maxDurationNs: range.maxDurationNs,
    minDurationUs: range.minDurationNs / 1000,
    maxDurationUs: range.maxDurationNs / 1000,
    bucketWidthUs: Math.max(1, range.maxDurationNs - range.minDurationNs) / 1000 / DURATION_HISTOGRAM_BUCKETS,
    bucketCounts,
  };
}

export function useDurationDistribution(operation: string, periodStart: bigint, periodEnd: bigint) {
  const { serverUrl } = useAppState();
  const tenant = useTenant();

  const [distribution, setDistribution] = useState<DurationDistribution>();
  const [isLoading, setIsLoading] = useState(false);
  const abortControllerRef = useRef(new AbortController());

  useEffect(() => () => abortControllerRef.current.abort(), []);

  useEffect(() => {
    if (!operation) return;

    abortControllerRef.current.abort();
    const controller = new AbortController();
    abortControllerRef.current = controller;
    const { signal } = controller;

    // eslint-disable-next-line @eslint-react/set-state-in-effect -- fetch below resolves with a fresh distribution (or clears it on error/abort); reacts only to operation/period identity changing
    setIsLoading(true);
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- see comment above
    setDistribution(undefined);

    (async () => {
      try {
        const url = getLogsqlQueryUrl(serverUrl);
        const startIso = dayjs(Number(periodStart / 1_000_000n)).toISOString();
        const endIso = dayjs(Number(periodEnd / 1_000_000n)).toISOString();

        const runQuery = async (query: string) => {
          const response = await fetch(url, {
            signal,
            method: "POST",
            headers: {
              ...tenant,
              Accept: "application/stream+json",
            },
            body: new URLSearchParams({ query, start: startIso, end: endIso }),
          });
          const text = await response.text();
          if (!response.ok) return [];
          return text.split("\n").map(parseLineToJSON).filter(Boolean);
        };

        const rangeRows = await runQuery(buildDurationRangeQuery(operation)) as DurationRangeStatsRow[];
        const range = parseDurationRange(rangeRows);
        if (!range) return;

        const histogramRows = await runQuery(
          buildDurationHistogramQuery(operation, range.minDurationNs, range.maxDurationNs)
        ) as DurationHistogramStatsRow[];
        setDistribution(parseDurationHistogram(histogramRows, range));
      } catch (e) {
        if (e instanceof Error && e.name !== "AbortError") console.error(e);
      } finally {
        if (abortControllerRef.current === controller) setIsLoading(false);
      }
    })();
  }, [operation, periodStart, periodEnd, serverUrl, tenant]);

  return { distribution, isLoading };
}
