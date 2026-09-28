import { useCallback, useRef, useState } from "preact/compat";
import _isEqual from "lodash/isEqual";
import { getTempoTraceByIdUrl } from "../../../api/tempo";
import { useAppState } from "../../../state/common/StateContext";
import { useTenant } from "../../../hooks/useTenant";
import { toOtelTrace } from "../utils";
import { OtelTrace, KeyValuePair, Process, Span, SpanData, SpanLink, SpanReference, Trace, TraceData } from "../types";

const REQUEST_TIMEOUT_MS = 30_000;

// Minimal OTLP/JSON shapes for the fields TraceView actually consumes, matching the
// server's Tempo v2 trace-by-ID response (app/vtselect/traces/tempo/tempo.qtpl).
interface OtlpAnyValue {
  stringValue?: string;
  boolValue?: boolean;
  intValue?: string;
  doubleValue?: number;
  bytesValue?: string;
  arrayValue?: { values: OtlpAnyValue[] };
  kvlistValue?: { values: OtlpKeyValue[] };
}

interface OtlpKeyValue {
  key: string;
  value?: OtlpAnyValue;
}

interface OtlpStatus {
  code?: string;
  message?: string;
}

interface OtlpLink {
  traceId: string;
  spanId: string;
  attributes?: OtlpKeyValue[];
}

interface OtlpEvent {
  timeUnixNano: string;
  name: string;
  attributes?: OtlpKeyValue[];
}

interface OtlpSpan {
  traceId: string;
  spanId: string;
  parentSpanId?: string;
  name: string;
  kind?: string;
  startTimeUnixNano: string;
  endTimeUnixNano: string;
  attributes?: OtlpKeyValue[];
  status?: OtlpStatus;
  links?: OtlpLink[];
  events?: OtlpEvent[];
}

interface OtlpScopeSpans {
  spans?: OtlpSpan[];
}

interface OtlpResourceSpans {
  resource?: { attributes?: OtlpKeyValue[] };
  scopeSpans?: OtlpScopeSpans[];
}

interface TempoTraceByIdV2Response {
  trace?: { resourceSpans?: OtlpResourceSpans[] };
}

// OTLP/JSON span kind enum names -> the short label toOtelSpan expects on a "span.kind" tag.
const SPAN_KIND_LABEL: Record<string, string> = {
  SPAN_KIND_INTERNAL: "INTERNAL",
  SPAN_KIND_SERVER: "SERVER",
  SPAN_KIND_CLIENT: "CLIENT",
  SPAN_KIND_PRODUCER: "PRODUCER",
  SPAN_KIND_CONSUMER: "CONSUMER",
};

function base64ToHex(b64: string): string {
  if (!b64) return "";
  const binary = atob(b64);
  let hex = "";
  for (let i = 0; i < binary.length; i++) {
    hex += binary.charCodeAt(i).toString(16).padStart(2, "0");
  }
  return hex;
}

function anyValueToString(v?: OtlpAnyValue): string {
  if (!v) return "";
  if (v.stringValue !== undefined) return v.stringValue;
  if (v.boolValue !== undefined) return String(v.boolValue);
  if (v.intValue !== undefined) return v.intValue;
  if (v.doubleValue !== undefined) return String(v.doubleValue);
  if (v.bytesValue !== undefined) return v.bytesValue;
  if (v.arrayValue) return JSON.stringify(v.arrayValue.values.map(anyValueToString));
  if (v.kvlistValue) return JSON.stringify(
    Object.fromEntries(v.kvlistValue.values.map(kv => [kv.key, anyValueToString(kv.value)]))
  );
  return "";
}

function attributesToTags(attributes: OtlpKeyValue[] | undefined, excludeKey?: string): KeyValuePair[] {
  return (attributes || [])
    .filter(kv => kv.key !== excludeKey)
    .map(kv => ({ key: kv.key, value: anyValueToString(kv.value) }));
}

function resourceServiceName(resource: OtlpResourceSpans["resource"]): string {
  const kv = resource?.attributes?.find(a => a.key === "service.name");
  return (kv?.value && anyValueToString(kv.value)) || "unknown-service";
}

// Rebuilds the exact shape transformTraceData()/toOtelTrace() already know how to consume
// (previously fed by the Jaeger REST API) directly from the Tempo API's OTLP/JSON response,
// so the rest of the TraceView pipeline needs no changes at all.
function resourceSpansToTraceData(traceId: string, resourceSpansList: OtlpResourceSpans[]): TraceData & { spans: SpanData[] } {
  const processes: Record<string, Process> = {};
  const knownProcesses: { serviceName: string; tags: KeyValuePair[]; processID: string }[] = [];
  const spanData: SpanData[] = [];

  for (const rs of resourceSpansList) {
    const serviceName = resourceServiceName(rs.resource);
    const tags = attributesToTags(rs.resource?.attributes, "service.name");
    const existingProcess = knownProcesses.find(p => p.serviceName === serviceName && _isEqual(p.tags, tags));
    const processID = existingProcess?.processID || `p${knownProcesses.length + 1}`;
    if (!existingProcess) {
      knownProcesses.push({ serviceName, tags, processID });
      processes[processID] = { serviceName, tags };
    }

    for (const ss of rs.scopeSpans || []) {
      for (const span of ss.spans || []) {
        const tags: KeyValuePair[] = attributesToTags(span.attributes);
        const kindLabel = span.kind ? SPAN_KIND_LABEL[span.kind] : undefined;
        if (kindLabel) tags.push({ key: "span.kind", value: kindLabel });
        if (span.status?.code === "STATUS_CODE_ERROR") tags.push({ key: "error", value: "true" });

        const parentSpanId = span.parentSpanId ? base64ToHex(span.parentSpanId) : "";
        const references: SpanReference[] = parentSpanId
          ? [{ refType: "CHILD_OF", spanID: parentSpanId, traceID: traceId }]
          : [];

        const links: SpanLink[] = (span.links || []).map(l => ({
          traceID: base64ToHex(l.traceId),
          spanID: base64ToHex(l.spanId),
          attributes: attributesToTags(l.attributes),
        }));

        const logs = (span.events || []).map(e => ({
          timestamp: Math.round((Number(e.timeUnixNano) || 0) / 1000),
          fields: [{ key: "event", value: e.name || "" }, ...attributesToTags(e.attributes)],
        }));

        const startTimeUnixNano = Number(span.startTimeUnixNano) || 0;
        const endTimeUnixNano = Number(span.endTimeUnixNano) || 0;

        spanData.push({
          spanID: base64ToHex(span.spanId),
          traceID: traceId,
          processID,
          operationName: span.name || "",
          // SpanData timestamps are microseconds; OTLP unix-nano fields are nanoseconds.
          startTime: Math.round(startTimeUnixNano / 1000),
          duration: Math.round((endTimeUnixNano - startTimeUnixNano) / 1000),
          tags,
          logs,
          references,
          links,
        });
      }
    }
  }

  return { traceID: traceId, processes, spans: spanData };
}

function deduplicateTags(spanTags: ReadonlyArray<KeyValuePair>) {
  const warningsHash: Map<string, string> = new Map<string, string>();
  const tags: KeyValuePair[] = spanTags.reduce<KeyValuePair[]>((uniqueTags, tag) => {
    if (!uniqueTags.some(t => t.key === tag.key && t.value === tag.value)) {
      uniqueTags.push(tag);
    } else {
      warningsHash.set(`${tag.key}:${tag.value}`, `Duplicate tag "${tag.key}:${tag.value}"`);
    }
    return uniqueTags;
  }, []);
  const warnings = Array.from(warningsHash.values());
  return { tags, warnings };
}

function orderTags(spanTags: KeyValuePair[]) {
  const orderedTags: KeyValuePair[] = spanTags.slice();

  orderedTags.sort((a, b) => {
    const aKey = a.key.toLowerCase();
    const bKey = b.key.toLowerCase();

    if (aKey > bKey) {
      return 1;
    }
    if (aKey < bKey) {
      return -1;
    }
    return 0;
  });

  return orderedTags;
}

// Mutates `data` - transforms the HTTP response data into the form the app generally requires.
function transformTraceData(data: TraceData & { spans: SpanData[] }): Trace | null {
  let { traceID } = data;
  if (!traceID) {
    return null;
  }
  traceID = traceID.toLowerCase();

  let traceEndTime = 0;
  let traceStartTime = Number.MAX_SAFE_INTEGER;
  const spanIdCounts = new Map<string, number>();
  const spanMap = new Map<string, Span>();

  // Filter out spans with empty start times
  data.spans = data.spans.filter(span => Boolean(span.startTime));

  const numSpans = data.spans.length;
  for (let i = 0; i < numSpans; i++) {
    // Unsafe cast to avoid memory allocations.
    // We populate/fix all properties below.
    const span: Span = data.spans[i] as Span;
    const { startTime, duration, processID } = span;
    let spanID = span.spanID;
    // make sure span IDs are unique
    const idCount = spanIdCounts.get(spanID);
    if (idCount != null) {
      console.warn(`Dupe spanID, ${idCount + 1} x ${spanID}`, span, spanMap.get(spanID));
      if (_isEqual(span, spanMap.get(spanID))) {
        console.warn("\t two spans with same ID have `isEqual(...) === true`");
      }
      spanIdCounts.set(spanID, idCount + 1);
      spanID = `${spanID}_${idCount}`;
      span.spanID = spanID;
    } else {
      spanIdCounts.set(spanID, 1);
    }
    span.process = data.processes[processID] || { serviceName: "unknown-service" };
    span.process.tags = span.process.tags || [];
    span.tags = span.tags || [];
    span.logs = span.logs || [];
    span.logs.forEach(log => {
      log.fields = log.fields || [];
    });
    span.references = span.references || [];
    span.links = span.links || [];
    span.childSpans = [];

    const tagsInfo = deduplicateTags(span.tags);
    span.tags = orderTags(tagsInfo.tags);

    spanMap.set(spanID, span);

    // update trace's start / end time
    if (startTime < traceStartTime) {
      traceStartTime = startTime;
    }
    if (startTime + duration > traceEndTime) {
      traceEndTime = startTime + duration;
    }
  }

  const rootSpans: Span[] = [];

  // Second pass: link parents/children and identify roots
  for (const span of spanMap.values()) {
    let parent: Span | undefined;
    if (Array.isArray(span.references) && span.references.length > 0) {
      // Find the first CHILD_OF or FOLLOWS_FROM reference that exists in the spanMap
      for (const ref of span.references) {
        if (ref.refType === "CHILD_OF" || ref.refType === "FOLLOWS_FROM") {
          parent = spanMap.get(ref.spanID);
          if (parent) {
            break;
          }
        }
      }
    }

    if (parent) {
      // It's a child
      (parent.childSpans as Span[]).push(span);
    } else {
      // It's a root
      rootSpans.push(span);
    }
  }

  const spans: Span[] = [];

  // Depth-first traversal to order spans and populate flat array
  const processSpan = (span: Span, depth: number) => {
    span.depth = depth;
    span.hasChildren = span.childSpans.length > 0;
    span.relativeStartTime = span.startTime - traceStartTime;

    spans.push(span);

    // Sort children by startTime before processing them
    (span.childSpans as Span[]).sort((a, b) => a.startTime - b.startTime);
    span.childSpans.forEach(child => processSpan(child, depth + 1));
  };

  rootSpans.sort((a, b) => a.startTime - b.startTime);
  rootSpans.forEach(root => processSpan(root, 0));

  return {
    spans,
    traceID,
    spanMap,
    rootSpans,
    processes: data.processes,
    duration: (traceEndTime - traceStartTime) as OtelTrace["duration"],
    startTime: traceStartTime as OtelTrace["startTime"],
    endTime: traceEndTime as OtelTrace["endTime"],
  };
}

export function useFetchTraceById() {
  const { serverUrl } = useAppState();
  const tenant = useTenant();
  const [trace, setTrace] = useState<OtelTrace>();
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string>();
  const abortControllerRef = useRef(new AbortController());

  const fetchTrace = useCallback(async (id: string) => {
    abortControllerRef.current.abort();
    const controller = new AbortController();
    abortControllerRef.current = controller;
    const { signal } = controller;

    setIsLoading(true);
    setError(undefined);
    setTrace(undefined);

    try {
      // The Tempo API looks the trace up via VictoriaTraces' own trace_id index (narrowing
      // to the trace's actual time window server-side) instead of scanning the whole
      // retention window like a plain `trace_id:"..."` LogsQL query would.
      const url = getTempoTraceByIdUrl(serverUrl, id);

      const response = await fetch(url, {
        signal: AbortSignal.any([signal, AbortSignal.timeout(REQUEST_TIMEOUT_MS)]),
        headers: { ...tenant },
      });

      const text = await response.text();
      if (!response.ok) {
        setError(text || "Trace not found");
        return;
      }

      const json = text ? JSON.parse(text) as TempoTraceByIdV2Response : undefined;
      const resourceSpansList = json?.trace?.resourceSpans || [];
      if (!resourceSpansList.length) {
        setError("Trace not found");
        return;
      }

      const legacyTrace = transformTraceData(resourceSpansToTraceData(id, resourceSpansList));
      const otelTrace = legacyTrace ? toOtelTrace(legacyTrace) : undefined;
      if (!otelTrace) {
        setError("Trace not found");
        return;
      }
      setTrace(otelTrace);
    } catch (e) {
      if (e instanceof Error && e.name !== "AbortError") {
        setError(e.name === "TimeoutError" ? "Request timed out. The server may be unreachable." : String(e));
        console.error(e);
      }
    } finally {
      if (abortControllerRef.current === controller) {
        setIsLoading(false);
      }
    }
  }, [serverUrl, tenant]);

  return { trace, isLoading, error, fetchTrace };
}
