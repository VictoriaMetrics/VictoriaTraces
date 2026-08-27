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
import { useFiltersSidebarVisible } from "./hooks/useFiltersSidebarVisible";
import TracesHeatmap, { HeatmapSelectionRange } from "./components/TracesHeatmap";
import { useHeatmapAggregation } from "./components/TracesHeatmap/useHeatmapAggregation";
import TraceInfoDrawer from "./components/TraceInfoDrawer";
import LineLoader from "../../components/Main/LineLoader";
import ApiErrorAlert from "./components/ApiErrorAlert";
import { addQueryToHistoryStorage } from "../../components/QueryHistory/utils";
import { DurationRequest } from "./hooks/useFiltersSidebarState";
import { buildDurationClause, formatDurationRangeForInput, parseDurationMs, parseFiltersFromQuery } from "./utils";
import { nanosToIsoString } from "../../utils/time";

const noop = () => {};

const TraceExplorer: FC = () => {
  const { isMobile } = useDeviceDetect();
  const { period, getUrlParams, refreshPeriod } = useTimePeriod();
  const [searchParams, setSearchParams] = useSearchParams();
  const { limit, setLimit } = useLimitController();
  const { executeQueryTrigger } = useQueryState();
  const navigate = useNavigate();
  const { isVisible: isFiltersSidebarVisible, setVisible: setFiltersSidebarVisible } = useFiltersSidebarVisible();

  const { query, setQuery } = useTraceQueryState();
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
  const {
    grid: heatmapGrid,
    isLoading: isHeatmapLoading, error: heatmapError, fetchHeatmap,
  } = useHeatmapAggregation();

  // A heatmap rectangle selection filters the table only (extra time+duration bounds on
  // top of the existing query) — the heatmap chart itself keeps showing the unfiltered picture.
  const {
    traces: previewTraces, spansByTraceId: previewSpansByTraceId,
    isLoading: isPreviewLoading, error: previewError, search: searchPreview,
  } = useLogsqlTracesSearch();
  const [heatmapSelection, setHeatmapSelection] = useState<HeatmapSelectionRange | null>(null);
  const [durationRequest, setDurationRequest] = useState<DurationRequest | null>(null);

  const displayedTraces = heatmapSelection ? previewTraces : traces;
  const displayedSpansByTraceId = heatmapSelection ? previewSpansByTraceId : spansByTraceId;
  const displayedError = heatmapSelection ? previewError : searchError;
  const selectedTrace = displayedTraces.find(t => t.traceID === selectedTraceId);

  // Recovered from the query text itself (rather than lifted out of FiltersSidebar's own
  // state) so the heatmap's y axis reacts the same way regardless of how the duration
  // clause got into the query — typed, committed from a heatmap selection, or URL-loaded.
  const { minDuration: durationFilterMin, maxDuration: durationFilterMax } = useMemo(
    () => parseFiltersFromQuery(query),
    [query]
  );
  const minDurationUs = (parseDurationMs(durationFilterMin) ?? 0) * 1000;
  const maxDurationUs = (parseDurationMs(durationFilterMax) ?? Infinity) * 1000;

  const handleRun = useCallback((nextQuery?: string, preserveSelection = false) => {
    const queryToRun = (nextQuery ?? query).trim();
    if (!queryToRun) return;
    if (!preserveSelection) setSelectedTraceId("");
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- called from run-triggering effects below; clears any heatmap selection before a new query executes
    setHeatmapSelection(null);
    addQueryToHistoryStorage(queryToRun);
    search(queryToRun, period.start, period.end, limit);
    fetchHeatmap(queryToRun, period.start, period.end);
  }, [query, period, limit, search, fetchHeatmap, setSelectedTraceId]);

  useEffect(() => {
    if (!heatmapSelection) return;
    const { min, max } = formatDurationRangeForInput(heatmapSelection.durationLowUs, heatmapSelection.durationHighUs);
    const extraClause = buildDurationClause(min, max);
    const combinedQuery = extraClause ? `${query.trim()} AND ${extraClause}` : query.trim();
    searchPreview(combinedQuery, heatmapSelection.timeLowNs, heatmapSelection.timeHighNs, limit);
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- intentionally reacts only to a new heatmap selection; `query`/`limit` are read as of that moment (already reset to null by handleRun whenever a fresh query runs), and `searchPreview` is stable per useLogsqlTracesSearch's own deps
  }, [heatmapSelection]);

  const handleCommitHeatmapSelection = useCallback((selection: HeatmapSelectionRange) => {
    const { min, max } = formatDurationRangeForInput(selection.durationLowUs, selection.durationHighUs);
    const extraClause = buildDurationClause(min, max);
    const combinedQuery = extraClause ? `${query.trim()} AND ${extraClause}` : query.trim();
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
      next.set(TRACE_QUERY_URL_PARAMS.QUERY, combinedQuery);
      return next;
    });
  }, [query, setFiltersSidebarVisible, getUrlParams, setSearchParams]);

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

  return (
    <div
      className={classNames({
        "vm-trace-explorer": true,
        "vm-trace-explorer_with-sidebar": isFiltersSidebarVisible,
      })}
    >
      {isFiltersSidebarVisible && (
        <FiltersSidebar
          query={query}
          onChangeQuery={setQuery}
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
        </div>

        <div className="vm-trace-explorer-traces-body">
          <TracesHeatmap
            grid={heatmapGrid}
            isLoading={isHeatmapLoading}
            error={heatmapError}
            periodStart={period.start}
            periodEnd={period.end}
            minDurationUs={minDurationUs}
            maxDurationUs={maxDurationUs}
            highlightedTrace={selectedTrace ? {
              startTimeUs: selectedTrace.startTime,
              durationUs: selectedTrace.duration,
            } : null}
            onSelectionChange={setHeatmapSelection}
            onCommitSelection={handleCommitHeatmapSelection}
          />
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
