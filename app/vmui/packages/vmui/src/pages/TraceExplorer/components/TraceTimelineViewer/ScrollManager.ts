import { OtelTrace, TNil } from "../../types";

/**
 * `Accessors` lets `ScrollManager` reach into `TraceTimelineViewer`'s row/scroll
 * state on demand without that state living in redux or being passed down as props.
 */
export type Accessors = {
  getViewHeight: () => number;
  getRowPosition: (rowIndex: number) => { height: number; y: number };
  mapSpanIndexToRowIndex: (spanIndex: number) => number;
};

type Scroller = {
  scrollTo: (rowIndex: number) => void;
};

export default class ScrollManager {
  _trace: OtelTrace | TNil;
  _scroller: Scroller | TNil;
  _accessors: Accessors | TNil;

  constructor(trace: OtelTrace | TNil, scroller: Scroller) {
    this._trace = trace;
    this._scroller = scroller;
    this._accessors = undefined;
  }

  /**
   * Sometimes the ScrollManager is created before the trace is loaded. This
   * setter allows the trace to be set asynchronously.
   */
  setTrace(trace: OtelTrace | TNil) {
    this._trace = trace;
  }

  /**
   * `setAccessors` is bound in the ctor, so it can be passed as a prop to
   * children components.
   */
  setAccessors = (accessors: Accessors) => {
    this._accessors = accessors;
  };

  /**
   * Scrolls so `spanID`'s own row is centered in the viewport, e.g. for a deep link that
   * targets one specific span rather than "whatever's near the top".
   */
  scrollToSpan = (spanID: string) => {
    const xrs = this._accessors;
    if (!xrs || !this._trace) {
      return;
    }
    const spanIndex = this._trace.spans.findIndex(span => span.spanID === spanID);
    if (spanIndex === -1) {
      return;
    }
    let rowIndex: number;
    try {
      rowIndex = xrs.mapSpanIndexToRowIndex(spanIndex);
    } catch {
      // Span is hidden behind a collapsed ancestor - nothing to scroll to.
      return;
    }
    const { y, height } = xrs.getRowPosition(rowIndex);
    const vh = xrs.getViewHeight();
    this._scroller?.scrollTo(y - vh / 2 + height / 2);
  };

  destroy() {
    this._trace = undefined;
    this._scroller = undefined;
    this._accessors = undefined;
  }
}
