import { HEATMAP_DURATION_BUCKETS, HEATMAP_TIME_BUCKETS } from "./constants";
import { computeHeatmapTimeStepNs } from "./computeHeatmapGrid";
import { splitFiltersAndPipes } from "../../utils";
import { getNanoTimestamp } from "../../../../utils/time";
import { ERROR_STATUS_CODE } from "../../hooks/useLogsqlTracesSearch";

export interface HeatmapGrid {
  /** counts[col][row] = number of traces in that (time-bucket, duration-band) cell */
  counts: number[][];
  /** errors[col][row] = number of traces with an error span in that (time-bucket, duration-band) cell */
  errors: number[][];
  /** highest single-cell count, used to scale color intensity */
  maxCount: number;
}

export interface HeatmapStatsRow {
  _time: string;
  band: string;
  count: string;
}

// One row per trace, written by vtinsert: _time is the trace start and duration is the whole trace duration (ns).
const TRACE_INDEX_FILTER = "{trace_id_idx_stream!=\"\"}";

// Index durations are stored in nanoseconds, band boundaries are defined in microseconds.
const NS_PER_US = 1000;

// Every duration below the first boundary (1us) lands in band 0. Raising zero durations to this value keeps ln() defined.
const MIN_DURATION_US = 0.5;

// Band boundaries repeat every power of 10 (see buildDurationBandBoundariesUs in constants.ts).
const DECADE_BASE = 10;

// Each decade is split into two bands at 1x and 3x of its power of 10.
const BANDS_PER_DECADE = 2;
const MID_DECADE_MULTIPLIER = 3;

// Band 0 holds durations below 1us, so the decade starting at 1us (10^0) begins at band 1.
const FIRST_DECADE_BAND = 1;

// Compensates floating-point error of ln(), so an exact boundary such as 1000us lands in the upper band.
// Kept as a string because LogsQL math doesn't parse exponent notation like 1e-9.
const FLOAT_EPSILON = "0.000000001";

const MAX_BAND = HEATMAP_DURATION_BUCKETS - 1;

// Maps duration to its HEATMAP_DURATION_BAND_BOUNDARIES_US row (1us, 3us, 10us, ..., 100s) in a single pass:
// band = BANDS_PER_DECADE * decade + (1 if the duration is at or above MID_DECADE_MULTIPLIER * 10^decade) + FIRST_DECADE_BAND.
const DURATION_BAND_PIPE = `math max(duration/${NS_PER_US}, ${MIN_DURATION_US}) as us, `
  + `floor(ln(us)/ln(${DECADE_BASE}) + ${FLOAT_EPSILON}) as decade, `
  + `max(min(${BANDS_PER_DECADE}*decade `
  + `+ min(floor(us/(${DECADE_BASE}^decade)/${MID_DECADE_MULTIPLIER} + ${FLOAT_EPSILON}), ${BANDS_PER_DECADE - 1}) `
  + `+ ${FIRST_DECADE_BAND}, ${MAX_BAND}), 0) as band`;

export function makeEmptyMatrix(): number[][] {
  return Array.from({ length: HEATMAP_TIME_BUCKETS }, () => new Array(HEATMAP_DURATION_BUCKETS).fill(0));
}

export function getMaxCount(counts: number[][]): number {
  return counts.reduce((max, column) => column.reduce((colMax, count) => Math.max(colMax, count), max), 0);
}

// Pipes are dropped: the filter is applied to spans inside a join, and extra filters can't be sent as the
// global `extra_filters` arg since the trace index rows don't carry span fields.
function buildSpanFilter(filterQuery: string, extraFilters: string[]): string {
  const { filters } = splitFiltersAndPipes(filterQuery);
  return [filters, ...extraFilters]
    .filter(filter => filter && filter !== "*")
    .map(filter => `(${filter})`)
    .join(" AND ");
}

function joinTraces(spanFilter: string): string {
  return `join by (trace_id_idx) (${spanFilter} | uniq by (trace_id) | rename trace_id as trace_id_idx) inner`;
}

export function buildHeatmapTracesQuery(
  filterQuery: string, extraFilters: string[], startNs: bigint, endNs: bigint
): string {
  const spanFilter = buildSpanFilter(filterQuery, extraFilters);
  const stepNs = computeHeatmapTimeStepNs(startNs, endNs, HEATMAP_TIME_BUCKETS);
  const offsetNs = startNs % stepNs;
  return [
    TRACE_INDEX_FILTER,
    ...(spanFilter ? [joinTraces(spanFilter)] : []),
    DURATION_BAND_PIPE,
    `stats by (_time:${stepNs}ns offset ${offsetNs}ns, band) count() as count`,
  ].join(" | ");
}

export function buildHeatmapErrorsQuery(
  filterQuery: string, extraFilters: string[], startNs: bigint, endNs: bigint
): string {
  return buildHeatmapTracesQuery(filterQuery, [...extraFilters, `status_code:="${ERROR_STATUS_CODE}"`], startNs, endNs);
}

export function parseHeatmapRows(rows: HeatmapStatsRow[], periodStartNs: bigint, periodEndNs: bigint): number[][] {
  const matrix = makeEmptyMatrix();
  if (periodEndNs <= periodStartNs) return matrix;
  const stepNs = computeHeatmapTimeStepNs(periodStartNs, periodEndNs, HEATMAP_TIME_BUCKETS);

  rows.forEach(row => {
    const timeMs = Date.parse(row._time);
    const band = Number(row.band);
    const count = Number(row.count);
    if (!Number.isFinite(timeMs) || !Number.isInteger(band) || band < 0 || band >= HEATMAP_DURATION_BUCKETS) return;
    if (!Number.isFinite(count) || count <= 0) return;

    const timeNs = getNanoTimestamp(row._time, timeMs);
    const col = Math.min(HEATMAP_TIME_BUCKETS - 1, Math.max(0, Number((timeNs - periodStartNs) / stepNs)));
    matrix[col][band] += count;
  });

  return matrix;
}
