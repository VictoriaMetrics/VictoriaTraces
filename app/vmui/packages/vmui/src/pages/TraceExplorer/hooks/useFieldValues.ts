import { useEffect, useState } from "preact/compat";
import { useAppState } from "../../../state/common/StateContext";
import { useTenant } from "../../../hooks/useTenant";
import { useTimePeriod } from "./useTimePeriod";
import { AUTOCOMPLETE_LIMITS } from "../../../constants/queryAutocomplete";
import { LogsFieldValues } from "../../../api/types";

export function useFieldValues(field: string) {
  const { serverUrl } = useAppState();
  const tenant = useTenant();
  const { period } = useTimePeriod();

  const [values, setValues] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);

  const start = Number(period.start / 1_000_000_000n);
  const end = Number(period.end / 1_000_000_000n);

  useEffect(() => {
    if (!serverUrl || !field) return;

    const controller = new AbortController();

    const params = new URLSearchParams({
      query: "*",
      field,
      start: `${start}`,
      end: `${end}`,
      limit: `${AUTOCOMPLETE_LIMITS.queryLimit}`,
    });

    // eslint-disable-next-line @eslint-react/set-state-in-effect -- tracks the in-flight fetch triggered by this effect's own dependencies
    setLoading(true);
    fetch(`${serverUrl}/select/logsql/field_values?${params}`, { signal: controller.signal, headers: { ...tenant } })
      .then(res => (res.ok ? res.json() : Promise.reject(res)))
      .then(data => {
        const fieldValues = (data?.values || []) as LogsFieldValues[];
        setValues(fieldValues.map(v => v.value).filter(Boolean));
      })
      .catch(e => {
        if (e instanceof Error && e.name !== "AbortError") console.error(e);
      })
      .finally(() => setLoading(false));

    return () => controller.abort();
  }, [serverUrl, field, tenant, start, end]);

  return { values, loading };
}
