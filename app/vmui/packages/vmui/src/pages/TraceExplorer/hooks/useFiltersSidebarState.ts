import { useCallback, useEffect, useMemo, useRef, useState } from "preact/compat";
import {
  buildDurationClause,
  buildInClause,
  buildTagClause,
  mapSpanKindsToCodes,
  mergeQuery,
  parseFiltersFromQuery,
  Tag,
  toggleValue,
} from "../utils";

export interface DurationRequest {
  min: string;
  max: string;
  token: number;
}

export function useFiltersSidebarState(
  query: string,
  onChangeQuery: (next: string) => void,
  durationRequest?: DurationRequest | null,
) {
  // intentionally computed once, from the query the hook was mounted with: initialFilters only ever
  // seeds useState/useRef initial values below, which already ignore later updates, and re-parsing on
  // every 'query' change would fight with the ownClause->onChangeQuery sync effect further down
  // eslint-disable-next-line @eslint-react/exhaustive-deps
  const initialFilters = useMemo(() => parseFiltersFromQuery(query), []);

  const [selectedServices, setSelectedServices] = useState<string[]>(initialFilters.services);
  const [selectedOperations, setSelectedOperations] = useState<string[]>(initialFilters.operations);
  const [minDuration, setMinDuration] = useState(initialFilters.minDuration);
  const [maxDuration, setMaxDuration] = useState(initialFilters.maxDuration);
  const [selectedKinds, setSelectedKinds] = useState<string[]>(initialFilters.kinds);
  const [tagName, setTagName] = useState("");
  const [tagValue, setTagValue] = useState("");
  const [tags, setTags] = useState<Tag[]>(initialFilters.tags);

  const toggleService = useCallback((value: string) => {
    setSelectedServices(prev => toggleValue(prev, value));
  }, []);

  const toggleOperation = useCallback((value: string) => {
    setSelectedOperations(prev => toggleValue(prev, value));
  }, []);

  const toggleKind = useCallback((value: string) => {
    setSelectedKinds(prev => toggleValue(prev, value));
  }, []);

  const selectTagName = useCallback((value: string) => {
    setTagName(value);
    setTagValue("");
  }, []);

  const addTag = useCallback((field: string) => {
    if (!tagName || !tagValue || !field) return;
    setTags(prev => (
      prev.some(t => t.name === tagName && t.value === tagValue)
        ? prev
        : [...prev, { name: tagName, value: tagValue, field }]
    ));
    setTagName("");
    setTagValue("");
  }, [tagName, tagValue]);

  const removeTag = useCallback((index: number) => {
    setTags(prev => prev.filter((_, i) => i !== index));
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

  const ownClause = useMemo(() => {
    const clauses = [
      buildInClause("resource_attr:service.name", selectedServices),
      buildInClause("name", selectedOperations),
      buildDurationClause(minDuration, maxDuration),
      buildInClause("kind", mapSpanKindsToCodes(selectedKinds)),
      ...tags.map(t => buildTagClause(t.field, t.value)),
    ].filter(Boolean);
    return clauses.join(" AND ");
  }, [selectedServices, selectedOperations, minDuration, maxDuration, selectedKinds, tags]);

  const queryRef = useRef(initialFilters.remainder);
  const prevOwnClauseRef = useRef("");

  useEffect(() => {
    if (ownClause === prevOwnClauseRef.current) return;

    const next = mergeQuery(ownClause, prevOwnClauseRef.current, queryRef.current);
    prevOwnClauseRef.current = ownClause;

    if (next !== queryRef.current.trim()) {
      onChangeQuery(next);
    }
  }, [ownClause, onChangeQuery]);

  useEffect(() => {
    queryRef.current = query;
  }, [query]);

  return {
    selectedServices,
    toggleService,
    selectedOperations,
    toggleOperation,
    minDuration,
    setMinDuration,
    maxDuration,
    setMaxDuration,
    selectedKinds,
    toggleKind,
    tagName,
    setTagName: selectTagName,
    tagValue,
    setTagValue,
    tags,
    addTag,
    removeTag,
  };
}
