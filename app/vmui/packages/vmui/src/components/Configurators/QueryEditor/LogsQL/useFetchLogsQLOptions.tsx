import { useEffect, useState, useRef, useCallback } from "preact/compat";
import dayjs from "dayjs";
import { ContextData, ContextType } from "./types";
import { SuggestFunctionIcon, SuggestLabelIcon, SuggestMetricIcon, SuggestValueIcon } from "../../../Main/Icons";
import { AutocompleteOptions } from "../../../Main/Autocomplete";
import { useAppState } from "../../../../state/common/StateContext";
import { useTimePeriod } from "../../../../pages/TraceExplorer/hooks/useTimePeriod";
import { AUTOCOMPLETE_LIMITS } from "../../../../constants/queryAutocomplete";
import { LogsFieldValues } from "../../../../api/types";
import { useLogsDispatch, useLogsState } from "../../../../state/logsPanel/LogsStateContext";
import { useTenant } from "../../../../hooks/useTenant";
import { generateQuery } from "./utils";

type FetchDataArgs = {
  urlSuffix: string;
  setter: (value: LogsFieldValues[]) => void;
  params?: URLSearchParams;
}

const icons = {
  [ContextType.FilterName]: <SuggestMetricIcon/>,
  [ContextType.FilterUnknown]: <SuggestMetricIcon/>,
  [ContextType.FilterValue]: <SuggestValueIcon/>,
  [ContextType.PipeName]: <SuggestFunctionIcon/>,
  [ContextType.PipeValue]: <SuggestLabelIcon/>,
  [ContextType.Unknown]: <SuggestValueIcon/>,
  [ContextType.FilterOrPipeName]: <SuggestFunctionIcon/>
};

export const useFetchLogsQLOptions = (contextData?: ContextData) => {
  const { serverUrl } = useAppState();
  const { period } = useTimePeriod();
  // period is nanoseconds (bigint); this endpoint wants unix seconds.
  const start = Number(period.start / 1_000_000_000n);
  const end = Number(period.end / 1_000_000_000n);
  const { autocompleteCache } = useLogsState();
  const dispatch = useLogsDispatch();
  const tenant = useTenant();

  const [loading, setLoading] = useState(false);

  const [fieldNames, setFieldNames] = useState<AutocompleteOptions[]>([]);
  const [fieldValues, setFieldValues] = useState<AutocompleteOptions[]>([]);

  const abortControllerRef = useRef(new AbortController());
  const fetchDataRef = useRef<(args: FetchDataArgs) => Promise<void>>();

  const getQueryParams = useCallback((params?: Record<string, string>) => {
    const startDay = dayjs(start * 1000).startOf("day").valueOf() / 1000;
    const endDay = dayjs(end * 1000).endOf("day").valueOf() / 1000;

    return new URLSearchParams({
      ...(params || {}),
      limit: `${AUTOCOMPLETE_LIMITS.queryLimit}`,
      start: `${startDay}`,
      end: `${endDay}`
    });
  }, [start, end]);

  const processData = (values: LogsFieldValues[], type: ContextType): AutocompleteOptions[] => {
    return values.map(v => ({
      value: v.value,
      type: `${type}`,
      icon: icons[type]
    }));
  };

  const fetchData = async ({ urlSuffix, setter, params }: FetchDataArgs) => {
    abortControllerRef.current.abort();
    abortControllerRef.current = new AbortController();
    const { signal } = abortControllerRef.current;
    const tenantString = new URLSearchParams(tenant).toString();

    const key = `${urlSuffix}?${params?.toString()}&${tenantString}`;

    setLoading(true);
    try {
      const cachedData = autocompleteCache.get(key);
      if (cachedData) {
        setter(cachedData);
        setLoading(false);
        return;
      }

      const response = await fetch(`${serverUrl}/select/logsql/${urlSuffix}?${params}`, {
        signal,
        headers: { ...tenant }
      });

      if (response.ok) {
        const data = await response.json();
        const value = (data?.values || []) as LogsFieldValues[];
        setter(value || []);
        dispatch({ type: "SET_AUTOCOMPLETE_CACHE", payload: { key, value } });
      }
      setLoading(false);
    } catch (e) {
      if (e instanceof Error && e.name !== "AbortError") {
        dispatch({ type: "SET_AUTOCOMPLETE_CACHE", payload: { key, value: [] } });
        setLoading(false);
        console.error(e);
      }
    }
  };

  fetchDataRef.current = fetchData;

  // fetch field names
  useEffect(() => {
    const validContexts = [ContextType.FilterName, ContextType.FilterUnknown, ContextType.FilterOrPipeName];
    const isInvalidContext = !validContexts.includes(contextData?.contextType || ContextType.Unknown);
    if (!serverUrl || isInvalidContext) {
      return;
    }

    // eslint-disable-next-line @eslint-react/set-state-in-effect -- clears stale options synchronously before starting a new fetch, so old results aren't shown while loading
    setFieldNames([]);

    const setter = (filterNames: LogsFieldValues[]) => {
      setFieldNames(processData(filterNames, ContextType.FilterName));
    };

    fetchDataRef.current?.({
      urlSuffix: "field_names",
      setter: setter,
      params: getQueryParams({ query: contextData?.queryBeforeIncompleteFilter || "*" })
    });

    return () => abortControllerRef.current?.abort();
  }, [serverUrl, contextData, start, end, getQueryParams]);

  // fetch field values
  useEffect(() => {
    const isInvalidContext = contextData?.contextType !== ContextType.FilterValue;
    if (!serverUrl || isInvalidContext || !contextData?.filterName) {
      return;
    }

    // eslint-disable-next-line @eslint-react/set-state-in-effect -- clears stale options synchronously before starting a new fetch, so old results aren't shown while loading
    setFieldValues([]);

    const setter = (filterValues: LogsFieldValues[]) => {
      setFieldValues(processData(filterValues, ContextType.FilterValue));
    };

    fetchDataRef.current?.({
      urlSuffix: "field_values",
      setter: setter,
      params: getQueryParams({ query: generateQuery(contextData), field: contextData.filterName })
    });

    return () => abortControllerRef.current?.abort();
  }, [serverUrl, contextData, start, end, getQueryParams]);

  return {
    fieldNames,
    fieldValues,
    loading,
  };
};
