import {
  ApiError,
  AttributeValue,
  Attribute,
  SpanEvent,
  OtelSpan,
  OtelTrace,
  Status,
  KeyValuePair,
  Span,
  SpanKind,
  StatusCode,
  TNil,
  Trace,
} from "../types";
import colorGenerator from "../../../utils/color-generator";

export interface Tag {
  name: string;
  value: string;
  field: string;
}

export const SPAN_KIND_OPTIONS = ["SERVER", "CLIENT", "PRODUCER", "CONSUMER", "INTERNAL"];

const SPAN_KIND_TO_CODE: Record<string, string> = {
  INTERNAL: "1",
  SERVER: "2",
  CLIENT: "3",
  PRODUCER: "4",
  CONSUMER: "5",
};

export const SERVICE_FIELD = "resource_attr:service.name";
export const OPERATION_FIELD = "name";

export const TAGS_EXCLUDED_FIELDS = new Set([
  "_time",
  "name",
  "trace_id",
  "span_id",
  "parent_span_id",
  "duration",
  "start_time_unix_nano",
  "end_time_unix_nano",
  "kind",
  SERVICE_FIELD,
]);

const DURATION_PATTERN = /^\d+(\.\d+)?(ns|us|ms|s|m|h)$/;
const DURATION_UNIT_TO_MS: Record<string, number> = { ns: 0.000001, us: 0.001, ms: 1, s: 1000, m: 60_000, h: 3_600_000 };
export const DURATION_PLACEHOLDER = "e.g. 100ms, 1.2s";

export function parseDurationMs(value: string): number | null {
  const match = value.match(/^(\d+(?:\.\d+)?)(ns|us|ms|s|m|h)$/);
  if (!match) return null;
  const unit = match[2];
  return Number(match[1]) * DURATION_UNIT_TO_MS[unit];
}

export function validateDurationInput(value: string): string {
  if (!value) return "";
  return DURATION_PATTERN.test(value) ? "" : `Format: ${DURATION_PLACEHOLDER}`;
}

export function quoteLogsqlValue(value: string): string {
  return JSON.stringify(value);
}

const SIMPLE_FIELD_NAME = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

// A field name containing `:` (e.g. resource_attr:service.name) is ambiguous with LogsQL's
// own `field:value` separator unless quoted — plain identifiers don't need it, but quoting
// them too is harmless, so only the ones that actually need it pay for it.
function formatLogsqlField(field: string): string {
  return SIMPLE_FIELD_NAME.test(field) ? field : quoteLogsqlValue(field);
}

export function mapSpanKindsToCodes(kinds: string[]): string[] {
  return kinds.map(kind => SPAN_KIND_TO_CODE[kind] ?? kind);
}

const CODE_TO_SPAN_KIND: Record<string, string> = Object.fromEntries(
  Object.entries(SPAN_KIND_TO_CODE).map(([kind, code]) => [code, kind])
);

export function mapCodesToSpanKinds(codes: string[]): string[] {
  return codes.map(code => CODE_TO_SPAN_KIND[code] ?? code);
}

// Inverse of buildDurationClause's ns math: picks the largest unit that keeps the number
// clean-ish, matching what a user would plausibly have typed into the Duration inputs.
export function formatDurationForInput(ns: number): string {
  const ms = ns / 1_000_000;
  const trim = (n: number) => Number(n.toFixed(3)).toString();

  if (ms >= DURATION_UNIT_TO_MS.h) return `${trim(ms / DURATION_UNIT_TO_MS.h)}h`;
  if (ms >= DURATION_UNIT_TO_MS.m) return `${trim(ms / DURATION_UNIT_TO_MS.m)}m`;
  if (ms >= DURATION_UNIT_TO_MS.s) return `${trim(ms / DURATION_UNIT_TO_MS.s)}s`;
  if (ms >= 1) return `${trim(ms)}ms`;
  if (ms >= DURATION_UNIT_TO_MS.us) return `${trim(ms / DURATION_UNIT_TO_MS.us)}us`;
  return `${trim(ms / DURATION_UNIT_TO_MS.ns)}ns`;
}

export function buildInClause(field: string, values: string[]): string {
  if (!values.length) return "";
  return `${formatLogsqlField(field)}:in(${values.map(quoteLogsqlValue).join(",")})`;
}

export function buildDurationClause(minValue: string, maxValue: string): string {
  const parts: string[] = [];
  const minMs = parseDurationMs(minValue);
  const maxMs = parseDurationMs(maxValue);
  // duration is stored in nanoseconds.
  if (minMs !== null) parts.push(`duration:>=${Math.round(minMs * 1_000_000)}`);
  if (maxMs !== null) parts.push(`duration:<=${Math.round(maxMs * 1_000_000)}`);
  return parts.join(" AND ");
}

// Converts a [lowUs, highUs) duration range (as bucketed by the heatmap) into the same
// input-string shape the Duration section's fields use, so a heatmap selection can be
// applied through the exact same code path as a user typing into those fields.
export function formatDurationRangeForInput(lowUs: number, highUs: number): { min: string; max: string } {
  return {
    min: lowUs > 0 ? formatDurationForInput(lowUs * 1000) : "",
    max: Number.isFinite(highUs) ? formatDurationForInput(highUs * 1000) : "",
  };
}

export function buildTagClause(field: string, tagValue: string): string {
  if (!field || !tagValue) return "";
  return `${formatLogsqlField(field)}:${quoteLogsqlValue(tagValue)}`;
}

export interface FieldNameGroup {
  displayName: string;
  realNames: string[];
  isIndexed: boolean;
}

const INDEXED_FIELD_SUFFIX = /^(.+):(\d+)$/;

export function groupQueryField(group: FieldNameGroup): string {
  return group.isIndexed ? `${group.displayName}:*` : (group.realNames[0] ?? group.displayName);
}

export function groupIndexedFieldNames(names: string[]): FieldNameGroup[] {
  const groups = new Map<string, string[]>();
  for (const name of names) {
    const key = name.match(INDEXED_FIELD_SUFFIX)?.[1] ?? name;
    const list = groups.get(key);
    if (list) list.push(name);
    else groups.set(key, [name]);
  }

  return Array.from(groups.entries()).map(([displayName, realNames]) => ({
    displayName,
    isIndexed: INDEXED_FIELD_SUFFIX.test(realNames[0]),
    realNames: realNames.sort((a, b) => {
      const indexA = Number(a.match(INDEXED_FIELD_SUFFIX)?.[2]);
      const indexB = Number(b.match(INDEXED_FIELD_SUFFIX)?.[2]);
      return Number.isFinite(indexA) && Number.isFinite(indexB) ? indexA - indexB : 0;
    }),
  }));
}

export function toggleValue(list: string[], value: string): string[] {
  return list.includes(value) ? list.filter(v => v !== value) : [...list, value];
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

const DURATION_CLAUSE_PATTERN = /^duration:[<>]=\d+$/;
const KNOWN_IN_CLAUSE_PATTERN = new RegExp(
  `^(?:${escapeRegExp(formatLogsqlField(SERVICE_FIELD))}|${OPERATION_FIELD}|kind):in\\(.*\\)$`
);

function isKnownFilterClause(part: string): boolean {
  return DURATION_CLAUSE_PATTERN.test(part) || KNOWN_IN_CLAUSE_PATTERN.test(part);
}

function stripStaleFilterClauses(text: string): string {
  if (!text) return text;
  return text
    .split(/\s+AND\s+/)
    .filter(part => !isKnownFilterClause(part.trim()))
    .join(" AND ");
}

export function mergeQuery(ownClause: string, previousOwnClause: string, currentQuery: string): string {
  const trimmedCurrent = currentQuery.trim();
  let remainder = trimmedCurrent;

  if (previousOwnClause) {
    const prefix = `${previousOwnClause} AND `;
    if (trimmedCurrent.startsWith(prefix)) {
      remainder = trimmedCurrent.slice(prefix.length).trim();
    } else if (trimmedCurrent === previousOwnClause) {
      remainder = "";
    }
  }

  remainder = stripStaleFilterClauses(remainder);

  return [ownClause, remainder].filter(Boolean).join(" AND ");
}

function parseFieldToken(token: string): string {
  try {
    return token.startsWith("\"") ? JSON.parse(token) : token;
  } catch {
    return token;
  }
}

function parseInValues(inner: string): string[] {
  try {
    const parsed = JSON.parse(`[${inner}]`);
    return Array.isArray(parsed) ? parsed.filter((v): v is string => typeof v === "string") : [];
  } catch {
    return [];
  }
}

const IN_CLAUSE_RE = /^(".*?"|[a-zA-Z_][a-zA-Z0-9_.]*):in\((.*)\)$/;
const EQ_CLAUSE_RE = /^(".*?"|[a-zA-Z_][a-zA-Z0-9_.]*):(".*")$/;
const WILDCARD_FIELD_SUFFIX = /:\*$/;
const DURATION_GE_RE = /^duration:>=(\d+)$/;
const DURATION_LE_RE = /^duration:<=(\d+)$/;

export interface ParsedFilters {
  services: string[];
  operations: string[];
  minDuration: string;
  maxDuration: string;
  kinds: string[];
  tags: Tag[];
  remainder: string;
}

// The inverse of ownClause's construction: given a query that may already contain filter
// clauses (e.g. the page was opened with `?logsql_query=...` already populated, from a
// shared link, Query History, or Query Examples), recover the structured selections those
// clauses represent so the Filters sidebar can start pre-populated instead of blank.
export function parseFiltersFromQuery(query: string): ParsedFilters {
  const trimmed = query.trim();
  const parts = trimmed ? trimmed.split(/\s+AND\s+/) : [];

  const services: string[] = [];
  const operations: string[] = [];
  const kinds: string[] = [];
  const tags: Tag[] = [];
  let minDuration = "";
  let maxDuration = "";
  const remainderParts: string[] = [];

  for (const rawPart of parts) {
    const part = rawPart.trim();
    if (!part) continue;

    let match = part.match(DURATION_GE_RE);
    if (match) {
      minDuration = formatDurationForInput(Number(match[1]));
      continue;
    }

    match = part.match(DURATION_LE_RE);
    if (match) {
      maxDuration = formatDurationForInput(Number(match[1]));
      continue;
    }

    match = part.match(IN_CLAUSE_RE);
    if (match) {
      const field = parseFieldToken(match[1]);
      const values = parseInValues(match[2]);
      if (field === SERVICE_FIELD) {
        services.push(...values);
        continue;
      }
      if (field === OPERATION_FIELD) {
        operations.push(...values);
        continue;
      }
      if (field === "kind") {
        kinds.push(...mapCodesToSpanKinds(values));
        continue;
      }
      remainderParts.push(part);
      continue;
    }

    match = part.match(EQ_CLAUSE_RE);
    if (match) {
      const field = parseFieldToken(match[1]);
      if (!TAGS_EXCLUDED_FIELDS.has(field)) {
        try {
          const name = field.replace(WILDCARD_FIELD_SUFFIX, "");
          tags.push({ name, value: JSON.parse(match[2]), field });
          continue;
        } catch {
          // not a valid quoted value after all — fall through to remainder
        }
      }
    }

    remainderParts.push(part);
  }

  return {
    services,
    operations,
    minDuration,
    maxDuration,
    kinds,
    tags,
    remainder: remainderParts.join(" AND "),
  };
}

// ============================================================
// Trace/Span conversion: legacy `Trace`/`Span` -> plain `OtelTrace`/`OtelSpan`
// ============================================================

function toOtelAttributes(tags: ReadonlyArray<KeyValuePair<unknown>>): Attribute[] {
  return tags
    .filter(kv => kv.value !== null && kv.value !== undefined)
    .map(kv => ({ key: kv.key, value: kv.value as AttributeValue }));
}

function toOtelSpan(span: Span): OtelSpan {
  const kindTag = span.tags.find(t => t.key === "span.kind");
  let kind = SpanKind.INTERNAL;
  if (kindTag) {
    const val = String(kindTag.value).toUpperCase();
    if (val in SpanKind) kind = SpanKind[val as keyof typeof SpanKind];
  }

  const parentSpanRef =
    span.references.find(r => r.traceID === span.traceID && r.refType === "CHILD_OF") ??
    span.references.find(r => r.traceID === span.traceID && r.refType === "FOLLOWS_FROM");

  const errorTag = span.tags.find(t => t.key === "error");
  const status: Status = errorTag && errorTag.value
    ? { code: StatusCode.ERROR, message: "error" }
    : { code: StatusCode.OK };

  return {
    traceID: span.traceID,
    spanID: span.spanID,
    parentSpanID: parentSpanRef?.spanID,
    parentSpan: undefined, // wired up in toOtelTrace's second pass
    name: span.operationName,
    kind,
    startTime: span.startTime as OtelSpan["startTime"],
    endTime: (span.startTime + span.duration) as OtelSpan["endTime"],
    duration: span.duration as OtelSpan["duration"],
    attributes: toOtelAttributes(span.tags),
    events: span.logs.map(log => ({
      timestamp: log.timestamp as SpanEvent["timestamp"],
      name: (log.fields.find(f => f.key === "event")?.value as string) || "log",
      // "event" itself is already surfaced as `name` above - don't duplicate it as an attribute.
      attributes: toOtelAttributes(log.fields.filter(f => f.key !== "event")),
    })),
    links: span.links.map(link => ({
      traceID: link.traceID,
      spanID: link.spanID,
      attributes: toOtelAttributes(link.attributes),
    })),
    status,
    resource: {
      attributes: span.process ? toOtelAttributes(span.process.tags) : [],
      serviceName: span.process ? span.process.serviceName : "unknown-service",
    },
    depth: span.depth,
    hasChildren: false, // wired up in toOtelTrace's second pass
    childSpans: [], // wired up in toOtelTrace's second pass
    relativeStartTime: span.relativeStartTime as OtelSpan["relativeStartTime"],
  };
}

// Converts the app's internal `Trace`/`Span` shape (tags deduplicated/ordered, parent/child
// references resolved by transformTraceData) into the plain `OtelTrace`/`OtelSpan` shape the
// rest of TraceView consumes. Done in two passes since `parentSpan`/`childSpans` are circular
// references — every span is converted once, then those pointers are wired up against the
// resulting spanMap.
export function toOtelTrace(trace: Trace): OtelTrace {
  const spans = trace.spans.map(toOtelSpan);
  const spanMap = new Map<string, OtelSpan>();
  spans.forEach(span => spanMap.set(span.spanID, span));

  spans.forEach((span, i) => {
    if (span.parentSpanID) {
      span.parentSpan = spanMap.get(span.parentSpanID);
    }
    const childSpans = trace.spans[i].childSpans
      .map(s => spanMap.get(s.spanID))
      .filter((s): s is OtelSpan => Boolean(s));
    span.childSpans = childSpans;
    span.hasChildren = childSpans.length > 0;
  });

  const rootSpans = trace.rootSpans.map(s => {
    const otelSpan = spanMap.get(s.spanID);
    if (!otelSpan) throw new Error(`Root span ${s.spanID} not found in spanMap`);
    return otelSpan;
  });

  return {
    traceID: trace.traceID,
    spans,
    duration: trace.duration,
    startTime: trace.startTime,
    endTime: trace.endTime,
    spanMap,
    rootSpans,
  };
}

// ============================================================
// API error formatting
// ============================================================

const MAX_API_ERROR_DETAIL_LENGTH = 1024;

export type ApiErrorDetails = {
  title: string;
  rows: { name: string; value: string | number }[];
};

export function formatApiError(error: ApiError): ApiErrorDetails {
  if (typeof error === "string") {
    return { title: error, rows: [] };
  }

  const { message, httpStatus, httpStatusText, httpUrl, httpQuery, httpBody } = error;
  const bodyExcerpt =
    httpBody && httpBody.length > MAX_API_ERROR_DETAIL_LENGTH
      ? `${httpBody.slice(0, MAX_API_ERROR_DETAIL_LENGTH - 3).trim()}...`
      : httpBody;

  const rows: ApiErrorDetails["rows"] = [];
  if (httpStatus) rows.push({ name: "Status", value: httpStatus });
  if (httpStatusText) rows.push({ name: "Status text", value: httpStatusText });
  if (httpUrl) rows.push({ name: "URL", value: httpUrl });
  if (httpQuery) rows.push({ name: "Query", value: httpQuery });
  if (bodyExcerpt) rows.push({ name: "Response body", value: bodyExcerpt });

  return { title: message, rows };
}

// ============================================================
// span_id URL param
// ============================================================

export const SPAN_ID_URL_PARAM = "span_id";

export function getSpanIdFromSearchParams(searchParams: URLSearchParams): { spanId: string | undefined } {
  return { spanId: searchParams.get(SPAN_ID_URL_PARAM) || undefined };
}

// ============================================================
// Span search filtering
// ============================================================

export default function filterSpans(textFilter: string, spans: ReadonlyArray<OtelSpan> | TNil) {
  if (!spans) {
    return null;
  }

  // if a span field includes at least one filter in includeFilters, the span is a match
  const includeFilters: string[] = [];

  // values with keys that include text in any one of the excludeKeys will be ignored
  const excludeKeys: string[] = [];

  // split textFilter by whitespace, but not that in double quotes, remove empty strings, and extract includeFilters and excludeKeys
  const regex = /[^\s"]+|"([^"]*)"/g;
  const match = textFilter.match(regex);
  const results = match ? match.map(e => e.replace(/"(.*)"/, "$1")) : [];

  results.filter(Boolean).forEach(w => {
    if (w[0] === "-") {
      excludeKeys.push(w.substr(1).toLowerCase());
    } else {
      includeFilters.push(w.toLowerCase());
    }
  });

  const isTextInFilters = (filters: Array<string>, text: string) =>
    filters.some(filter => text.toLowerCase().includes(filter));

  const isTextInKeyValues = (kvs: ReadonlyArray<Attribute>) =>
    kvs
      ? kvs.some(kv => {
        // ignore checking key and value for a match if key is in excludeKeys
        if (isTextInFilters(excludeKeys, kv.key)) return false;
        const valueString = String(kv.value);
        // match if key, value or key=value string matches an item in includeFilters
        return (
          isTextInFilters(includeFilters, kv.key) ||
            isTextInFilters(includeFilters, valueString) ||
            isTextInFilters(includeFilters, `${kv.key}=${valueString}`)
        );
      })
      : false;

  const isSpanAMatch = (span: OtelSpan) =>
    isTextInFilters(includeFilters, span.name) ||
    isTextInFilters(includeFilters, span.resource.serviceName) ||
    isTextInKeyValues(span.attributes) ||
    (Array.isArray(span.events) && span.events.some(event => isTextInKeyValues(event.attributes))) ||
    isTextInKeyValues(span.resource.attributes) ||
    includeFilters.some(filter => filter.replace(/^0*/, "") === span.spanID.replace(/^0*/, ""));

  // declare as const because need to disambiguate the type
  const rv: Set<string> = new Set(spans.filter(isSpanAMatch).map((span: OtelSpan) => span.spanID));
  return rv;
}

// ============================================================
// RPC pairing
// ============================================================

export type RpcInfo = {
  viewStart: number;
  viewEnd: number;
  color: string;
  operationName: string;
  serviceName: string;
};

// A CLIENT/PRODUCER span's row gets annotated with the SERVER/CONSUMER child span it invoked,
// when that downstream service is instrumented and present in this same trace.
export function computeRpc(
  span: OtelSpan,
  getViewedBounds: ViewedBoundsFunctionType,
  isDarkTheme: boolean
): RpcInfo | null {
  if (span.kind !== SpanKind.CLIENT && span.kind !== SpanKind.PRODUCER) return null;

  const serverSpan = span.childSpans.find(
    child => child.kind === SpanKind.SERVER || child.kind === SpanKind.CONSUMER
  );
  if (!serverSpan) return null;

  const { start, end } = getViewedBounds(serverSpan.startTime, serverSpan.endTime);
  return {
    viewStart: start,
    viewEnd: end,
    color: colorGenerator.getColorByKey(serverSpan.resource.serviceName, isDarkTheme),
    operationName: serverSpan.name,
    serviceName: serverSpan.resource.serviceName,
  };
}

export type ViewedBoundsFunctionType = (start: number, end: number) => { start: number; end: number };
/**
 * Given a range (`min`, `max`) and factoring in a zoom (`viewStart`, `viewEnd`)
 * a function is created that will find the position of a sub-range (`start`, `end`).
 * The calling the generated method will return the result as a `{ start, end }`
 * object with values ranging in [0, 1].
 *
 * @param  {number} min       The start of the outer range.
 * @param  {number} max       The end of the outer range.
 * @param  {number} viewStart The start of the zoom, on a range of [0, 1],
 *                            relative to the `min`, `max`.
 * @param  {number} viewEnd   The end of the zoom, on a range of [0, 1],
 *                            relative to the `min`, `max`.
 * @returns {(number, number) => Object} Created view bounds function
 */
export function createViewedBoundsFunc(viewRange: {
  min: number;
  max: number;
  viewStart: number;
  viewEnd: number;
}) {
  const { min, max, viewStart, viewEnd } = viewRange;
  const duration = max - min;
  const viewMin = min + viewStart * duration;
  const viewMax = max - (1 - viewEnd) * duration;
  const viewWindow = viewMax - viewMin;

  /**
   * View bounds function
   * @param  {number} start     The start of the sub-range.
   * @param  {number} end       The end of the sub-range.
   * @return {Object}           The resultant range.
   */
  return (start: number, end: number) => ({
    start: (start - viewMin) / viewWindow,
    end: (end - viewMin) / viewWindow,
  });
}

/**
 * Returns `true` if the span has an error status.
 *
 * @param  {OtelSpan} span  The OTEL span to check.
 * @return {boolean}         True if the span has an error status.
 */
export function isErrorSpan(span: OtelSpan): boolean {
  return span.status.code === StatusCode.ERROR;
}

/**
 * Returns `true` if at least one of the descendants of the `parentSpanIndex`
 * span contains an error status.
 *
 * @param      {OtelSpan[]}  spans            The OTEL spans for a trace - should be
 *                                             sorted with children following parents.
 * @param      {number}       parentSpanIndex  The index of the parent span - only
 *                                             subsequent spans with depth less than
 *                                             the parent span will be checked.
 * @return     {boolean}      Returns `true` if a descendant contains an error status.
 */
export const isKindClient = (span: OtelSpan): boolean => span.kind === SpanKind.CLIENT;

export const isKindProducer = (span: OtelSpan): boolean => span.kind === SpanKind.PRODUCER;
