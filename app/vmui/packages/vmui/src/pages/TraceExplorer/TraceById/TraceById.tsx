import { FC, useCallback, useEffect, useMemo, useRef, useState } from "preact/compat";
import { useLocation, useNavigationType, useSearchParams } from "react-router-dom";
import classNames from "classnames";

import TraceExplorerHeader from "../components/TraceExplorerHeader";
import { useFetchTraceById } from "../hooks/useFetchTraceById";
import useDeviceDetect from "../../../hooks/useDeviceDetect";

import TraceTimelineViewer from "../components/TraceTimelineViewer";
import SpanDetailPanel from "../components/SpanDetailPanel";
import { useSpanDetailPanelWidth } from
  "../components/SpanDetailPanel/useSpanDetailPanelWidth";
import { TraceTimelineStateProvider, useTraceTimelineDispatch, useTraceTimelineState } from
  "../components/TraceTimelineViewer/TraceTimelineStateContext";
import DragResizeHandle from "../../../components/Main/DragResizeHandle";
import ApiErrorAlert from "../components/ApiErrorAlert";
import { SpinnerIcon } from "../../../components/Main/Icons";
import filterSpans, { getSpanIdFromSearchParams } from "../utils";
import { CriticalPathSection, OtelSpan, OtelTrace, ViewRange } from "../types";

import "../style.scss";
import "./style.scss";

const TRACE_ID_URL_PARAM = "trace_id";
const noop = () => {};

// Returns a view of `trace` scoped to `isolatedSpanID` and its descendants — the picked
// span becomes the new root (depth 0), and the trace-level time bounds become that span's
// own [startTime, endTime]. Always resolves `isolatedSpanID` against the original `trace`,
// never a previously-derived one, so re-isolating a different (or nested) span from an
// already-isolated view still works with nothing more than a single id.
//
// Every field not explicitly overridden below is delegated back to the original trace/span
// objects via the prototype chain (Object.create), rather than a plain `{...spread}`, so the
// rebased copies stay in sync with any fields added to OtelTrace/OtelSpan later.
function buildIsolatedTrace(trace: OtelTrace, isolatedSpanID: string | null): OtelTrace {
  if (!isolatedSpanID) return trace;

  const rootIndex = trace.spans.findIndex(span => span.spanID === isolatedSpanID);
  if (rootIndex === -1) return trace;

  const root = trace.spans[rootIndex];

  // trace.spans is flat and depth-first ordered (children immediately follow their parent),
  // so the contiguous run of spans deeper than root, starting right after it, is exactly its subtree.
  const subtree: OtelSpan[] = [root];
  for (let i = rootIndex + 1; i < trace.spans.length; i++) {
    if (trace.spans[i].depth <= root.depth) break;
    subtree.push(trace.spans[i]);
  }

  const rebasedSpans = subtree.map(span => Object.create(span, {
    depth: { value: span.depth - root.depth, enumerable: true },
  }) as OtelSpan);
  const rebasedRoot = rebasedSpans[0];

  return Object.create(trace, {
    spans: { value: rebasedSpans, enumerable: true },
    rootSpans: { value: [rebasedRoot], enumerable: true },
    spanMap: { value: new Map(rebasedSpans.map(span => [span.spanID, span])), enumerable: true },
    startTime: { value: root.startTime, enumerable: true },
    endTime: { value: root.endTime, enumerable: true },
    duration: { value: root.duration, enumerable: true },
  }) as OtelTrace;
}

// Walks `span`'s own [span.startTime, endTime] interval backward, picking at each step the child
// (by latest endTime <= the current cursor) that was still running right up to the cursor — that
// child is what's actually blocking `span` from finishing at that point, so recurse into it for its
// own interval. Any gap not covered by a child is `span` itself directly on the critical path.
function walkCriticalPath(span: OtelSpan, endTime: OtelSpan["endTime"], sections: CriticalPathSection[]): void {
  const children = [...span.childSpans].sort((a, b) => b.endTime - a.endTime);
  let cursor = endTime;
  let i = 0;
  while (cursor > span.startTime) {
    while (i < children.length && children[i].endTime > cursor) i++;
    const child = children[i];
    if (!child) {
      sections.push({ spanID: span.spanID, sectionStart: span.startTime, sectionEnd: cursor });
      break;
    }
    if (child.endTime < cursor) {
      sections.push({ spanID: span.spanID, sectionStart: child.endTime, sectionEnd: cursor });
    }
    walkCriticalPath(child, child.endTime, sections);
    cursor = child.startTime;
    i++;
  }
}

function computeCriticalPath(trace: OtelTrace): CriticalPathSection[] {
  const sections: CriticalPathSection[] = [];
  trace.rootSpans.forEach(root => walkCriticalPath(root, root.endTime, sections));
  return sections;
}

type TraceViewBodyProps = {
  trace: OtelTrace | undefined;
  isLoading: boolean;
  error?: string;
};

function TraceViewBody({ trace, isLoading, error }: TraceViewBodyProps) {
  const location = useLocation();
  const dispatch = useTraceTimelineDispatch();
  const { selectedSpanID, isolatedSpanID } = useTraceTimelineState();

  const [viewRange, setViewRange] = useState<ViewRange>({ time: { current: [0, 1] } });
  const [searchText, setSearchText] = useState("");

  const panelRef = useRef<HTMLDivElement>(null);
  const { width: panelWidth, setWidth: setPanelWidth } = useSpanDetailPanelWidth();

  const spanId = useMemo(() => {
    const params = new URLSearchParams(location.search);
    return getSpanIdFromSearchParams(params).spanId;
  }, [location.search]);

  // Scoped to isolatedSpanID's subtree when set; otherwise just the original trace.
  const displayTrace = useMemo(
    () => (trace ? buildIsolatedTrace(trace, isolatedSpanID) : undefined),
    [trace, isolatedSpanID]
  );

  // Looked up from the original trace, not displayTrace, so the panel still shows the
  // selected span even if it's currently scrolled out of an isolated view.
  const selectedSpan = selectedSpanID ? trace?.spanMap.get(selectedSpanID) : undefined;

  const findMatchesIDs = useMemo(() => {
    if (!searchText.trim() || !displayTrace) return undefined;
    return filterSpans(searchText, displayTrace.spans) ?? undefined;
  }, [searchText, displayTrace]);

  const closeSpanDetail = useCallback(() => {
    dispatch({ type: "SELECT_SPAN", payload: { spanID: null } });
  }, [dispatch]);

  const criticalPath = useMemo(
    () => (displayTrace ? computeCriticalPath(displayTrace) : []),
    [displayTrace]
  );

  useEffect(() => {
    if (trace) {
      dispatch({ type: "SET_TRACE", payload: { trace, spanId } });
    }
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- resets zoom/pan range whenever a genuinely new trace loads
    setViewRange({ time: { current: [0, 1] } });
    // Only reset when the trace identity changes, not on every spanId/URL change.
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- `trace`/`spanId` are intentionally excluded: only reset when the trace identity changes, not on every spanId/URL change (see comment above); `dispatch` is stable (useReducer)
  }, [trace?.traceID, dispatch]);

  if (isLoading) {
    return <span className="vm-trace-view-loading"><SpinnerIcon /></span>;
  }

  if (error) {
    return (
      <ApiErrorAlert
        error={error}
        className="vm-trace-view-error"
      />
    );
  }

  if (!trace) {
    return (
      <div className="vm-trace-view-placeholder">
        Enter a trace ID above and press Execute to view a trace.
      </div>
    );
  }

  // trace is defined past this point, so displayTrace (derived from it) always is too.
  const resolvedDisplayTrace = displayTrace as OtelTrace;

  return (
    <div className="vm-trace-explorer-body">
      <TraceTimelineViewer
        findMatchesIDs={findMatchesIDs}
        trace={resolvedDisplayTrace}
        trueRootSpanID={trace.rootSpans[0]?.spanID}
        criticalPath={criticalPath}
        viewRange={viewRange}
        searchText={searchText}
        onSearchTextChange={setSearchText}
      />
      {selectedSpan && (
        <div className="vm-trace-explorer-body__resize-handle">
          <DragResizeHandle
            targetRef={panelRef}
            minSize={320}
            dir={-1}
            onResizeEnd={setPanelWidth}
          />
        </div>
      )}
      {selectedSpan && (
        <SpanDetailPanel
          span={selectedSpan}
          panelRef={panelRef}
          width={panelWidth}
          onClose={closeSpanDetail}
        />
      )}
    </div>
  );
}

// Dedicated "look up a trace by ID" page (route: /trace) — the same content TraceExplorer's
// TraceID mode renders, minus the Search/TraceID switch (this page has no other mode to switch to).
const TraceById: FC = () => {
  const { isMobile } = useDeviceDetect();
  const location = useLocation();
  const navigationType = useNavigationType();
  const [searchParams, setSearchParams] = useSearchParams();

  const traceId = searchParams.get(TRACE_ID_URL_PARAM) || "";
  const { trace, isLoading, error, fetchTrace } = useFetchTraceById();

  const setTraceId = useCallback((value: string) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev);
      if (value) next.set(TRACE_ID_URL_PARAM, value);
      else next.delete(TRACE_ID_URL_PARAM);
      return next;
    });
  }, [setSearchParams]);

  const handleRun = useCallback(() => {
    if (!traceId.trim()) return;
    fetchTrace(traceId.trim());
  }, [traceId, fetchTrace]);

  // Restore results on a reload that already has a trace ID in the URL.
  useEffect(() => {
    if (traceId.trim()) handleRun();
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- intentionally mount-only; adding traceId/handleRun would re-run on every keystroke
  }, []);

  useEffect(() => {
    const isAutoRunLink = (location.state as { autoRun?: boolean } | null)?.autoRun;
    if (isAutoRunLink || navigationType === "POP") {
      handleRun();
    }
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- `handleRun`/`location.state`/`navigationType` intentionally excluded; see comment above on why this must not re-fire as traceId/handleRun change while typing
  }, [location.key]);

  return (
    <div className="vm-trace-explorer vm-trace-explorer_bounded">
      <div
        className={classNames({
          "vm-trace-explorer-header": true,
          "vm-block": true,
          "vm-block_mobile": isMobile,
        })}
      >
        <TraceExplorerHeader
          mode="traceId"
          traceId={traceId}
          onChangeTraceId={setTraceId}
          query=""
          onChangeQuery={noop}
          isLoading={isLoading}
          onRun={handleRun}
        />
      </div>

      <TraceTimelineStateProvider>
        <TraceViewBody
          trace={trace}
          isLoading={isLoading}
          error={error}
        />
      </TraceTimelineStateProvider>
    </div>
  );
};

export default TraceById;
