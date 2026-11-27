import { useCallback } from "preact/compat";
import { useSearchParams } from "react-router-dom";
import { CHART_MODE_URL_PARAM, resolveChartId } from "../charts/chartRegistry";
import { TraceChartDefinition } from "../charts/types";

/** Which registered chart is shown above the traces table; persisted in the URL so links keep the selected view */
export function useChartMode(charts: TraceChartDefinition[]): [string, (id: string) => void] {
  const [searchParams, setSearchParams] = useSearchParams();
  const activeId = resolveChartId(searchParams.get(CHART_MODE_URL_PARAM), charts);
  const defaultId = charts[0].id;

  const setActiveId = useCallback((next: string) => {
    if (next === activeId) return;
    setSearchParams(prev => {
      const params = new URLSearchParams(prev);
      if (next === defaultId) params.delete(CHART_MODE_URL_PARAM);
      else params.set(CHART_MODE_URL_PARAM, next);
      return params;
    }, { replace: true });
  }, [setSearchParams, defaultId, activeId]);

  return [activeId, setActiveId];
}
