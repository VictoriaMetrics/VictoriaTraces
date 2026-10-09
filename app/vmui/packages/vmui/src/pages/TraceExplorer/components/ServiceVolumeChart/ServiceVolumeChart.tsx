import { FC } from "preact/compat";
import classNames from "classnames";
import { AlignedData } from "uplot";
import { LogHits } from "../../../../api/types";
import { TimeParams, TimePeriod } from "../../../../types";
import BarHitsChart from "../../../../components/Chart/BarHitsChart/BarHitsChart";
import { ApplyHitsFilter } from "../../../../components/Chart/BarHitsChart/types";
import LineLoader from "../../../../components/Main/LineLoader";
import Alert from "../../../../components/Main/Alert/Alert";
import ApiErrorAlert from "../ApiErrorAlert";
import "./style.scss";

export interface ServiceVolumeChartProps {
  hits: LogHits[];
  data: AlignedData;
  isLoading: boolean;
  error?: string;
  period: TimeParams;
  onSetPeriod: (nextPeriod: TimePeriod) => void;
  onApplyFilter: ApplyHitsFilter;
}

const NO_DATA_MESSAGE = "No span volume available for the current query and time range.";

const ServiceVolumeChart: FC<ServiceVolumeChartProps> = ({
  hits, data, isLoading, error, period, onSetPeriod, onApplyFilter,
}) => {
  const hasData = hits.length > 0;

  return (
    <section
      className={classNames({
        "vm-service-volume-chart": true,
        "vm-service-volume-chart_loading": isLoading,
      })}
    >
      {isLoading && <LineLoader/>}
      {error && (
        <div className="vm-service-volume-chart__empty">
          <ApiErrorAlert error={error}/>
        </div>
      )}
      {!error && !hasData && !isLoading && (
        <div className="vm-service-volume-chart__empty">
          <Alert variant="info">{NO_DATA_MESSAGE}</Alert>
        </div>
      )}
      {!error && hasData && (
        <BarHitsChart
          logHits={hits}
          data={data}
          period={period}
          setPeriod={onSetPeriod}
          onApplyFilter={onApplyFilter}
        />
      )}
    </section>
  );
};

export default ServiceVolumeChart;
