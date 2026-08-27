import { useEffect, useState } from "preact/compat";
import { useAppState } from "../../../state/common/StateContext";
import { useTenant } from "../../../hooks/useTenant";
import { useTimePeriod } from "./useTimePeriod";
import { AUTOCOMPLETE_LIMITS } from "../../../constants/queryAutocomplete";
import { LogsFieldValues } from "../../../api/types";
import { FieldNameGroup, groupIndexedFieldNames, TAGS_EXCLUDED_FIELDS } from "../utils";

export function useFieldNames() {
  const { serverUrl } = useAppState();
  const tenant = useTenant();
  const { period } = useTimePeriod();

  const [groups, setGroups] = useState<FieldNameGroup[]>([]);
  const [loading, setLoading] = useState(false);

  const start = Number(period.start / 1_000_000_000n);
  const end = Number(period.end / 1_000_000_000n);

  useEffect(() => {
    if (!serverUrl) return;

    const controller = new AbortController();

    const params = new URLSearchParams({
      query: "*",
      start: `${start}`,
      end: `${end}`,
      limit: `${AUTOCOMPLETE_LIMITS.queryLimit}`,
    });

    // eslint-disable-next-line @eslint-react/set-state-in-effect -- tracks the loading state of the fetch this effect triggers
    setLoading(true);
    fetch(`${serverUrl}/select/logsql/field_names?${params}`, { signal: controller.signal, headers: { ...tenant } })
      .then(res => (res.ok ? res.json() : Promise.reject(res)))
      .then(data => {
        const fieldNames = (data?.values || []) as LogsFieldValues[];
        const names = fieldNames.map(v => v.value).filter(v => v && !TAGS_EXCLUDED_FIELDS.has(v));
        setGroups(groupIndexedFieldNames(names));
      })
      .catch(e => {
        if (e instanceof Error && e.name !== "AbortError") console.error(e);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });

    return () => controller.abort();
  }, [serverUrl, tenant, start, end]);

  return { groups, loading };
}
