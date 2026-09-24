import { useMemo } from "preact/compat";
import { vmDate } from "../../../../../utils/time";
import { type Column } from "../../../../../components/Table/types";
import { TraceSummary } from "../../../hooks/useLogsqlTracesSearch";

export const ALL_TRACES_COLUMN_KEYS = [
  "traceID",
  "service",
  "operation",
  "duration",
  "spanCount",
  "errorCount",
  "startTime",
];

const formatDurationUs = (durationUs: number): string => {
  if (durationUs >= 1_000_000) return `${(durationUs / 1_000_000).toFixed(2)}s`;
  if (durationUs >= 1_000) return `${(durationUs / 1_000).toFixed(2)}ms`;
  return `${durationUs}µs`;
};

const getRenderColumnByKey = (key: string): Column<TraceSummary>["render"] => {
  switch (key) {
    case "duration":
      return row => formatDurationUs(row.duration);
    case "spanCount":
      return row => row.spanCount;
    case "errorCount":
      return row => row.errorCount;
    case "startTime":
      return row => vmDate(row.startTime / 1000).nano().format("YYYY-MM-DD HH:mm:ss.SSS");
    default:
      return undefined;
  }
};

const TITLE_BY_KEY: Record<string, string> = {
  traceID: "Trace ID",
  service: "Service",
  operation: "Operation",
  duration: "Duration",
  spanCount: "Spans",
  errorCount: "Errors",
  startTime: "Start time",
};

const getBaseColumn = (key: string, isLast: boolean): Column<TraceSummary> => ({
  key: key as keyof TraceSummary,
  title: TITLE_BY_KEY[key] || key,
  className: isLast ? "vm-table-cell_full" : undefined,
  classNameHeader: isLast ? "vm-table-cell-header_full" : undefined,
  options: {
    sortable: true,
    resizable: true,
    draggable: !isLast,
    menuEnabled: true,
  },
  render: getRenderColumnByKey(key),
});

type Options = {
  keys: readonly string[];
};

export const useTableTracesColumns = ({ keys }: Options) => {
  const tableColumns = useMemo(() => {
    const orderedKeys = ALL_TRACES_COLUMN_KEYS.filter(key => keys.includes(key));
    return orderedKeys.map((key, idx) => getBaseColumn(key, idx === orderedKeys.length - 1));
  }, [keys]);

  return { tableColumns };
};
