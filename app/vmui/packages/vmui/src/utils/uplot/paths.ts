import uPlot, { Series } from "uplot";
import { HITS_BAR_WIDTH_FACTOR, HITS_BARS_VIEW } from "../../constants/logs";
import { GRAPH_STYLES } from "../../components/Chart/BarHitsChart/types";

const barPaths = (
  u: uPlot,
  seriesIdx: number,
  idx0: number,
  idx1: number,
): Series.Paths | null => {
  // Max bar width keeps bars the same size as with a full grid when the period has fewer than HITS_BARS_VIEW points.
  // Clamped to 1px: on very narrow plots the formula goes non-positive, which collapses the bars.
  const maxBarWidth = Math.max(1, (u.under.clientWidth / HITS_BARS_VIEW) - 1);
  const pathBuilderFactory = uPlot?.paths?.bars?.({ size: [HITS_BAR_WIDTH_FACTOR, maxBarWidth] });
  return pathBuilderFactory ? pathBuilderFactory(u, seriesIdx, idx0, idx1) : null;
};

const getSeriesPaths = (type?: GRAPH_STYLES) => {
  switch (type) {
    case GRAPH_STYLES.BAR:
      return barPaths;
    default:
      return;
  }
};

export default getSeriesPaths;
