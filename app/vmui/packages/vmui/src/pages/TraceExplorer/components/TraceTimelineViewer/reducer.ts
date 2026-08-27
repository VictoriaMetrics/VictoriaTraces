import { OtelTrace, TNil, TTraceTimeline } from "../../types";

export type TraceTimelineAction =
  | { type: "CLEAR_SHOULD_SCROLL_TO_SELECTED_SPAN" }
  | { type: "SELECT_SPAN"; payload: { spanID: string | null } }
  | { type: "ISOLATE_SPAN"; payload: { spanID: string } }
  | { type: "TOGGLE_CHILDREN"; payload: { spanID: string } }
  | { type: "SET_SPAN_NAME_COLUMN_WIDTH"; payload: { width: number } }
  | { type: "SET_TRACE"; payload: { trace: OtelTrace; spanId?: string | TNil } };

export function newInitialState(): TTraceTimeline {
  return {
    selectedSpanID: null,
    isolatedSpanID: null,
    collapsedSpanIDs: new Set(),
    shouldScrollToSelectedSpan: false,
    spanNameColumnWidth: parseFloat(localStorage.getItem("spanNameColumnWidth") || "300"),
    traceID: null,
  };
}

export function reducer(state: TTraceTimeline, action: TraceTimelineAction): TTraceTimeline {
  switch (action.type) {
    case "CLEAR_SHOULD_SCROLL_TO_SELECTED_SPAN": {
      if (state.shouldScrollToSelectedSpan) {
        return { ...state, shouldScrollToSelectedSpan: false };
      }
      return state;
    }
    case "SELECT_SPAN": {
      return { ...state, selectedSpanID: action.payload.spanID };
    }
    case "ISOLATE_SPAN": {
      const { spanID } = action.payload;
      return { ...state, isolatedSpanID: state.isolatedSpanID === spanID ? null : spanID };
    }
    case "TOGGLE_CHILDREN": {
      const collapsedSpanIDs = new Set(state.collapsedSpanIDs);
      const { spanID } = action.payload;
      if (collapsedSpanIDs.has(spanID)) {
        collapsedSpanIDs.delete(spanID);
      } else {
        collapsedSpanIDs.add(spanID);
      }
      return { ...state, collapsedSpanIDs };
    }
    case "SET_SPAN_NAME_COLUMN_WIDTH": {
      localStorage.setItem("spanNameColumnWidth", action.payload.width.toString());
      return { ...state, spanNameColumnWidth: action.payload.width };
    }
    case "SET_TRACE": {
      const { trace, spanId } = action.payload;
      const { traceID, spans } = trace;
      if (traceID === state.traceID) {
        return state;
      }
      const { spanNameColumnWidth } = state;
      const nextState = { ...newInitialState(), spanNameColumnWidth, traceID };

      // spanId names an exact span from a link/deep-link - jump straight to it if it's
      // actually in this trace, no fuzzy matching needed.
      const targetSpan = spanId ? spans.find(s => s.spanID === spanId) : undefined;
      if (targetSpan) {
        nextState.selectedSpanID = targetSpan.spanID;
        nextState.shouldScrollToSelectedSpan = true;
      } else {
        // Default to the root span selected so opening a trace shows something useful right
        // away, instead of an empty detail panel with nothing selected.
        const rootSpan = spans.find(s => !s.parentSpanID) || spans[0];
        if (rootSpan) {
          nextState.selectedSpanID = rootSpan.spanID;
        }
      }
      return nextState;
    }
    default:
      return state;
  }
}
