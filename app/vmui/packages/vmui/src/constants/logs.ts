// LogsQL is one shared query language across VictoriaLogs and VictoriaTraces,
// so this URL intentionally points at the VictoriaLogs docs domain.
export const LOGSQL_DOCS_URL = "https://docs.victoriametrics.com/victorialogs/logsql/";

// Multitenancy is documented on the VictoriaLogs docs site; no VictoriaTraces-specific
// page exists for it, so this intentionally points at the VictoriaLogs docs domain too.
export const VICTORIALOGS_DOCS_URL = "https://docs.victoriametrics.com/victorialogs";

export const TRACES_DEFAULT_LIMIT = 5000;
export const TRACES_MAX_LIMIT = 10000;

// URL parameters for the traces page.
export const TRACES_URL_PARAMS = {
  LIMIT: "limit",
  ROWS_PER_PAGE: "rows_per_page",
  COLUMNS: "columns",
};

// Maximum values for the logs autocomplete.
export const MAX_QUERY_FIELDS = 1;
export const MAX_QUERIES_HISTORY = 25;

// Default fields for the table.
export const DEFAULT_COMMON_FIELDS = ["_time", "_msg" ];
