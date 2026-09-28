import { useCallback, useEffect, useRef, useState } from "preact/compat";
import { formatDurationForInput, parseDurationMs } from "../utils";
import { DURATION_MAX_FIELD, DURATION_MIN_FIELD, useExtraFilters } from "./useExtraFilters";

export interface DurationRequest {
  min: string;
  max: string;
  token: number;
}

function nsToDisplay(value: string): string {
  if (!value) return "";
  const ns = Number(value);
  return Number.isFinite(ns) ? formatDurationForInput(ns) : "";
}

export function useFiltersSidebarState(durationRequest?: DurationRequest | null) {
  const { selectedValues, setFieldValue } = useExtraFilters();

  const currentMinNs = selectedValues(DURATION_MIN_FIELD)[0] ?? "";
  const currentMaxNs = selectedValues(DURATION_MAX_FIELD)[0] ?? "";

  const [minDuration, setMinDuration] = useState(() => nsToDisplay(currentMinNs));
  const [maxDuration, setMaxDuration] = useState(() => nsToDisplay(currentMaxNs));
  const lastCommittedMinRef = useRef(currentMinNs);
  const lastCommittedMaxRef = useRef(currentMaxNs);
  const [tagName, setTagName] = useState("");
  const [tagValue, setTagValue] = useState("");

  const selectTagName = useCallback((value: string) => {
    setTagName(value);
    setTagValue("");
  }, []);

  useEffect(() => {
    if (!durationRequest) return;
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- applies an externally-requested duration filter, signalled via durationRequest.token
    setMinDuration(durationRequest.min);
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- applies an externally-requested duration filter, signalled via durationRequest.token
    setMaxDuration(durationRequest.max);
    // deliberately keyed on the token, not the whole object: a new durationRequest object with the same
    // token must not re-trigger this, since 'token' is what signals a genuinely new request
    // eslint-disable-next-line @eslint-react/exhaustive-deps
  }, [durationRequest?.token]);

  // Picks up a duration filter removed (or changed) from outside this hook - e.g. via the
  // ExtraFiltersPanel's own remove button - and reflects it back into the input text.
  useEffect(() => {
    if (currentMinNs === lastCommittedMinRef.current) return;
    lastCommittedMinRef.current = currentMinNs;
    // eslint-disable-next-line @eslint-react/set-state-in-effect
    setMinDuration(nsToDisplay(currentMinNs));
  }, [currentMinNs]);

  useEffect(() => {
    if (currentMaxNs === lastCommittedMaxRef.current) return;
    lastCommittedMaxRef.current = currentMaxNs;
    // eslint-disable-next-line @eslint-react/set-state-in-effect
    setMaxDuration(nsToDisplay(currentMaxNs));
  }, [currentMaxNs]);

  useEffect(() => {
    const ms = parseDurationMs(minDuration);
    const next = ms !== null ? `${Math.round(ms * 1_000_000)}` : "";
    if (next !== currentMinNs) {
      lastCommittedMinRef.current = next;
      setFieldValue(DURATION_MIN_FIELD, next);
    }
    // eslint-disable-next-line @eslint-react/exhaustive-deps
  }, [minDuration]);

  useEffect(() => {
    const ms = parseDurationMs(maxDuration);
    const next = ms !== null ? `${Math.round(ms * 1_000_000)}` : "";
    if (next !== currentMaxNs) {
      lastCommittedMaxRef.current = next;
      setFieldValue(DURATION_MAX_FIELD, next);
    }
    // eslint-disable-next-line @eslint-react/exhaustive-deps
  }, [maxDuration]);

  return {
    minDuration,
    setMinDuration,
    maxDuration,
    setMaxDuration,
    tagName,
    setTagName: selectTagName,
    tagValue,
    setTagValue,
  };
}
