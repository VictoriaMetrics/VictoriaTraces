import { AlignedData } from "uplot";
import { LogHits } from "../../../../api/types";
import { HITS_BARS_VIEW, HITS_FIELDS_LIMIT } from "../../../../constants/logs";
import { NANOSECONDS_PER_MILLISECOND, nanosToIsoString } from "../../../../utils/time";
import { SERVICE_FIELD, splitFiltersAndPipes } from "../../utils";
import { isEmptyObject } from "../../../../utils/object";

export type RawLogHits = Omit<LogHits, "_isOther">;

// Trace index rows (`trace_id_idx_stream`) carry no span fields, so this filter drops them from the span counts.
const SERVICE_FILTER = `"${SERVICE_FIELD}":*`;

const MS_PER_SECOND = 1000;

// Hits are counted on spans only, so user pipes are dropped: the hits API returns nothing for queries with pipes.
export function buildServiceHitsQuery(filterQuery: string, extraClauses: string[]): string {
  const { filters } = splitFiltersAndPipes(filterQuery);
  return [filters, ...extraClauses]
    .filter(filter => filter && filter !== "*")
    .map(filter => `(${filter})`)
    .concat(SERVICE_FILTER)
    .join(" AND ");
}

// Splits the period into HITS_BARS_VIEW buckets; the server rejects sub-millisecond steps.
export function getHitsStepNs(startNs: bigint, endNs: bigint): bigint {
  const bars = BigInt(HITS_BARS_VIEW);
  const step = (endNs - startNs + bars - 1n) / bars;
  return step < NANOSECONDS_PER_MILLISECOND ? NANOSECONDS_PER_MILLISECOND : step;
}

export function buildServiceHitsParams(
  filterQuery: string, startNs: bigint, endNs: bigint, extraClauses: string[]
): URLSearchParams {
  return new URLSearchParams({
    query: buildServiceHitsQuery(filterQuery, extraClauses),
    step: `${getHitsStepNs(startNs, endNs)}ns`,
    start: nanosToIsoString(startNs),
    end: nanosToIsoString(endNs),
    field: SERVICE_FIELD,
    fields_limit: `${HITS_FIELDS_LIMIT}`,
  });
}

// "Other" aggregates series beyond fields_limit; it goes first so its bars stay behind the named series.
export function normalizeHits(hits: RawLogHits[]): LogHits[] {
  return hits
    .map(hit => ({ ...hit, _isOther: isEmptyObject(hit.fields) }))
    .sort((a, b) => {
      if (a._isOther !== b._isOther) return a._isOther ? -1 : 1;
      return b.total - a.total;
    });
}

// Server buckets are aligned to multiples of the step since the epoch, so the x axis is rebuilt the same way
// and every bucket in the period gets a point even when the server returned no rows for it.
export function hitsToAlignedData(hits: LogHits[], startNs: bigint, endNs: bigint): AlignedData {
  if (!hits.length) return [[], []];

  const stepNs = getHitsStepNs(startNs, endNs);
  const bucketsMs: number[] = [];
  for (let bucket = (startNs / stepNs) * stepNs; bucket <= endNs; bucket += stepNs) {
    bucketsMs.push(Number(bucket / NANOSECONDS_PER_MILLISECOND));
  }

  const series = hits.map(hit => {
    const valueByBucketMs = new Map<number, number>();
    hit.timestamps.forEach((ts, idx) => valueByBucketMs.set(Date.parse(ts), hit.values[idx]));
    return bucketsMs.map(ms => valueByBucketMs.get(ms) ?? null);
  });

  return [bucketsMs.map(ms => ms / MS_PER_SECOND), ...series];
}
