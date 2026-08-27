import dayjs from "dayjs";
import { HEATMAP_DURATION_BAND_BOUNDARIES_US, HEATMAP_DURATION_BUCKETS, HEATMAP_TIME_BUCKETS } from "./constants";
import { computeHeatmapTimeStepNs } from "./computeHeatmapGrid";

export interface HeatmapGrid {
  /** counts[col][row] = number of traces in that (time-bucket, duration-band) cell */
  counts: number[][];
  /** errors[col][row] = number of error-status spans in that (time-bucket, duration-band) cell */
  errors: number[][];
  /** highest single-cell count, used to scale color intensity */
  maxCount: number;
}

export interface HeatmapStatsRow {
  _time: string;
  [bandField: string]: string;
}

export function makeEmptyGrid(): HeatmapGrid {
  return {
    counts: Array.from({ length: HEATMAP_TIME_BUCKETS }, () => new Array(HEATMAP_DURATION_BUCKETS).fill(0)),
    errors: Array.from({ length: HEATMAP_TIME_BUCKETS }, () => new Array(HEATMAP_DURATION_BUCKETS).fill(0)),
    maxCount: 0,
  };
}

// OTel status code, as encoded by the LogsQL `status_code` field: 0=UNSET, 1=OK, 2=ERROR.
const ERROR_STATUS_CODE = "2";

// duration is stored in nanoseconds; band boundaries are defined in microseconds for
// display, so convert once when building the query.
function buildDurationBandClauses(): string[] {
  const boundariesNs = HEATMAP_DURATION_BAND_BOUNDARIES_US.map(us => us * 1000);
  const bandConditions = boundariesNs.map((boundary, i) => {
    if (i === 0) return `duration:<${boundary}`;
    return `duration:>=${boundariesNs[i - 1]} AND duration:<${boundary}`;
  });
  bandConditions.push(`duration:>=${boundariesNs[boundariesNs.length - 1]}`);

  const counts = bandConditions.map((cond, i) => `count() if (${cond}) b${i}`);
  const errors = bandConditions.map((cond, i) => `count() if (${cond} AND status_code:="${ERROR_STATUS_CODE}") e${i}`);
  return [...counts, ...errors];
}

// Collapses to one row per trace_id first, so a trace with more than one matching span
// isn't plotted more than once.
const PER_TRACE_ROW_PIPE =
  "stats by (trace_id) min(_time) as _time, min(duration) as duration, max(status_code) as status_code";

export function buildHeatmapStatsQuery(filterQuery: string, startNs: bigint, endNs: bigint): string {
  const stepNs = computeHeatmapTimeStepNs(startNs, endNs, HEATMAP_TIME_BUCKETS);
  return `${filterQuery.trim()} | ${PER_TRACE_ROW_PIPE}` +
    ` | stats by (_time:${stepNs}ns) ${buildDurationBandClauses().join(", ")}`;
}

export function parseHeatmapRows(rows: HeatmapStatsRow[], periodStartNs: bigint, periodEndNs: bigint): HeatmapGrid {
  const grid = makeEmptyGrid();
  if (periodEndNs <= periodStartNs) return grid;
  const stepNs = computeHeatmapTimeStepNs(periodStartNs, periodEndNs, HEATMAP_TIME_BUCKETS);

  rows.forEach(row => {
    const timeMs = dayjs(row._time).valueOf();
    if (!Number.isFinite(timeMs)) return;
    // Uses the exact same integer stepNs the server was asked to bucket by, so this always
    // lands in the same column that columnToTimeRangeNs will reconstruct for a selection
    // covering it - see that function's comment for why this matters.
    const timeNs = BigInt(Math.round(timeMs)) * 1_000_000n;
    const colBig = (timeNs - periodStartNs) / stepNs;
    const col = Math.min(HEATMAP_TIME_BUCKETS - 1, Math.max(0, Number(colBig)));

    for (let band = 0; band < HEATMAP_DURATION_BUCKETS; band++) {
      const count = Number(row[`b${band}`]);
      if (Number.isFinite(count) && count > 0) {
        const next = grid.counts[col][band] + count;
        grid.counts[col][band] = next;
        if (next > grid.maxCount) grid.maxCount = next;
      }

      const errorCount = Number(row[`e${band}`]);
      if (Number.isFinite(errorCount) && errorCount > 0) {
        grid.errors[col][band] += errorCount;
      }
    }
  });

  return grid;
}
