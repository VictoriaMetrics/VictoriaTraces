import { FC, useEffect, useMemo, useRef, useState } from "preact/compat";
import dayjs from "dayjs";
import classNames from "classnames";
import { useResizeObserver } from "../../../../hooks/useResizeObserver";
import LineLoader from "../../../../components/Main/LineLoader";
import ApiErrorAlert from "../ApiErrorAlert";
import { useAppState } from "../../../../state/common/StateContext";
import { formatDatetime, formatDurationCompact } from "../../../../utils/date";
import { Microseconds } from "../../types";
import {
  columnToTimeRangeNs,
  columnToTimeRangeUs,
  durationUsToRow,
  formatDurationRange,
  rowToDurationRangeUs,
  timeUsToColumn,
} from "./computeHeatmapGrid";
import { HeatmapGrid } from "./useHeatmapAggregation";
import renderIntoCanvas from "./render-into-canvas";
import {
  HEATMAP_AXIS_FALLBACK_COLOR,
  HEATMAP_CANVAS_HEIGHT,
  HEATMAP_DURATION_BAND_BOUNDARIES_US,
  HEATMAP_DURATION_BUCKETS,
  HEATMAP_FALLBACK_COLOR,
  HEATMAP_MARKER_COLOR,
  HEATMAP_TIME_BUCKETS,
  HEATMAP_X_AXIS_HEIGHT,
  HEATMAP_X_AXIS_TICKS,
  HEATMAP_Y_AXIS_TICKS,
} from "./constants";

import "./TracesHeatmap.scss";

export interface HeatmapSelectionRange {
  timeLowNs: bigint;
  timeHighNs: bigint;
  durationLowUs: number;
  durationHighUs: number;
}

export interface HighlightedTrace {
  startTimeUs: number;
  durationUs: number;
}

export interface TracesHeatmapProps {
  grid: HeatmapGrid;
  isLoading: boolean;
  error?: string;
  periodStart: bigint;
  periodEnd: bigint;
  minDurationUs?: number;
  maxDurationUs?: number;
  highlightedTrace?: HighlightedTrace | null;
  onSelectionChange?: (selection: HeatmapSelectionRange | null) => void;
  onCommitSelection?: (selection: HeatmapSelectionRange) => void;
}

interface HoverCell {
  x: number;
  y: number;
  canvasWidth: number;
  count: number;
  errorCount: number;
  durationLowUs: number;
  durationHighUs: number;
  timeLowUs: number;
  timeHighUs: number;
}

interface Bucket {
  col: number;
  row: number;
}

interface BucketRect {
  colMin: number;
  colMax: number;
  rowMin: number;
  rowMax: number;
}

const SECOND_US = 1_000_000;
const MINUTE_US = 60_000_000;
const HOUR_US = 3_600_000_000;
const DAY_US = HOUR_US * 24;
const MONTH_US = DAY_US * 30;
const YEAR_US = DAY_US * 365;

interface AxisTick {
  date: string | null;
  time: string;
}

function formatAxisTick(us: number, prevUs: number | null, incrementUs: number): AxisTick {
  const d = dayjs(us / 1000);
  const prevD = prevUs === null ? null : dayjs(prevUs / 1000);
  const crossedYear = !prevD || d.year() !== prevD.year();
  const crossedDay = crossedYear || !d.isSame(prevD, "day");

  if (incrementUs >= YEAR_US) return { date: null, time: d.format("YYYY") };
  if (incrementUs >= MONTH_US) return { date: null, time: crossedYear ? d.format("MMM YYYY") : d.format("MMM") };
  if (incrementUs >= DAY_US) return { date: null, time: crossedYear ? d.format("YYYY-MM-DD") : d.format("MM-DD") };

  if (incrementUs >= SECOND_US) {
    const time = d.format(incrementUs >= MINUTE_US ? "HH:mm" : "HH:mm:ss");
    if (crossedYear) return { date: d.format("YYYY-MM-DD"), time };
    if (crossedDay) return { date: d.format("MM-DD"), time };
    return { date: null, time };
  }

  if (crossedYear) return { date: d.format("YYYY-MM-DD"), time: d.format("HH:mm:ss.SSS") };
  if (crossedDay) return { date: d.format("MM-DD"), time: d.format("HH:mm:ss.SSS") };
  return { date: null, time: d.format("ss.SSS") };
}

function boundsFromBuckets(a: Bucket, b: Bucket): BucketRect {
  return {
    colMin: Math.min(a.col, b.col),
    colMax: Math.max(a.col, b.col),
    rowMin: Math.min(a.row, b.row),
    rowMax: Math.max(a.row, b.row),
  };
}

function isBucketInRect(bucket: Bucket, rect: BucketRect): boolean {
  return bucket.col >= rect.colMin && bucket.col <= rect.colMax &&
    bucket.row >= rect.rowMin && bucket.row <= rect.rowMax;
}

function computeVisibleRowRange(minDurationUs: number, maxDurationUs: number): [number, number] {
  const boundaries = HEATMAP_DURATION_BAND_BOUNDARIES_US;
  let rowMin = 0;
  let rowMax = HEATMAP_DURATION_BUCKETS - 1;

  if (minDurationUs > 0) {
    while (rowMin < rowMax && rowToDurationRangeUs(rowMin, boundaries)[1] <= minDurationUs) rowMin++;
  }
  if (Number.isFinite(maxDurationUs)) {
    while (rowMax > rowMin && rowToDurationRangeUs(rowMax, boundaries)[0] >= maxDurationUs) rowMax--;
  }

  return [rowMin, rowMax];
}

const TracesHeatmap: FC<TracesHeatmapProps> = ({
  grid,
  isLoading,
  error,
  periodStart,
  periodEnd,
  minDurationUs = 0,
  maxDurationUs = Infinity,
  highlightedTrace,
  onSelectionChange,
  onCommitSelection,
}) => {
  const wrapperRef = useRef<HTMLDivElement | null>(null);
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const { isDarkTheme } = useAppState();
  const { width: containerWidth } = useResizeObserver({ ref: wrapperRef });
  const [hover, setHover] = useState<HoverCell | null>(null);
  const [hoveredBucket, setHoveredBucket] = useState<Bucket | null>(null);
  const [dragStart, setDragStart] = useState<Bucket | null>(null);
  const [dragCurrent, setDragCurrent] = useState<Bucket | null>(null);
  const [selection, setSelection] = useState<BucketRect | null>(null);
  const [yAxisWidth, setYAxisWidth] = useState(40);
  const [pointerInPlot, setPointerInPlot] = useState(false);
  const periodStartUs = Number(periodStart / 1000n);
  const periodEndUs = Number(periodEnd / 1000n);

  const [rowWindowMin, rowWindowMax] = useMemo(
    () => computeVisibleRowRange(minDurationUs, maxDurationUs),
    [minDurationUs, maxDurationUs]
  );
  const visibleRows = rowWindowMax - rowWindowMin + 1;

  const visibleCounts = useMemo(
    () => grid.counts.map(colCounts => colCounts.slice(rowWindowMin, rowWindowMax + 1)),
    [grid, rowWindowMin, rowWindowMax]
  );
  const visibleErrors = useMemo(
    () => grid.errors.map(colErrors => colErrors.slice(rowWindowMin, rowWindowMax + 1)),
    [grid, rowWindowMin, rowWindowMax]
  );

  const xTicks = useMemo(() => {
    const spanUs = periodEndUs - periodStartUs;
    const incrementUs = spanUs / (HEATMAP_X_AXIS_TICKS - 1);
    let prevUs: number | null = null;

    return Array.from({ length: HEATMAP_X_AXIS_TICKS }, (_, i) => {
      const fraction = i / (HEATMAP_X_AXIS_TICKS - 1);
      const tickUs = periodStartUs + fraction * spanUs;
      const tick = formatAxisTick(tickUs, prevUs, incrementUs);
      prevUs = tickUs;
      return { fraction, ...tick };
    });
  }, [periodStartUs, periodEndUs]);

  const [visibleDurationLowUs, visibleDurationHighUs] = useMemo(() => {
    const [lowRaw] = rowToDurationRangeUs(rowWindowMin, HEATMAP_DURATION_BAND_BOUNDARIES_US);
    const [, highRaw] = rowToDurationRangeUs(rowWindowMax, HEATMAP_DURATION_BAND_BOUNDARIES_US);
    const low = lowRaw > 0 ? lowRaw : HEATMAP_DURATION_BAND_BOUNDARIES_US[0] / 10;
    const high = Number.isFinite(highRaw)
      ? highRaw
      : HEATMAP_DURATION_BAND_BOUNDARIES_US[HEATMAP_DURATION_BAND_BOUNDARIES_US.length - 1] * 10;
    return [low, high];
  }, [rowWindowMin, rowWindowMax]);

  const yTicks = useMemo(() => {
    const logLow = Math.log10(visibleDurationLowUs);
    const logHigh = Math.log10(visibleDurationHighUs);
    return Array.from({ length: HEATMAP_Y_AXIS_TICKS }, (_, i) => {
      const fraction = i / (HEATMAP_Y_AXIS_TICKS - 1);
      const value = 10 ** (logLow + fraction * (logHigh - logLow));
      return formatDurationCompact(value as Microseconds);
    }).reverse();
  }, [visibleDurationLowUs, visibleDurationHighUs]);

  // Null when the highlighted trace's own duration band is filtered out of the currently
  // visible row window - nothing meaningful to point at in that case.
  const highlightedBucket = useMemo((): Bucket | null => {
    if (!highlightedTrace) return null;
    const row = durationUsToRow(highlightedTrace.durationUs, HEATMAP_DURATION_BAND_BOUNDARIES_US);
    if (row < rowWindowMin || row > rowWindowMax) return null;
    const col = timeUsToColumn(highlightedTrace.startTimeUs, periodStartUs, periodEndUs, HEATMAP_TIME_BUCKETS);
    return { col, row };
  }, [highlightedTrace, rowWindowMin, rowWindowMax, periodStartUs, periodEndUs]);

  useEffect(() => {
    if (!canvasRef.current) return;
    const style = window.getComputedStyle(document.documentElement);
    const accentColor = style.getPropertyValue("--color-primary").trim();
    const axisColor = style.getPropertyValue("--color-text-secondary").trim();
    const { yAxisWidth: nextYAxisWidth } = renderIntoCanvas({
      canvas: canvasRef.current,
      counts: visibleCounts,
      errors: visibleErrors,
      maxCount: grid.maxCount,
      cellColor: accentColor || HEATMAP_FALLBACK_COLOR,
      markerColor: HEATMAP_MARKER_COLOR,
      axisColor: axisColor || HEATMAP_AXIS_FALLBACK_COLOR,
      plotHeight: HEATMAP_CANVAS_HEIGHT,
      xAxisHeight: HEATMAP_X_AXIS_HEIGHT,
      containerWidth,
      xTicks,
      yTicks,
    });
    setYAxisWidth(nextYAxisWidth);
  }, [visibleCounts, visibleErrors, grid.maxCount, isDarkTheme, containerWidth, xTicks, yTicks]);

  useEffect(() => {
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- intentional reset of stale selection when the underlying grid/window changes, see comment above
    setSelection(null);
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- intentional reset of stale selection when the underlying grid/window changes, see comment above
    setDragStart(null);
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- intentional reset of stale selection when the underlying grid/window changes, see comment above
    setDragCurrent(null);
  }, [grid, rowWindowMin, rowWindowMax]);

  const getBucketAt = (e: MouseEvent): Bucket | null => {
    const canvas = canvasRef.current;
    if (!canvas) return null;
    const rect = canvas.getBoundingClientRect();
    if (!rect.width || !rect.height) return null;

    const x = e.clientX - rect.left - yAxisWidth;
    const y = e.clientY - rect.top;
    if (x < 0 || y < 0 || y > HEATMAP_CANVAS_HEIGHT) return null;

    const plotWidth = rect.width - yAxisWidth;
    const col = Math.min(HEATMAP_TIME_BUCKETS - 1, Math.max(0, Math.floor(x / (plotWidth / HEATMAP_TIME_BUCKETS))));
    const rowFromTop = Math.min(
      visibleRows - 1,
      Math.max(0, Math.floor(y / (HEATMAP_CANVAS_HEIGHT / visibleRows)))
    );
    return { col, row: rowWindowMin + (visibleRows - 1 - rowFromTop) };
  };

  const bucketRectToSelection = (rect: BucketRect): HeatmapSelectionRange => {
    const [, timeHighNs] = columnToTimeRangeNs(rect.colMax, periodStart, periodEnd, HEATMAP_TIME_BUCKETS);
    const [timeLowNs] = columnToTimeRangeNs(rect.colMin, periodStart, periodEnd, HEATMAP_TIME_BUCKETS);
    const [durationLowUs] = rowToDurationRangeUs(rect.rowMin, HEATMAP_DURATION_BAND_BOUNDARIES_US);
    const [, durationHighUs] = rowToDurationRangeUs(rect.rowMax, HEATMAP_DURATION_BAND_BOUNDARIES_US);
    return { timeLowNs, timeHighNs, durationLowUs, durationHighUs };
  };

  const handleMouseMove = (e: MouseEvent) => {
    const bucket = getBucketAt(e);
    if (!bucket) {
      setPointerInPlot(false);
      setHoveredBucket(null);
      setHover(null);
      return;
    }
    setPointerInPlot(true);
    setHoveredBucket(bucket);

    if (dragStart) {
      setDragCurrent(bucket);
      setHover(null);
      return;
    }

    const canvas = canvasRef.current;
    const rect = canvas?.getBoundingClientRect();
    if (!rect) return;

    const count = grid.counts[bucket.col]?.[bucket.row] ?? 0;
    if (count <= 0) {
      setHover(null);
      return;
    }
    const errorCount = grid.errors[bucket.col]?.[bucket.row] ?? 0;

    const [durationLowUs, durationHighUs] = rowToDurationRangeUs(bucket.row, HEATMAP_DURATION_BAND_BOUNDARIES_US);
    const [timeLowUs, timeHighUs] = columnToTimeRangeUs(bucket.col, periodStartUs, periodEndUs, HEATMAP_TIME_BUCKETS);

    setHover({
      x: e.clientX - rect.left - yAxisWidth,
      y: e.clientY - rect.top,
      canvasWidth: rect.width - yAxisWidth,
      count,
      errorCount,
      durationLowUs,
      durationHighUs,
      timeLowUs,
      timeHighUs,
    });
  };

  const handleMouseDown = (e: MouseEvent) => {
    const bucket = getBucketAt(e);
    if (!bucket) return;
    setDragStart(bucket);
    setDragCurrent(bucket);
    setHover(null);
  };

  const handleMouseUp = (e: MouseEvent) => {
    const start = dragStart;
    setDragStart(null);
    setDragCurrent(null);
    if (!start) return;

    const bucket = getBucketAt(e) ?? start;
    const isRealDrag = bucket.col !== start.col || bucket.row !== start.row;

    if (isRealDrag) {
      const rect = boundsFromBuckets(start, bucket);
      setSelection(rect);
      onSelectionChange?.(bucketRectToSelection(rect));
      return;
    }

    if (selection && isBucketInRect(bucket, selection)) {
      onCommitSelection?.(bucketRectToSelection(selection));
      setSelection(null);
      onSelectionChange?.(null);
      return;
    }

    if (selection) {
      setSelection(null);
      onSelectionChange?.(null);
    }

    if ((grid.counts[bucket.col]?.[bucket.row] ?? 0) > 0) {
      const rect = boundsFromBuckets(bucket, bucket);
      setSelection(rect);
      onSelectionChange?.(bucketRectToSelection(rect));
    }
  };

  const handleMouseLeave = () => {
    setHover(null);
    setHoveredBucket(null);
    setPointerInPlot(false);
    setDragStart(null);
    setDragCurrent(null);
  };

  const tooltipStyle = hover ? {
    ...(hover.x < hover.canvasWidth / 2
      ? { left: `${hover.x + 12}px` }
      : { right: `${hover.canvasWidth - hover.x + 12}px` }),
    ...(hover.y < HEATMAP_CANVAS_HEIGHT / 2
      ? { top: `${hover.y + 12}px` }
      : { bottom: `${HEATMAP_CANVAS_HEIGHT - hover.y + 12}px` }),
  } : undefined;

  const displayRect = dragStart && dragCurrent ? boundsFromBuckets(dragStart, dragCurrent) : selection;
  const isDraggingSelection = !!(dragStart && dragCurrent);
  const isHoveringSelection = !!(selection && hoveredBucket && isBucketInRect(hoveredBucket, selection));

  const selectionStyle = displayRect ? {
    left: `${(displayRect.colMin / HEATMAP_TIME_BUCKETS) * 100}%`,
    width: `${((displayRect.colMax - displayRect.colMin + 1) / HEATMAP_TIME_BUCKETS) * 100}%`,
    top: `${((visibleRows - 1 - (displayRect.rowMax - rowWindowMin)) / visibleRows) * 100}%`,
    height: `${((displayRect.rowMax - displayRect.rowMin + 1) / visibleRows) * 100}%`,
  } : undefined;

  const highlightStyle = highlightedBucket ? {
    left: `${(highlightedBucket.col / HEATMAP_TIME_BUCKETS) * 100}%`,
    width: `${(1 / HEATMAP_TIME_BUCKETS) * 100}%`,
    top: `${((visibleRows - 1 - (highlightedBucket.row - rowWindowMin)) / visibleRows) * 100}%`,
    height: `${(1 / visibleRows) * 100}%`,
  } : undefined;

  if (error) {
    return (
      <div className="vm-traces-heatmap">
        <ApiErrorAlert
          error={error}
          className="vm-traces-heatmap__error"
        />
      </div>
    );
  }

  return (
    <div className={classNames("vm-traces-heatmap", { "vm-traces-heatmap_loading": isLoading })}>
      {isLoading && <LineLoader/>}
      <div
        className="vm-traces-heatmap__canvas-wrapper"
        ref={wrapperRef}
        style={{ height: HEATMAP_CANVAS_HEIGHT + HEATMAP_X_AXIS_HEIGHT }}
      >
        <canvas
          className="vm-traces-heatmap__canvas"
          ref={canvasRef}
          style={{ cursor: !pointerInPlot ? "default" : isHoveringSelection ? "pointer" : "crosshair" }}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseLeave}
        />
        <div
          className="vm-traces-heatmap__plot-bounds"
          style={{ left: yAxisWidth, right: 0, height: HEATMAP_CANVAS_HEIGHT }}
        >
          {displayRect && (
            <div
              className={classNames("vm-traces-heatmap__selection", {
                "vm-traces-heatmap__selection_dragging": isDraggingSelection,
              })}
              style={selectionStyle}
            />
          )}
          {highlightStyle && (
            <div
              className="vm-traces-heatmap__highlight"
              style={highlightStyle}
            />
          )}
          {hover && (
            <div
              className="vm-traces-heatmap__tooltip"
              style={tooltipStyle}
            >
              <div className="vm-traces-heatmap__tooltip-row">
                <span className="vm-traces-heatmap__tooltip-label">Traces</span>
                <span className="vm-traces-heatmap__tooltip-value">{hover.count}</span>
              </div>
              {hover.errorCount > 0 && (
                <div className="vm-traces-heatmap__tooltip-row">
                  <span className="vm-traces-heatmap__tooltip-label">Errors</span>
                  <span className="vm-traces-heatmap__tooltip-value vm-traces-heatmap__tooltip-value_error">
                    {hover.errorCount}
                  </span>
                </div>
              )}
              <div className="vm-traces-heatmap__tooltip-row">
                <span className="vm-traces-heatmap__tooltip-label">Duration</span>
                <span className="vm-traces-heatmap__tooltip-value">
                  {formatDurationRange(hover.durationLowUs, hover.durationHighUs)}
                </span>
              </div>
              <div className="vm-traces-heatmap__tooltip-row">
                <span className="vm-traces-heatmap__tooltip-label">Time</span>
                <span className="vm-traces-heatmap__tooltip-value">
                  {formatDatetime((hover.timeLowUs + hover.timeHighUs) / 2)}
                </span>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default TracesHeatmap;
