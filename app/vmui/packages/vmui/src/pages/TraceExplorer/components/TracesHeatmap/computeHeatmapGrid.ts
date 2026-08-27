import { formatDurationCompact } from "../../../../utils/date";
import { Microseconds } from "../../types";

/** [lowUs, highUs] duration range that band `row` covers, given the fixed ascending
 * boundaries from HEATMAP_DURATION_BAND_BOUNDARIES_US. Row 0 is open-ended below
 * (lowUs = 0) and the last row is open-ended above (highUs = Infinity). */
export function rowToDurationRangeUs(row: number, boundariesUs: number[]): [number, number] {
  const low = row === 0 ? 0 : boundariesUs[row - 1];
  const high = row === boundariesUs.length ? Infinity : boundariesUs[row];
  return [low, high];
}

/** Formats a [lowUs, highUs] duration band (as returned by rowToDurationRangeUs) into a
 * single display string, collapsing the open-ended 0/Infinity edges into "< x"/"≥ x". */
export function formatDurationRange(lowUs: number, highUs: number): string {
  if (highUs === Infinity) return `≥ ${formatDurationCompact(lowUs as Microseconds)}`;
  if (lowUs === 0) return `< ${formatDurationCompact(highUs as Microseconds)}`;
  return `${formatDurationCompact(lowUs as Microseconds)} – ${formatDurationCompact(highUs as Microseconds)}`;
}

/** Inverse of rowToDurationRangeUs: which band a single duration falls into, matching the
 * same `duration:>=X AND duration:<Y` conditions buildHeatmapStatsQuery sends the server. */
export function durationUsToRow(durationUs: number, boundariesUs: number[]): number {
  const row = boundariesUs.findIndex(boundary => durationUs < boundary);
  return row === -1 ? boundariesUs.length : row;
}

/** Inverse of columnToTimeRangeUs: which evenly-spaced column a single point in time falls
 * into, clamped to the grid's bounds. Approximate (floating-point microseconds) - fine for
 * a display highlight, not for reconstructing a selection into a query. */
export function timeUsToColumn(
  timeUs: number,
  periodStartUs: number,
  periodEndUs: number,
  columns: number,
): number {
  const cellSpan = (periodEndUs - periodStartUs) / columns;
  const col = Math.floor((timeUs - periodStartUs) / cellSpan);
  return Math.min(columns - 1, Math.max(0, col));
}

/** [startUs, endUs] time range that bucket `col` spans, matching the evenly-spaced
 * columns requested from the server-side aggregation. Approximate (floating-point
 * microseconds) - fine for display (tooltips, axis labels), but not for reconstructing a
 * selection into a query: use columnToTimeRangeNs for that. */
export function columnToTimeRangeUs(
  col: number,
  periodStartUs: number,
  periodEndUs: number,
  columns: number,
): [number, number] {
  const cellSpan = (periodEndUs - periodStartUs) / columns;
  return [periodStartUs + col * cellSpan, periodStartUs + (col + 1) * cellSpan];
}

/** Same step math buildHeatmapStatsQuery sends the server (integer nanoseconds, exact). */
export function computeHeatmapTimeStepNs(periodStartNs: bigint, periodEndNs: bigint, columns: number): bigint {
  const interval = (periodEndNs - periodStartNs) / BigInt(columns);
  return interval > 0n ? interval : 1n;
}

/** [startNs, endNs) time range that bucket `col` spans, in exact integer nanoseconds.
 *
 * This is the exact inverse of how a row's `_time` gets assigned to a column (see
 * parseHeatmapRows): both sides floor-divide by the same integer stepNs relative to
 * periodStartNs, so a selection built from this always reconstructs precisely the set of
 * columns it visually covers - no floating-point rounding can drop a boundary row.
 *
 * This does NOT reproduce the server's own bucket edges (VictoriaLogs aligns `_time:step`
 * buckets to absolute Unix epoch, not to periodStartNs), but it doesn't need to: what
 * matters for a selection is only that it's self-consistent with the column a row was
 * counted into, which it is by construction.
 */
export function columnToTimeRangeNs(
  col: number,
  periodStartNs: bigint,
  periodEndNs: bigint,
  columns: number,
): [bigint, bigint] {
  const stepNs = computeHeatmapTimeStepNs(periodStartNs, periodEndNs, columns);
  const low = periodStartNs + BigInt(col) * stepNs;
  const high = col === columns - 1 ? periodEndNs : periodStartNs + BigInt(col + 1) * stepNs;
  return [low, high];
}
