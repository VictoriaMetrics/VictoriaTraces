import { FC, useCallback, useEffect, useMemo, useRef, useState } from "preact/compat";
import { useNavigate, useSearchParams } from "react-router-dom";
import TraceExplorerHeader from "./components/TraceExplorerHeader";
import { TRACE_QUERY_URL_PARAMS, useTraceQueryState } from "./hooks/useTraceQueryState";
import TracesResultsTable from "./components/TracesResultsTable";
import "./style.scss";
import { useTimePeriod } from "./hooks/useTimePeriod";
import { useLimitController } from "../../components/Configurators/TracesLimitController/hooks/useLimitController";
import { useLogsqlTracesSearch } from "./hooks/useLogsqlTracesSearch";
import classNames from "classnames";
import useDeviceDetect from "../../hooks/useDeviceDetect";
import { useQueryState } from "../../state/query/QueryStateContext";
import FiltersSidebar from "./components/FiltersSidebar";
import ExtraFiltersPanel from "./components/ExtraFiltersPanel";
import { DURATION_MAX_FIELD, DURATION_MIN_FIELD, useExtraFilters } from "./hooks/useExtraFilters";
import { useFiltersSidebarVisible } from "./hooks/useFiltersSidebarVisible";
import { HeatmapSelectionRange } from "./components/TracesHeatmap";
import TraceInfoDrawer from "./components/TraceInfoDrawer";
import LineLoader from "../../components/Main/LineLoader";
import ApiErrorAlert from "./components/ApiErrorAlert";
import { addQueryToHistoryStorage } from "../../components/QueryHistory/utils";
import { DurationRequest } from "./hooks/useFiltersSidebarState";
import { addFilterClause, buildDurationClause, buildExcludeClause, formatDurationRangeForInput } from "./utils";
import { nanosToIsoString } from "../../utils/time";
import ChartModeToggle from "./components/ChartModeToggle";
import { TRACE_CHARTS, TraceChartRun } from "./charts";
import { TraceExplorerChartContextValue, TraceExplorerChartProvider } from "./charts/TraceExplorerChartContext";
import { useChartMode } from "./hooks/useChartMode";
import { HitsFilterMode } from "../../components/Chart/BarHitsChart/types";

const noop = () => {};

const TraceExplorer: FC = () => {
  const { isMobile } = useDeviceDetect();
  const { period, getUrlParams, refreshPeriod } = useTimePeriod();
  const [searchParams, setSearchParams] = useSearchParams();
  const { limit, setLimit } = useLimitController();
  const { executeQueryTrigger } = useQueryState();
  const navigate = useNavigate();
  const { isVisible: isFiltersSidebarVisible, setVisible: setFiltersSidebarVisible } = useFiltersSidebarVisible();
  const [activeChartId, setActiveChartId] = useChartMode(TRACE_CHARTS);

  const { query, setQuery } = useTraceQueryState();
  const { extraFilters, extraClauses, extraParams, addFilter, removeFilter, selectedValues } = useExtraFilters();
  const extraFiltersKey = useMemo(
    () => extraFilters.map(f => `${f.field}::${f.value}`).sort().join("|"),
    [extraFilters]
  );
  const selectedTraceId = searchParams.get(TRACE_QUERY_URL_PARAMS.TRACE_ID) || "";
  const setSelectedTraceId = useCallback((traceId: string) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev);
      if (traceId) next.set(TRACE_QUERY_URL_PARAMS.TRACE_ID, traceId);
      else next.delete(TRACE_QUERY_URL_PARAMS.TRACE_ID);
      return next;
    });
  }, [setSearchParams]);

  const {
    traces, spansByTraceId,
    isLoading: isSearchLoading, error: searchError, search,
  } = useLogsqlTracesSearch();

  // A heatmap rectangle selection filters the table only (extra time+duration bounds on
  // top of the existing query) — the heatmap chart itself keeps showing the unfiltered picture.
  const {
    traces: previewTraces, spansByTraceId: previewSpansByTraceId,
    isLoading: isPreviewLoading, error: previewError, search: searchPreview,
  } = useLogsqlTracesSearch();
  const [heatmapSelection, setHeatmapSelection] = useState<HeatmapSelectionRange | null>(null);
  const [durationRequest, setDurationRequest] = useState<DurationRequest | null>(null);
  const [committedQuery, setCommittedQuery] = useState(query);
  // Snapshot of the last committed run; chart panels fetch their own data from it.
  const [chartRun, setChartRun] = useState<TraceChartRun | null>(null);
  const runCounterRef = useRef(0);

  const displayedTraces = heatmapSelection ? previewTraces : traces;
  const displayedSpansByTraceId = heatmapSelection ? previewSpansByTraceId : spansByTraceId;
  const displayedError = heatmapSelection ? previewError : searchError;
  const selectedTrace = displayedTraces.find(t => t.traceID === selectedTraceId);

  const minDurationNs = selectedValues(DURATION_MIN_FIELD)[0];
  const maxDurationNs = selectedValues(DURATION_MAX_FIELD)[0];
  const minDurationUs = minDurationNs ? Number(minDurationNs) / 1000 : 0;
  const maxDurationUs = maxDurationNs ? Number(maxDurationNs) / 1000 : Infinity;

  const handleRun = useCallback((nextQuery?: string, preserveSelection = false) => {
    const queryToRun = (nextQuery ?? query).trim();
    if (!queryToRun) return;
    if (!preserveSelection) setSelectedTraceId("");
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- called from run-triggering effects below; clears any heatmap selection before a new query executes
    setHeatmapSelection(null);
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- called from run-triggering effects below
    setCommittedQuery(queryToRun);
    addQueryToHistoryStorage(queryToRun);
    search(queryToRun, period.start, period.end, limit, extraParams);
    runCounterRef.current += 1;
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- called from run-triggering effects below; hands chart panels the run to load
    setChartRun({
      id: runCounterRef.current,
      query: queryToRun,
      startNs: period.start,
      endNs: period.end,
      extraClauses,
    });
  }, [query, period, limit, extraParams, extraClauses, search, setSelectedTraceId]);

  const handleApplyFieldFilter = useCallback((field: string, value: string, mode: HitsFilterMode) => {
    if (mode === "include") {
      addFilter(field, value);
      return;
    }
    const nextQuery = addFilterClause(query, buildExcludeClause(field, value));
    setQuery(nextQuery);
    // preserveSelection: a second URL update in the same tick (clearing trace_id) would be computed from
    // stale search params and drop the query just written by setQuery.
    handleRun(nextQuery, true);
  }, [query, addFilter, setQuery, handleRun]);

  useEffect(() => {
    if (!heatmapSelection) return;
    const { min, max } = formatDurationRangeForInput(heatmapSelection.durationLowUs, heatmapSelection.durationHighUs);
    const extraClause = buildDurationClause(min, max);
    const combinedQuery = addFilterClause(query, extraClause);
    searchPreview(combinedQuery, heatmapSelection.timeLowNs, heatmapSelection.timeHighNs, limit, extraParams);
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- intentionally reacts only to a new heatmap selection; `query`/`limit` are read as of that moment (already reset to null by handleRun whenever a fresh query runs), and `searchPreview` is stable per useLogsqlTracesSearch's own deps
  }, [heatmapSelection]);

  const handleCommitHeatmapSelection = useCallback((selection: HeatmapSelectionRange) => {
    const { min, max } = formatDurationRangeForInput(selection.durationLowUs, selection.durationHighUs);
    const timeParams = getUrlParams({
      nextPeriod: {
        from: nanosToIsoString(selection.timeLowNs),
        to: nanosToIsoString(selection.timeHighNs),
      },
    });

    setFiltersSidebarVisible(true);
    setDurationRequest({ min, max, token: Date.now() });
    setHeatmapSelection(null);
    setSearchParams(prev => {
      const next = new URLSearchParams(prev);
      timeParams.forEach((value, key) => next.set(key, value));
      return next;
    });
  }, [setFiltersSidebarVisible, getUrlParams, setSearchParams]);

  const chartContext = useMemo<TraceExplorerChartContextValue>(() => ({
    heatmap: {
      minDurationUs,
      maxDurationUs,
      highlightedTrace: selectedTrace ? {
        startTimeUs: selectedTrace.startTime,
        durationUs: selectedTrace.duration,
      } : null,
      onSelectionChange: setHeatmapSelection,
      onCommitSelection: handleCommitHeatmapSelection,
    },
    applyFieldFilter: handleApplyFieldFilter,
  }), [minDurationUs, maxDurationUs, selectedTrace, handleCommitHeatmapSelection, handleApplyFieldFilter]);

  useEffect(() => {
    refreshPeriod();
    if (executeQueryTrigger > 0 && query.trim()) handleRun(undefined, true);
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- intentionally reacts only to the external "run now" signal (executeQueryTrigger); `query`/`handleRun` are read fresh at fire time, and including `handleRun` would also re-fire on every query/period/limit change since handleRun depends on those
  }, [executeQueryTrigger]);

  useEffect(() => {
    if (query.trim()) handleRun(undefined, true);
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- intentionally mount-only: runs the initial (e.g. URL-prefilled) query once; adding query/handleRun would re-run on every keystroke
  }, []);

  const isFirstPeriodRenderRef = useRef(true);
  useEffect(() => {
    if (isFirstPeriodRenderRef.current) {
      isFirstPeriodRenderRef.current = false;
      return;
    }
    if (query.trim()) handleRun(undefined, true);
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- intentionally reacts only to period changes (skips the first render via the ref guard, since mount is handled above); adding query/handleRun would re-run on every keystroke or unrelated handleRun-dependency change
  }, [period.start, period.end]);

  const isFirstExtraFiltersRenderRef = useRef(true);
  useEffect(() => {
    if (isFirstExtraFiltersRenderRef.current) {
      isFirstExtraFiltersRenderRef.current = false;
      return;
    }
    if (query.trim()) handleRun(undefined, true);
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- intentionally reacts only to extraFilters content changes (skips the first render via the ref guard); adding query/handleRun would re-run on every keystroke or unrelated handleRun-dependency change
  }, [extraFiltersKey]);

  return (
    <div
      className={classNames({
        "vm-trace-explorer": true,
        "vm-trace-explorer_with-sidebar": isFiltersSidebarVisible,
      })}
    >
      {isFiltersSidebarVisible && (
        <FiltersSidebar
          query={committedQuery}
          onClose={() => setFiltersSidebarVisible(false)}
          durationRequest={durationRequest}
        />
      )}
      <div className="vm-trace-explorer-content">
        <div
          className={classNames({
            "vm-trace-explorer-header": true,
            "vm-block": true,
            "vm-block_mobile": isMobile,
          })}
        >
          <TraceExplorerHeader
            mode="search"
            traceId=""
            onChangeTraceId={noop}
            query={query}
            onChangeQuery={setQuery}
            limit={limit}
            onChangeLimit={setLimit}
            isLoading={isSearchLoading}
            onRun={handleRun}
          />
          <ExtraFiltersPanel
            extraFilters={extraFilters}
            onRemove={removeFilter}
          />
        </div>

        <div className="vm-trace-explorer-traces-body">
          <ChartModeToggle
            charts={TRACE_CHARTS}
            activeId={activeChartId}
            onChange={setActiveChartId}
          />
          {/* Inactive charts stay mounted (hidden) so their data survives tab switches; they load the run lazily. */}
          <TraceExplorerChartProvider value={chartContext}>
            {TRACE_CHARTS.map(chart => {
              const isActive = chart.id === activeChartId;
              return (
                <div
                  key={chart.id}
                  className={classNames({
                    "vm-trace-explorer-chart": true,
                    "vm-trace-explorer-chart_hidden": !isActive,
                  })}
                >
                  <chart.Component
                    run={chartRun}
                    isActive={isActive}
                  />
                </div>
              );
            })}
          </TraceExplorerChartProvider>
          <div
            className={classNames("vm-trace-explorer-traces-body-table", "vm-block", {
              "vm-trace-explorer-traces-body-table_loading": heatmapSelection ? isPreviewLoading : isSearchLoading,
            })}
          >
            {(heatmapSelection ? isPreviewLoading : isSearchLoading) && <LineLoader/>}
            {displayedError ? (
              <ApiErrorAlert
                error={displayedError}
                className="vm-trace-explorer-traces-body-error"
              />
            ) : (
              <TracesResultsTable
                results={displayedTraces}
                activeTraceID={selectedTraceId}
                onClickRow={row => setSelectedTraceId(row.traceID)}
                onOpenTrace={row => navigate(
                  `/trace?trace_id=${encodeURIComponent(row.traceID)}`,
                  { state: { autoRun: true } }
                )}
              />
            )}
          </div>
          {selectedTrace && (
            <TraceInfoDrawer
              trace={selectedTrace}
              spans={displayedSpansByTraceId.get(selectedTrace.traceID) || []}
              periodStart={period.start}
              periodEnd={period.end}
              onClose={() => setSelectedTraceId("")}
              onViewFullTrace={() => navigate(
                `/trace?trace_id=${encodeURIComponent(selectedTrace.traceID)}`,
                { state: { autoRun: true } }
              )}
            />
          )}
        </div>
      </div>
    </div>
  );
};

export default TraceExplorer;
