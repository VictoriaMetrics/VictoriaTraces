/** Number of columns across the x (time) axis. Traces are bucketed into this many
 * equal-width slices spanning the selected time range. */
export const HEATMAP_TIME_BUCKETS = 96;

/** Fixed CSS-pixel height of the plot area (excludes the x-axis gutter). The chart is not
 * user-resizable. */
export const HEATMAP_CANVAS_HEIGHT = 120;

/** Fixed CSS-pixel height reserved below the plot for x-axis labels - two lines (time, and
 * an anchoring date on ticks that need one). */
export const HEATMAP_X_AXIS_HEIGHT = 30;

/** Axis label font, matching VictoriaLogs' uPlot charts exactly (utils/uplot/axes.ts) so
 * both products' time-series axes read the same. */
export const HEATMAP_AXIS_FONT = "10px Arial";

/** Fallback cell color if the --color-primary CSS custom property can't be read
 * (e.g. in a non-browser test environment). Matches $color-dodger-blue in variables.scss. */
export const HEATMAP_FALLBACK_COLOR = "#1A90FF";

/** Fallback axis label color if --color-text-secondary can't be read. Matches
 * --color-text-secondary's light-theme value in style.scss. */
export const HEATMAP_AXIS_FALLBACK_COLOR = "#706F6F";

/** Error-cell corner-triangle color. Fixed rather than theme-derived: it needs to hold
 * contrast against the full 8%-95% density range of the blue fill, which a neutral ink/paper
 * color can't do (it vanishes against the darkest cells) - this pink was chosen for holding
 * up at both extremes. */
export const HEATMAP_MARKER_COLOR = "#E93D82";

/** Number of evenly-spaced tick labels drawn along the x (time) axis. */
export const HEATMAP_X_AXIS_TICKS = 5;

/** Number of evenly-spaced tick labels drawn along the y (duration) axis. */
export const HEATMAP_Y_AXIS_TICKS = 5;

const DURATION_BAND_FLOOR_US = 1; // 1us
const DURATION_BAND_TOP_ANCHOR_US = 100_000_000; // 100s

function buildDurationBandBoundariesUs(): number[] {
  const boundaries: number[] = [];
  for (let magnitude = DURATION_BAND_FLOOR_US; magnitude <= DURATION_BAND_TOP_ANCHOR_US; magnitude *= 10) {
    boundaries.push(magnitude);
    if (magnitude * 3 <= DURATION_BAND_TOP_ANCHOR_US) boundaries.push(magnitude * 3);
  }
  return boundaries;
}

/** Ascending duration-band boundaries in microseconds, e.g. [B0, B1, ..., B_{K-1}].
 * Band `i` covers `[boundaries[i-1], boundaries[i])`, band 0 covers `< boundaries[0]`,
 * and the last band covers `>= boundaries[K-1]`. */
export const HEATMAP_DURATION_BAND_BOUNDARIES_US = buildDurationBandBoundariesUs();

/** Number of rows across the y (duration) axis: one band below the lowest boundary,
 * one above the highest, and one per gap in between. */
export const HEATMAP_DURATION_BUCKETS = HEATMAP_DURATION_BAND_BOUNDARIES_US.length + 1;
