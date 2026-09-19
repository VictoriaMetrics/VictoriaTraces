import { useCallback, useMemo } from "preact/compat";
import { useSearchParams } from "react-router-dom";
import { buildInClause } from "../utils";

export interface ExtraFilter {
  field: string;
  value: string;
}

const EXTRA_FILTERS_KEY = "extra_filters";
export const DURATION_MIN_FIELD = "duration_min";
export const DURATION_MAX_FIELD = "duration_max";

function buildClauseForField(field: string, values: string[]): string {
  if (field === DURATION_MIN_FIELD) return values[0] ? `duration:>=${values[0]}` : "";
  if (field === DURATION_MAX_FIELD) return values[0] ? `duration:<=${values[0]}` : "";
  return buildInClause(field, values);
}

function parseExtraFilters(searchParams: URLSearchParams): ExtraFilter[] {
  return searchParams.getAll(EXTRA_FILTERS_KEY).flatMap(param => {
    try {
      const obj = JSON.parse(param);
      if (!obj || typeof obj !== "object") return [];
      const { f, v } = obj as Record<string, unknown>;
      if (typeof f !== "string" || typeof v !== "string") return [];
      return [{ field: f, value: v }];
    } catch {
      return [];
    }
  });
}

function serializeExtraFilters(searchParams: URLSearchParams, filters: ExtraFilter[]): URLSearchParams {
  const next = new URLSearchParams(searchParams);
  next.delete(EXTRA_FILTERS_KEY);
  filters.forEach(f => next.append(EXTRA_FILTERS_KEY, JSON.stringify({ f: f.field, v: f.value })));
  return next;
}

export function useExtraFilters() {
  const [searchParams, setSearchParams] = useSearchParams();

  const extraFilters = useMemo(() => parseExtraFilters(searchParams), [searchParams]);

  const extraParams = useMemo(() => {
    const params = new URLSearchParams();
    const valuesByField = new Map<string, string[]>();
    extraFilters.forEach(f => {
      valuesByField.set(f.field, [...(valuesByField.get(f.field) || []), f.value]);
    });
    valuesByField.forEach((values, field) => {
      const clause = buildClauseForField(field, values);
      if (clause) params.append(EXTRA_FILTERS_KEY, clause);
    });
    return params;
  }, [extraFilters]);

  const updateFilters = useCallback((updater: (current: ExtraFilter[]) => ExtraFilter[]) => {
    setSearchParams(prev => serializeExtraFilters(prev, updater(parseExtraFilters(prev))));
  }, [setSearchParams]);

  const toggleFilter = useCallback((field: string, value: string) => {
    updateFilters(current => (
      current.some(f => f.field === field && f.value === value)
        ? current.filter(f => !(f.field === field && f.value === value))
        : [...current, { field, value }]
    ));
  }, [updateFilters]);

  const addFilter = useCallback((field: string, value: string) => {
    updateFilters(current => (
      current.some(f => f.field === field && f.value === value)
        ? current
        : [...current, { field, value }]
    ));
  }, [updateFilters]);

  const removeFilter = useCallback((field: string, value: string) => {
    updateFilters(current => current.filter(f => !(f.field === field && f.value === value)));
  }, [updateFilters]);

  const setFieldValue = useCallback((field: string, value: string) => {
    updateFilters(current => {
      const others = current.filter(f => f.field !== field);
      return value ? [...others, { field, value }] : others;
    });
  }, [updateFilters]);

  const selectedValues = useCallback((field: string) => (
    extraFilters.filter(f => f.field === field).map(f => f.value)
  ), [extraFilters]);

  return { extraFilters, extraParams, toggleFilter, addFilter, removeFilter, setFieldValue, selectedValues };
}
