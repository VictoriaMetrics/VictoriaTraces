import { FC, useCallback, useEffect, useMemo, useRef, useState } from "preact/compat";
import { useResizeObserver } from "../../../../hooks/useResizeObserver";
import uPlot, { AlignedData } from "uplot";
import { ApplyHitsFilter, GraphOptions } from "../types";
import usePlotScale from "../../../../hooks/uplot/usePlotScale";
import useReadyChart from "../../../../hooks/uplot/useReadyChart";
import useZoomChart from "../../../../hooks/uplot/useZoomChart";
import stack from "../../../../utils/uplot/stack";
import useBarHitsOptions, { getLabelFromLogHit } from "../hooks/useBarHitsOptions";
import { LegendLogHits, LogHits } from "../../../../api/types";
import { addSeries, delSeries, setBand } from "../../../../utils/uplot";
import classNames from "classnames";
import BarHitsTooltip from "../BarHitsTooltip/BarHitsTooltip";
import { TimeParams, TimePeriod } from "../../../../types";
import BarHitsLegend from "../BarHitsLegend/BarHitsLegend";
import { calculateTotalHits, getHitKey, pruneHiddenSeries, sortLogHits, toggleHiddenSeries } from "../hitsUtils";

interface Props {
  logHits: LogHits[];
  data: AlignedData;
  period: TimeParams;
  setPeriod: (nextPeriod: TimePeriod) => void;
  onApplyFilter: ApplyHitsFilter;
  graphOptions: GraphOptions;
}

const BarHitsPlot: FC<Props> = ({ graphOptions, logHits, data: _data, period, setPeriod, onApplyFilter }: Props) => {
  // ResizeObserver, not window resize: a hidden panel gets its real size when shown.
  const containerRef = useRef<HTMLDivElement>(null);
  const { width: containerWidth = 0, height: containerHeight = 0 } = useResizeObserver({ ref: containerRef });
  const containerSize = useMemo(() => ({ width: containerWidth, height: containerHeight }), [containerWidth, containerHeight]);
  const uPlotRef = useRef<HTMLDivElement>(null);
  const [uPlotInst, setUPlotInst] = useState<uPlot>();

  const { xRange, setPlotScale } = usePlotScale({ period, setPeriod });
  const { onReadyChart, isPanning } = useReadyChart(setPlotScale);
  useZoomChart({ uPlotInst, element: uPlotRef, xRange, setPlotScale });

  const [hiddenKeys, setHiddenKeys] = useState<Set<string>>(() => new Set());
  const seriesKeys = useMemo(() => logHits.map(getHitKey), [logHits]);

  // Adjusted during render so the stack never sees keys of a previous data set.
  const [prevSeriesKeys, setPrevSeriesKeys] = useState(seriesKeys);
  if (prevSeriesKeys !== seriesKeys) {
    setPrevSeriesKeys(seriesKeys);
    setHiddenKeys(prev => pruneHiddenSeries(prev, seriesKeys));
  }

  const handleToggleSeries = useCallback((key: string, isMultiSelect: boolean) => {
    setHiddenKeys(prev => toggleHiddenSeries(seriesKeys, prev, key, isMultiSelect));
  }, [seriesKeys]);

  const { data, bands } = useMemo(() => {
    if (!graphOptions.stacked) return { data: _data, bands: [] };
    return stack(_data, i => hiddenKeys.has(seriesKeys[i - 1]));
  }, [graphOptions, _data, seriesKeys, hiddenKeys]);

  const { options, series, focusDataIdx } = useBarHitsOptions({
    data,
    logHits,
    bands,
    xRange,
    containerSize,
    onReadyChart,
    setPlotScale,
    graphOptions
  });

  const prepareLegend = useCallback((hits: LogHits[], totalHits: number): LegendLogHits[] => {
    return hits.map((hit, i) => {
      const legendItem: LegendLogHits = {
        key: getHitKey(hit),
        label: getLabelFromLogHit(hit),
        isOther: hit._isOther,
        fields: hit.fields,
        total: hit.total || 0,
        totalHits,
        stroke: series[i + 1]?.stroke,
      };

      return legendItem;
    }).sort(sortLogHits("total"));
  }, [series]);


  const legendDetails: LegendLogHits[] = useMemo(() => {
    const totalHits = calculateTotalHits(logHits);
    return prepareLegend(logHits, totalHits);
  }, [logHits, prepareLegend]);

  useEffect(() => {
    if (!uPlotInst) return;

    const syncedSeries = series.map((s, i) => (i ? { ...s, show: !hiddenKeys.has(seriesKeys[i - 1]) } : s));

    delSeries(uPlotInst);
    addSeries(uPlotInst, syncedSeries, true);
    setBand(uPlotInst, syncedSeries);
    bands.forEach(band => {
      uPlotInst.addBand(band);
    });
    uPlotInst.redraw();
  }, [series, bands, seriesKeys, hiddenKeys, uPlotInst]);

  useEffect(() => {
    if (!uPlotRef.current) return;
    const uplot = new uPlot(options, data, uPlotRef.current);
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- the uPlot instance needs its container in the DOM
    setUPlotInst(uplot);
    return () => uplot.destroy();
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- mount-only: the effects below apply later `options`/`data` changes
  }, []);

  useEffect(() => {
    if (!uPlotInst) return;
    uPlotInst.scales.x.range = () => [xRange.min, xRange.max];
    uPlotInst.redraw();
  }, [xRange, uPlotInst]);

  useEffect(() => {
    if (!uPlotInst || !containerSize.width) return;
    uPlotInst.setSize(containerSize);
    uPlotInst.redraw();
  }, [containerSize, uPlotInst]);

  useEffect(() => {
    if (!uPlotInst) return;
    uPlotInst.setData(data);
    uPlotInst.redraw();
  }, [data, uPlotInst]);

  return (
    <>
      <div
        className={classNames({
          "vm-bar-hits-chart": true,
          "vm-bar-hits-chart_panning": isPanning
        })}
        ref={containerRef}
      >
        <div
          className="vm-line-chart__u-plot"
          ref={uPlotRef}
        />
        <BarHitsTooltip
          uPlotInst={uPlotInst}
          data={_data}
          logHits={logHits}
          focusDataIdx={focusDataIdx}
        />
      </div>
      {uPlotInst && <BarHitsLegend
        hiddenKeys={hiddenKeys}
        onToggleSeries={handleToggleSeries}
        onApplyFilter={onApplyFilter}
        legendDetails={legendDetails}
      />}
    </>
  );
};

export default BarHitsPlot;
