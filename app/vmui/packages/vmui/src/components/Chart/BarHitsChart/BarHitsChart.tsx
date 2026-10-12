import { FC, useState } from "preact/compat";
import "./style.scss";
import "uplot/dist/uPlot.min.css";
import { AlignedData } from "uplot";
import { TimeParams, TimePeriod } from "../../../types";
import { LogHits } from "../../../api/types";
import { ApplyHitsFilter, GraphOptions, GRAPH_STYLES } from "./types";
import BarHitsOptions from "./BarHitsOptions/BarHitsOptions";
import BarHitsPlot from "./BarHitsPlot/BarHitsPlot";

interface Props {
  logHits: LogHits[];
  data: AlignedData;
  period: TimeParams;
  setPeriod: (nextPeriod: TimePeriod) => void;
  onApplyFilter: ApplyHitsFilter;
}

const BarHitsChart: FC<Props> = ({ logHits, data, period, setPeriod, onApplyFilter }) => {
  const [graphOptions, setGraphOptions] = useState<GraphOptions>({
    graphStyle: GRAPH_STYLES.BAR,
    stacked: false,
    fill: true,
  });

  return (
    <div className="vm-bar-hits-chart__wrapper">
      <BarHitsPlot
        logHits={logHits}
        data={data}
        period={period}
        setPeriod={setPeriod}
        onApplyFilter={onApplyFilter}
        graphOptions={graphOptions}
      />
      <BarHitsOptions onChange={setGraphOptions}/>
    </div>
  );
};

export default BarHitsChart;
