import { HEATMAP_AXIS_FONT } from "./constants";

const MIN_CELL_ALPHA = 0.08;
const MAX_CELL_ALPHA = 0.95;
const AXIS_PADDING = 6;
const AXIS_LINE_HEIGHT = 13;

function clamp(n: number, lo: number, hi: number) {
  return Math.max(lo, Math.min(hi, n));
}

interface XTick {
  fraction: number;
  date: string | null;
  time: string;
}

export interface RenderHeatmapParams {
  canvas: HTMLCanvasElement;
  counts: number[][];
  errors: number[][];
  maxCount: number;
  cellColor: string;
  markerColor: string;
  axisColor: string;
  plotHeight: number;
  xAxisHeight: number;
  containerWidth?: number;
  xTicks: XTick[];
  yTicks: string[];
}

export interface RenderHeatmapResult {
  /** The y-axis gutter width actually drawn, in css px - callers align HTML overlays
   * (selection box, tooltip) to the same plot bounds using this. */
  yAxisWidth: number;
}

/**
 * Renders the heatmap - density cells, error markers, and both axes - into a single canvas,
 * matching how VictoriaLogs' uPlot charts draw their own axes (utils/uplot/axes.ts): tick
 * text is drawn with ctx.fillText rather than positioned HTML, in the same "10px Arial".
 *
 * Features:
 * - High-DPI (Retina) rendering using devicePixelRatio
 * - Transparent plot background (0-count cells draw nothing)
 * - Cells with error spans get a small triangle in their bottom-right corner, in
 *   `markerColor`, layered on top of the density fill so error-containing cells stay
 *   findable without hovering.
 *
 * cssWidth is taken from `containerWidth` rather than `canvas.clientWidth`: this
 * function pins canvas.style.width to a fixed px value below, so re-reading
 * clientWidth on a later call would just echo back that stale, self-imposed size
 * instead of the real layout width (see CanvasSpanGraph/render-into-canvas.ts).
 */
export default function renderIntoCanvas(params: RenderHeatmapParams): RenderHeatmapResult {
  const {
    canvas, counts, errors, maxCount, cellColor, markerColor, axisColor,
    plotHeight, xAxisHeight, containerWidth, xTicks, yTicks,
  } = params;

  const columns = counts.length;
  const rows = columns > 0 ? counts[0].length : 0;

  const totalCssWidth = Math.max(1, Math.floor(containerWidth || canvas.clientWidth || window.innerWidth));
  const totalCssHeight = plotHeight + xAxisHeight;
  const dpr = clamp(window.devicePixelRatio || 1, 1, 3);

  canvas.style.width = `${totalCssWidth}px`;
  canvas.style.height = `${totalCssHeight}px`;
  canvas.width = Math.floor(totalCssWidth * dpr);
  canvas.height = Math.floor(totalCssHeight * dpr);

  const ctx = canvas.getContext("2d", { alpha: true }) as CanvasRenderingContext2D | null;
  if (!ctx) return { yAxisWidth: 0 };

  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.imageSmoothingEnabled = false;
  ctx.clearRect(0, 0, totalCssWidth, totalCssHeight);

  ctx.font = HEATMAP_AXIS_FONT;
  ctx.fillStyle = axisColor;

  const yAxisWidth = Math.ceil(Math.max(0, ...yTicks.map(label => ctx.measureText(label).width))) + AXIS_PADDING;
  const plotWidth = Math.max(1, totalCssWidth - yAxisWidth);

  // Y-axis labels, evenly spaced top (longest duration) to bottom (shortest) - matches the
  // grid's own row-0-at-bottom convention. Top/bottom labels sit flush inside the plot edge
  // (baseline "top"/"bottom"); interior ones are centered on their gridline.
  ctx.textAlign = "right";
  yTicks.forEach((label, i) => {
    const y = (i / (yTicks.length - 1)) * plotHeight;
    ctx.textBaseline = i === 0 ? "top" : i === yTicks.length - 1 ? "bottom" : "middle";
    ctx.fillText(label, yAxisWidth - AXIS_PADDING, y);
  });

  // X-axis labels: time on the first line, an anchoring date (when present) on the second.
  // Both lines are drawn textAlign "center" on a shared centerX so a two-row tick's time and
  // date are centered on each other, not just sharing a left/right edge (they can differ in
  // width). The edge ticks still pin their outer edge to the tick position, same as before -
  // centerX is offset by half of whichever line is wider so that edge holds.
  ctx.textBaseline = "top";
  ctx.textAlign = "center";
  xTicks.forEach(({ fraction, date, time }, i) => {
    const anchorX = yAxisWidth + fraction * plotWidth;
    const align = i === 0 ? "left" : i === xTicks.length - 1 ? "right" : "center";
    const maxWidth = date ? Math.max(ctx.measureText(time).width, ctx.measureText(date).width) : 0;
    const centerX = align === "left" ? anchorX + maxWidth / 2
      : align === "right" ? anchorX - maxWidth / 2
        : anchorX;

    ctx.fillText(time, centerX, plotHeight + AXIS_PADDING);
    if (date) ctx.fillText(date, centerX, plotHeight + AXIS_PADDING + AXIS_LINE_HEIGHT);
  });

  if (!columns || !rows || maxCount <= 0) return { yAxisWidth };

  const cellWidth = plotWidth / columns;
  const cellHeight = plotHeight / rows;

  ctx.fillStyle = cellColor;

  for (let col = 0; col < columns; col++) {
    for (let row = 0; row < rows; row++) {
      const count = counts[col][row];
      if (count <= 0) continue;

      const intensity = clamp(count / maxCount, 0, 1);
      ctx.globalAlpha = MIN_CELL_ALPHA + intensity * (MAX_CELL_ALPHA - MIN_CELL_ALPHA);

      const x = yAxisWidth + col * cellWidth;
      // row 0 = shortest durations: draw at the bottom, longest durations at the top
      const y = plotHeight - (row + 1) * cellHeight;
      ctx.fillRect(x, y, Math.ceil(cellWidth), Math.ceil(cellHeight));
    }
  }

  ctx.globalAlpha = 1;
  ctx.fillStyle = markerColor;
  const triangleSize = clamp(Math.min(cellWidth, cellHeight), 3, 6);

  for (let col = 0; col < columns; col++) {
    for (let row = 0; row < rows; row++) {
      if (!(errors[col]?.[row] > 0)) continue;

      const right = yAxisWidth + (col + 1) * cellWidth;
      const bottom = plotHeight - row * cellHeight;

      ctx.beginPath();
      ctx.moveTo(right, bottom - triangleSize);
      ctx.lineTo(right, bottom);
      ctx.lineTo(right - triangleSize, bottom);
      ctx.closePath();
      ctx.fill();
    }
  }

  return { yAxisWidth };
}
