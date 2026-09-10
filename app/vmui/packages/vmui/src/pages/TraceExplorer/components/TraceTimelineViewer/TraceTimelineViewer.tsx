import { useCallback, useEffect, useMemo, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import cx from "classnames";
import _groupBy from "lodash/groupBy";

import { useTraceTimelineDispatch, useTraceTimelineState } from "./TraceTimelineStateContext";
import TimelineHeaderRow from "../TimelineHeaderRow";
import SpanBarRow from "../SpanBarRow";
import {
  createViewedBoundsFunc,
  isErrorSpan,
  isKindClient,
  isKindProducer,
} from "../../utils";
import ScrollManager, { Accessors } from "./ScrollManager";
import { createScroller } from "./scroll-page";
import colorGenerator from "../../../../utils/color-generator";
import { useAppState } from "../../../../state/common/StateContext";
import { computeRpc, getSpanIdFromSearchParams, SPAN_ID_URL_PARAM } from "../../utils";
import { ViewRange } from "../../types";
import { CriticalPathSection, OtelSpan, OtelTrace, PEER_SERVICE, TNil } from "../../types";

import "./style.scss";

type RowState = {
  span: OtelSpan;
  spanIndex: number;
};

type TProps = {
  findMatchesIDs: Set<string> | TNil;
  trace: OtelTrace;
  trueRootSpanID: string | undefined;
  criticalPath: CriticalPathSection[];
  viewRange: ViewRange;
  searchText: string;
  onSearchTextChange: (text: string) => void;
};

const NUM_TICKS = 5;
// Rows are auto-height now (content + padding), but the scroll accessors below still need a
// single known row height to do arithmetic without per-row measurement — every row has the same
// content shape (one endpoint-name line + one service-name line), so this stays a close estimate.
const ROW_HEIGHT = 44;

function generateRowStatesFromTrace(trace: OtelTrace | TNil, collapsedSpanIDs: Set<string>): RowState[] {
  if (!trace) {
    return [];
  }
  const hiddenSpanIDs = new Set<string>();
  const rowStates: RowState[] = [];
  trace.spans.forEach((span, spanIndex) => {
    if (span.parentSpanID && hiddenSpanIDs.has(span.parentSpanID)) {
      hiddenSpanIDs.add(span.spanID);
      return;
    }
    rowStates.push({ span, spanIndex });
    if (collapsedSpanIDs.has(span.spanID)) {
      hiddenSpanIDs.add(span.spanID);
    }
  });
  return rowStates;
}

const TraceTimelineViewer = (props: TProps) => {
  const {
    findMatchesIDs,
    trace,
    trueRootSpanID,
    criticalPath,
    viewRange,
    searchText,
    onSearchTextChange,
  } = props;
  const {
    spanNameColumnWidth,
    selectedSpanID,
    isolatedSpanID,
    collapsedSpanIDs,
    shouldScrollToSelectedSpan,
  } = useTraceTimelineState();
  const { isDarkTheme: isDarkThemeState } = useAppState();
  const isDarkTheme = isDarkThemeState ?? true;
  const dispatch = useTraceTimelineDispatch();
  const [searchParams, setSearchParams] = useSearchParams();
  const rowsContainerRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const scrollerRef = useRef(createScroller(() => scrollContainerRef.current));
  const scrollManagerRef = useRef<ScrollManager>();
  if (!scrollManagerRef.current) {
    scrollManagerRef.current = new ScrollManager(trace, scrollerRef.current);
  }

  const spanId = useMemo(() => getSpanIdFromSearchParams(searchParams).spanId, [searchParams]);

  const setSpanNameColumnWidth = useCallback(
    (width: number) => dispatch({ type: "SET_SPAN_NAME_COLUMN_WIDTH", payload: { width } }),
    [dispatch]
  );

  const headerSpan = isolatedSpanID ? trace.spanMap.get(isolatedSpanID) : undefined;

  const cancelSpanSelection = useCallback(() => {
    if (isolatedSpanID) dispatch({ type: "ISOLATE_SPAN", payload: { spanID: isolatedSpanID } });
    if (trueRootSpanID) dispatch({ type: "SELECT_SPAN", payload: { spanID: trueRootSpanID } });
  }, [dispatch, isolatedSpanID, trueRootSpanID]);

  useEffect(() => {
    dispatch({ type: "SET_TRACE", payload: { trace, spanId } });
    scrollManagerRef.current?.setTrace(trace);
  }, [dispatch, trace, spanId]);

  useEffect(() => {
    if (shouldScrollToSelectedSpan && selectedSpanID) {
      scrollManagerRef.current?.scrollToSpan(selectedSpanID);
      dispatch({ type: "CLEAR_SHOULD_SCROLL_TO_SELECTED_SPAN" });
    }
  }, [shouldScrollToSelectedSpan, selectedSpanID, dispatch]);

  useEffect(() => {
    const scroller = scrollerRef.current;
    return () => {
      scroller.cancel();
      scrollManagerRef.current?.destroy();
    };
  }, []);

  const rowStates = useMemo(
    () => generateRowStatesFromTrace(trace, collapsedSpanIDs),
    [trace, collapsedSpanIDs]
  );

  const [zoomStart, zoomEnd] = viewRange.time.current;

  const getViewedBounds = useMemo(
    () => createViewedBoundsFunc({
      min: trace.startTime,
      max: trace.endTime,
      viewStart: zoomStart,
      viewEnd: zoomEnd,
    }),
    [trace.startTime, trace.endTime, zoomStart, zoomEnd]
  );

  const clippingClassName = useMemo(() => cx({
    "vm-span-bar-row_clipping-left": zoomStart > 0,
    "vm-span-bar-row_clipping-right": zoomEnd < 1,
  }), [zoomStart, zoomEnd]);

  const criticalPathBySpanID = useMemo(() => _groupBy(criticalPath, x => x.spanID), [criticalPath]);

  const selectSpan = useCallback(
    (spanID: string) => {
      dispatch({ type: "SELECT_SPAN", payload: { spanID } });
      setSearchParams(prev => {
        const next = new URLSearchParams(prev);
        next.set(SPAN_ID_URL_PARAM, spanID);
        return next;
      }, { replace: true });
    },
    [dispatch, setSearchParams]
  );
  const isolateSpan = useCallback(
    (spanID: string) => dispatch({ type: "ISOLATE_SPAN", payload: { spanID } }),
    [dispatch]
  );
  const toggleChildren = useCallback(
    (spanID: string) => dispatch({ type: "TOGGLE_CHILDREN", payload: { spanID } }),
    [dispatch]
  );

  const mapSpanIndexToRowIndex = useCallback(
    (spanIndex: number) => {
      const row = rowStates.findIndex(rowState => rowState.spanIndex === spanIndex);
      if (row === -1) {
        throw new Error(`unable to find row for span index: ${spanIndex}`);
      }
      return row;
    },
    [rowStates]
  );

  useEffect(() => {
    // Position of the rows wrapper within the scroll container's own content, in the same
    // coordinate space as the container's scrollTop — independent of current scroll position.
    const containerTop = () => {
      const container = scrollContainerRef.current;
      const rows = rowsContainerRef.current;
      if (!container || !rows) return 0;
      const rowsRect = rows.getBoundingClientRect();
      const containerRect = container.getBoundingClientRect();
      return (rowsRect.top - containerRect.top) + container.scrollTop;
    };

    const accessors: Accessors = {
      getViewHeight: () => scrollContainerRef.current?.clientHeight ?? 0,
      getRowPosition: rowIndex => ({ height: ROW_HEIGHT, y: containerTop() + rowIndex * ROW_HEIGHT }),
      mapSpanIndexToRowIndex,
    };
    scrollManagerRef.current?.setAccessors(accessors);
  }, [mapSpanIndexToRowIndex]);

  return (
    <div
      ref={scrollContainerRef}
      className="vm-trace-timeline-viewer"
    >
      <TimelineHeaderRow
        duration={trace.duration}
        startTime={trace.startTime}
        nameColumnWidth={spanNameColumnWidth}
        numTicks={NUM_TICKS}
        onColummWidthChange={setSpanNameColumnWidth}
        viewRangeTime={viewRange.time}
        selectedSpan={headerSpan}
        onCancelSelection={cancelSpanSelection}
        searchText={searchText}
        onSearchTextChange={onSearchTextChange}
      />
      <div
        ref={rowsContainerRef}
        className="vm-trace-timeline-viewer__rows"
      >
        {rowStates.map(({ span }) => {
          const { spanID } = span;
          const { serviceName } = span.resource;
          const color = colorGenerator.getColorByKey(serviceName, isDarkTheme);
          const isSelected = selectedSpanID === spanID;
          const isIsolatedRoot = isolatedSpanID === spanID;
          const isMatchingFilter = findMatchesIDs ? findMatchesIDs.has(spanID) : false;
          const hasOwnError = isErrorSpan(span);
          const criticalPathSections = spanID in criticalPathBySpanID ? criticalPathBySpanID[spanID] : [];
          const peerServiceAttr = span.attributes.find(attr => attr.key === PEER_SERVICE);
          let noInstrumentedServer = null;
          if (!span.hasChildren && peerServiceAttr && (isKindClient(span) || isKindProducer(span))) {
            noInstrumentedServer = {
              serviceName: String(peerServiceAttr.value),
              color: colorGenerator.getColorByKey(String(peerServiceAttr.value), isDarkTheme),
            };
          }
          const rpc = computeRpc(span, getViewedBounds, isDarkTheme);

          return (
            <SpanBarRow
              key={spanID}
              className={clippingClassName}
              color={color}
              criticalPath={criticalPathSections}
              nameColumnWidth={spanNameColumnWidth}
              isSelected={isSelected}
              isIsolatedRoot={isIsolatedRoot}
              isChildrenExpanded={!collapsedSpanIDs.has(spanID)}
              isMatchingFilter={isMatchingFilter}
              numTicks={NUM_TICKS}
              onSelectSpan={selectSpan}
              onIsolateSpan={isolateSpan}
              onChildrenToggled={toggleChildren}
              noInstrumentedServer={noInstrumentedServer}
              rpc={rpc}
              hasOwnError={hasOwnError}
              getViewedBounds={getViewedBounds}
              span={span}
            />
          );
        })}
      </div>
    </div>
  );
};

export default TraceTimelineViewer;
