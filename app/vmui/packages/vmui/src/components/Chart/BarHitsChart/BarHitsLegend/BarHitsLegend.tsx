import { FC, useEffect, useState } from "preact/compat";
import uPlot, { Series } from "uplot";
import "./style.scss";
import BarHitsLegendItem from "./BarHitsLegendItem";
import { LegendLogHits } from "../../../../api/types";
import { ApplyHitsFilter } from "../types";

interface Props {
  uPlotInst: uPlot;
  legendDetails: LegendLogHits[];
  onApplyFilter: ApplyHitsFilter;
}

const BarHitsLegend: FC<Props> = ({ uPlotInst, legendDetails, onApplyFilter }) => {
  const [series, setSeries] = useState<Series[]>([]);
  const totalHits = legendDetails[0]?.totalHits || 0;

  const handleRedrawGraph = () => {
    uPlotInst.redraw();
  };

  useEffect(() => {
    if (!uPlotInst.hooks.draw) {
      uPlotInst.hooks.draw = [];
    }
    uPlotInst.hooks.draw.push(() => {
      setSeries(uPlotInst.series.filter(s => s.scale !== "x"));
    });
  }, [uPlotInst]);

  return (
    <div className="vm-bar-hits-legend">
      {legendDetails.map((legend) => (
        <BarHitsLegendItem
          key={legend.label}
          legend={legend}
          series={series}
          onRedrawGraph={handleRedrawGraph}
          onApplyFilter={onApplyFilter}
        />
      ))}
      <div className="vm-bar-hits-legend-info">
        <div>
          Total hits: <b>{totalHits.toLocaleString("en-US")}</b>
        </div>
        <div>
          <code>L-Click</code> toggles visibility.&nbsp;
          <code>R-Click</code> opens menu.
        </div>
      </div>
    </div>
  );
};

export default BarHitsLegend;
