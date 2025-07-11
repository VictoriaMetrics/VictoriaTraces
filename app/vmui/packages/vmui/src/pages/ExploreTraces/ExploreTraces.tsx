import { FC, useEffect, useMemo, useState } from "preact/compat";
import ExploreTracesBody from "./ExploreTracesBody/ExploreTracesBody";
import useStateSearchParams from "../../hooks/useStateSearchParams";
import useSearchParamsFromObject from "../../hooks/useSearchParamsFromObject";
import { useFetchTraces } from "./hooks/useFetchTraces";
import { useAppState } from "../../state/common/StateContext";
import Alert from "../../components/Main/Alert/Alert";
import ExploreTracesHeader from "./ExploreTracesHeader/ExploreTracesHeader";
import "./style.scss";
import { ErrorTypes, TimeParams } from "../../types";
import { useTimeState } from "../../state/time/TimeStateContext";
import { getFromStorage, saveToStorage } from "../../utils/storage";
import ExploreTracesBarChart from "./ExploreTracesBarChart/ExploreTracesBarChart";
import { useFetchTraceHits } from "./hooks/useFetchTraceHits";
import { TRACES_ENTRIES_LIMIT } from "../../constants/traces";
import { getTimeperiodForDuration, relativeTimeOptions } from "../../utils/time";
import { useSearchParams } from "react-router-dom";
import { useQueryDispatch, useQueryState } from "../../state/query/QueryStateContext";
import { getUpdatedHistory } from "../../components/QueryHistory/utils";
import { useDebounceCallback } from "../../hooks/useDebounceCallback";
import usePrevious from "../../hooks/usePrevious";

const storageLimit = Number(getFromStorage("TRACES_LIMIT"));
const defaultLimit = isNaN(storageLimit) ? TRACES_ENTRIES_LIMIT : storageLimit;

const ExploreTraces: FC = () => {
  const { serverUrl } = useAppState();
  const { queryHistory } = useQueryState();
  const queryDispatch = useQueryDispatch();
  const { duration, relativeTime, period: periodState } = useTimeState();
  const { setSearchParamsFromKeys } = useSearchParamsFromObject();
  const [searchParams] = useSearchParams();
  const hideChart = useMemo(() => searchParams.get("hide_chart"), [searchParams]);
  const prevHideChart = usePrevious(hideChart);

  const [limit, setLimit] = useStateSearchParams(defaultLimit, "limit");
  const [query, setQuery] = useStateSearchParams("*", "query");

  const updateHistory = () => {
    const history = getUpdatedHistory(query, queryHistory[0]);
    queryDispatch({
      type: "SET_QUERY_HISTORY",
      payload: {
        key: "TRACES_QUERY_HISTORY",
        history: [history],
      }
    });
  };

  const [isUpdatingQuery, setIsUpdatingQuery] = useState(false);
  const [period, setPeriod] = useState<TimeParams>(periodState);
  const [queryError, setQueryError] = useState<ErrorTypes | string>("");

  const { traces, isLoading, error, fetchTraces, abortController } = useFetchTraces(serverUrl, query, limit);
  const { fetchTraceHits, ...dataTraceHits } = useFetchTraceHits(serverUrl, query);

  const fetchData = (p: TimeParams, hits: boolean) => {
    fetchTraces(p).then((isSuccess) => {
      if (isSuccess && hits) fetchTraceHits(p);
    }).catch(() => {/* error handled elsewhere */});
  };

  const debouncedFetchTraces = useDebounceCallback(fetchData, 300);

  const getPeriod = () => {
    const relativeTimeOpts = relativeTimeOptions.find(d => d.id === relativeTime);
    if (!relativeTimeOpts) return periodState;
    const { duration, until } = relativeTimeOpts;
    return getTimeperiodForDuration(duration, until());
  };

  const handleRunQuery = () => {
    if (!query) {
      setQueryError(ErrorTypes.validQuery);
      return;
    }
    setQueryError("");

    const newPeriod = getPeriod();
    setPeriod(newPeriod);
    debouncedFetchTraces(newPeriod, !hideChart);
    setSearchParamsFromKeys({
      query,
      "g0.range_input": duration,
      "g0.end_input": newPeriod.date,
      "g0.relative_time": relativeTime || "none",
    });
    updateHistory();
  };

  const handleChangeLimit = (limit: number) => {
    setLimit(limit);
    setSearchParamsFromKeys({ limit });
    saveToStorage("TRACES_LIMIT", `${limit}`);
  };

  const handleApplyFilter = (val: string) => {
    setQuery(prev => `${val} AND (${prev})`);
    setIsUpdatingQuery(true);
  };

  const handleUpdateQuery = () => {
    if (isLoading || dataTraceHits.isLoading) {
      abortController.abort?.();
      dataTraceHits.abortController.abort?.();
    } else {
      handleRunQuery();
    }
  };

  useEffect(() => {
    if (!query) return;
    handleRunQuery();
  }, [periodState]);

  useEffect(() => {
    if (!isUpdatingQuery) return;
    handleRunQuery();
    setIsUpdatingQuery(false);
  }, [query, isUpdatingQuery]);

  useEffect(() => {
    if (!hideChart && prevHideChart) {
      fetchTraceHits(period);
    }
  }, [hideChart, prevHideChart, period]);

  return (
    <div className="vm-explore-traces">
      <ExploreTracesHeader
        query={query}
        error={queryError}
        limit={limit}
        onChange={setQuery}
        onChangeLimit={handleChangeLimit}
        onRun={handleUpdateQuery}
        isLoading={isLoading || dataTraceHits.isLoading}
      />
      {error && <Alert variant="error">{error}</Alert>}
      {!error && (
        <ExploreTracesBarChart
          {...dataTraceHits}
          query={query}
          period={period}
          onApplyFilter={handleApplyFilter}
        />
      )}
      <ExploreTracesBody
        data={traces}
        isLoading={isLoading}
      />
    </div>
  );
};

export default ExploreTraces;
