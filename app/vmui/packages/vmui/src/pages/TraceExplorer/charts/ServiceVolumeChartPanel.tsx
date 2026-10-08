import { FC, useCallback } from "preact/compat";
import ServiceVolumeChart from "../components/ServiceVolumeChart";
import { useServiceHits } from "../components/ServiceVolumeChart/useServiceHits";
import { useTimePeriod } from "../hooks/useTimePeriod";
import { SERVICE_FIELD } from "../utils";
import { TimePeriod } from "../../../types";
import { ApplyHitsFilter } from "../../../components/Chart/BarHitsChart/types";
import { useTraceExplorerChartContext } from "./TraceExplorerChartContext";
import { useChartRun } from "./useChartRun";
import { TraceChartPanelProps, TraceChartRun } from "./types";

const ServiceVolumeChartPanel: FC<TraceChartPanelProps> = ({ run, isActive }) => {
  const { hits, data, period: hitsPeriod, isLoading, error, fetchHits } = useServiceHits();
  const { period: livePeriod, setPeriod } = useTimePeriod();
  const { applyFieldFilter } = useTraceExplorerChartContext();

  const load = useCallback((chartRun: TraceChartRun) => {
    fetchHits(chartRun.query, chartRun.startNs, chartRun.endNs, chartRun.extraClauses);
  }, [fetchHits]);
  useChartRun(isActive, run, load);

  const period = hitsPeriod ?? livePeriod;

  const handleSetPeriod = useCallback((nextPeriod: TimePeriod) => {
    setPeriod({ nextPeriod });
  }, [setPeriod]);

  const handleApplyFilter = useCallback<ApplyHitsFilter>((value, mode) => {
    applyFieldFilter(SERVICE_FIELD, value, mode);
  }, [applyFieldFilter]);

  return (
    <ServiceVolumeChart
      hits={hits}
      data={data}
      isLoading={isLoading}
      error={error}
      period={period}
      onSetPeriod={handleSetPeriod}
      onApplyFilter={handleApplyFilter}
    />
  );
};

export default ServiceVolumeChartPanel;
