import { useEffect, useRef } from "preact/compat";
import { TraceChartRun } from "./types";

/**
 * Calls `load` for an active chart once per committed run.
 * An inactive (hidden) chart skips runs and catches up with the latest one when it becomes active,
 * so switching tabs never re-fetches data that is already current.
 */
export function useChartRun(
  isActive: boolean,
  run: TraceChartRun | null,
  load: (run: TraceChartRun) => void,
): void {
  const lastRunIdRef = useRef<number | null>(null);

  useEffect(() => {
    if (!isActive || !run || lastRunIdRef.current === run.id) return;
    lastRunIdRef.current = run.id;
    load(run);
  }, [isActive, run, load]);
}
