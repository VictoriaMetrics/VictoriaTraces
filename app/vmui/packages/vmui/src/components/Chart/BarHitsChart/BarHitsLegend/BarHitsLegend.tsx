import { FC } from "preact/compat";
import "./style.scss";
import BarHitsLegendItem from "./BarHitsLegendItem";
import { LegendLogHits } from "../../../../api/types";
import { ApplyHitsFilter } from "../types";

interface Props {
  legendDetails: LegendLogHits[];
  hiddenKeys: ReadonlySet<string>;
  onToggleSeries: (key: string, isMultiSelect: boolean) => void;
  onApplyFilter: ApplyHitsFilter;
}

const BarHitsLegend: FC<Props> = ({ legendDetails, hiddenKeys, onToggleSeries, onApplyFilter }) => {
  const totalHits = legendDetails[0]?.totalHits || 0;

  return (
    <div className="vm-bar-hits-legend">
      {legendDetails.map((legend) => (
        <BarHitsLegendItem
          key={legend.key}
          legend={legend}
          isHidden={hiddenKeys.has(legend.key)}
          onToggleSeries={onToggleSeries}
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
