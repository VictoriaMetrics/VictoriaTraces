import { DATE_TIME_FORMAT } from "./date";

export const TRACES_ENTRIES_LIMIT = 50;
export const TRACES_BARS_VIEW = 100;
export const TRACES_LIMIT_HITS = 5;

// "Ungrouped" is a string that is used as a value for the "groupBy" parameter.
export const WITHOUT_GROUPING = "Ungrouped";

// Default values for the traces configurators.
export const TRACES_GROUP_BY = "_stream";
export const TRACES_DISPLAY_FIELDS = "_msg";
export const TRACES_DATE_FORMAT = `${DATE_TIME_FORMAT}.SSS`;

// URL parameters for the traces page.
export const TRACES_URL_PARAMS = {
  GROUP_BY: "groupBy",
  DISPLAY_FIELDS: "displayFields",
  NO_WRAP_LINES: "noWrapLines",
  COMPACT_GROUP_HEADER: "compactGroupHeader",
  DATE_FORMAT: "dateFormat",
  ROWS_PER_PAGE: "rows_per_page",
};
