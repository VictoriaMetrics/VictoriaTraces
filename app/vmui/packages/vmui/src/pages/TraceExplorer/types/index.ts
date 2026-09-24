/**
 * A branded type representing time values in microseconds.
 * This provides type safety to ensure time values are not confused with other numeric values.
 */
export type Microseconds = number & { readonly __brand: unique symbol };

export type TNil = null | undefined;

export type ApiError =
  | string
  | {
    message: string;
    httpStatus?: number;
    httpStatusText?: string;
    httpUrl?: string;
    httpQuery?: string;
    httpBody?: string;
  };

export enum SpanKind {
  INTERNAL = "INTERNAL",
  SERVER = "SERVER",
  CLIENT = "CLIENT",
  PRODUCER = "PRODUCER",
  CONSUMER = "CONSUMER",
}

export enum StatusCode {
  UNSET = "UNSET",
  OK = "OK",
  ERROR = "ERROR",
}

export type AttributeValue =
  | string
  | number
  | boolean
  | Array<AttributeValue>
  | { [key: string]: AttributeValue }
  | Uint8Array;

export type Attribute = {
  key: string;
  value: AttributeValue;
};

export type Resource = {
  attributes: Attribute[]; // includes service.name, etc.
  serviceName: string; // convenience: attributes['service.name']
};

export type SpanEvent = {
  timestamp: Microseconds;
  name: string;
  attributes: Attribute[];
};

export type Link = {
  traceID: string;
  spanID: string;
  attributes: Attribute[];
};

export type Status = {
  code: StatusCode;
  message?: string;
};

export type OtelSpan = {
  // Identity
  traceID: string;
  spanID: string;
  parentSpanID?: string;
  parentSpan?: OtelSpan;

  // Naming & Classification
  name: string;
  kind: SpanKind;

  // Timing
  startTime: Microseconds;
  endTime: Microseconds;
  duration: Microseconds;

  // Core Data
  attributes: Attribute[];
  events: SpanEvent[];
  links: Link[];
  status: Status;

  // Context
  resource: Resource;

  // Derived properties
  depth: number;
  hasChildren: boolean;
  childSpans: ReadonlyArray<OtelSpan>;
  relativeStartTime: Microseconds; // microseconds since trace start
};

export type OtelTrace = {
  traceID: string;
  spans: ReadonlyArray<OtelSpan>;

  // Some trace-level convenience properties
  duration: Microseconds;
  startTime: Microseconds;
  endTime: Microseconds;

  // Optimized data structures - created once during trace transformation
  spanMap: ReadonlyMap<string, OtelSpan>;
  rootSpans: ReadonlyArray<OtelSpan>;
};

// A section of a span that lies on the critical path
export type CriticalPathSection = {
  spanID: string;
  sectionStart: OtelSpan["startTime"];
  sectionEnd: OtelSpan["endTime"];
};

/**
 * All timestamps are in microseconds
 */
export type KeyValuePair<ValueType = string> = {
  key: string;
  value: ValueType;
};

export type Log = {
  timestamp: number;
  fields: ReadonlyArray<KeyValuePair>;
};

export type Process = {
  serviceName: string;
  tags: ReadonlyArray<KeyValuePair>;
};

export type SpanReference = {
  refType: "CHILD_OF" | "FOLLOWS_FROM";

  spanID: string;
  traceID: string;
};

export type SpanLink = {
  traceID: string;
  spanID: string;
  attributes: ReadonlyArray<KeyValuePair>;
};

export type SpanData = {
  spanID: string;
  traceID: string;
  processID: string;
  operationName: string;
  startTime: number;
  duration: number;
  tags?: ReadonlyArray<KeyValuePair>;
  logs?: ReadonlyArray<Log>;
  references?: ReadonlyArray<SpanReference>;
  links?: ReadonlyArray<SpanLink>;
  warnings?: ReadonlyArray<string> | null;
};

export type Span = SpanData & {
  tags: NonNullable<SpanData["tags"]>;
  logs: NonNullable<SpanData["logs"]>;
  references: NonNullable<SpanData["references"]>;
  links: NonNullable<SpanData["links"]>;
  warnings: NonNullable<SpanData["warnings"]>;

  depth: number;
  relativeStartTime: number;
  process: Process;

  hasChildren: boolean;
  childSpans: ReadonlyArray<Span>;
};

export type TraceData = {
  processes: Record<string, Process>;
  traceID: string;
};

export type Trace = TraceData & {
  duration: OtelTrace["duration"];
  endTime: OtelTrace["endTime"];
  spans: ReadonlyArray<Span>;
  startTime: OtelTrace["startTime"];

  // Optimized data structures - created once during trace transformation
  spanMap: ReadonlyMap<string, Span>;
  rootSpans: ReadonlyArray<Span>;
};

export type TTraceTimeline = {
  selectedSpanID: string | null;
  isolatedSpanID: string | null;
  collapsedSpanIDs: Set<string>;
  shouldScrollToSelectedSpan: boolean;
  spanNameColumnWidth: number;
  traceID: string | TNil;
};

export type ViewRangeTime = {
  current: [number, number];
};

export type ViewRange = {
  time: ViewRangeTime;
};

export const PEER_SERVICE = "peer.service";
