import { useEffect, useRef, useState } from "react";
import cx from "classnames";
import _groupBy from "lodash/groupBy";

import Tooltip from "../../../../components/Main/Tooltip";
import SpanEventsList from "../SpanEventsList";
import { formatClockTime, formatDuration } from "../../../../utils/date";
import { ViewedBoundsFunctionType } from "../../utils";
import { CriticalPathSection, SpanEvent, OtelSpan, TNil } from "../../types";

import "./style.scss";

type TBaseProps = {
  numTicks: number;
};

type TScaleModeProps = TBaseProps & {
  mode: "scale";
  startTime: number;
  endTime: number;
};

type TGraphModeProps = TBaseProps & {
  mode: "graph";
  color: string;
  criticalPath: CriticalPathSection[];
  viewEnd: number;
  viewStart: number;
  getViewedBounds: ViewedBoundsFunctionType;
  rpc:
    | {
      viewStart: number;
      viewEnd: number;
      color: string;
      operationName: string;
      serviceName: string;
    }
    | TNil;
  span: OtelSpan;
  shortLabel: string;
};

// A standalone axis for a single span's own duration, with clickable/selectable event
// markers - no rpc/critical-path, unlike graph mode's per-row rendering.
type TEventsModeProps = TBaseProps & {
  mode: "events";
  color: string;
  startTime: number;
  endTime: number;
  events: ReadonlyArray<SpanEvent>;
  selectedEvent: SpanEvent | null;
  onSelectEvent: (event: SpanEvent) => void;
  // Tick labels: elapsed time since the span's own start (default) vs. absolute clock time.
  absolute?: boolean;
};

type TimeScaleProps = TScaleModeProps | TGraphModeProps | TEventsModeProps;

function toPercent(value: number) {
  return `${(value * 100).toFixed(1)}%`;
}

function toPercentInDecimal(value: number) {
  return `${value * 100}%`;
}

function TimeScaleCriticalPath(props: { criticalPathViewStart: number; criticalPathViewEnd: number; color: string }) {
  const [isHovering, setIsHovering] = useState(false);

  const criticalPath = (
    <div
      data-testid="TimeScale--criticalPath"
      className="vm-time-scale__critical-path"
      onMouseEnter={() => setIsHovering(true)}
      onMouseLeave={() => setIsHovering(false)}
      style={{
        background: `color-mix(in oklch, ${props.color} 70%, black)`,
        left: toPercentInDecimal(props.criticalPathViewStart),
        width: toPercentInDecimal(props.criticalPathViewEnd - props.criticalPathViewStart),
      }}
    />
  );

  // Only mount the tooltip once hovering starts, to avoid the up-front cost of rendering one
  // per critical-path segment on page load. `open` tracks our own hover state directly (rather
  // than Tooltip's own mouseenter listener, which wouldn't be attached yet for this first hover).
  if (!isHovering) {
    return criticalPath;
  }

  return (
    <Tooltip
      placement="top-center"
      open={isHovering}
      title={
        <div>
          A segment on the <em>critical path</em> of the overall trace/request/workflow.
        </div>
      }
    >
      {criticalPath}
    </Tooltip>
  );
}

function TimeScaleRpc(props: {
  viewStart: number;
  viewEnd: number;
  color: string;
  operationName: string;
  serviceName: string;
}) {
  const [isHovering, setIsHovering] = useState(false);

  const rpc = (
    <div
      className="vm-time-scale__rpc"
      onMouseEnter={() => setIsHovering(true)}
      onMouseLeave={() => setIsHovering(false)}
      style={{
        background: props.color,
        left: toPercent(props.viewStart),
        width: toPercent(props.viewEnd - props.viewStart),
      }}
    />
  );

  if (!isHovering) {
    return rpc;
  }

  return (
    <Tooltip
      placement="top-center"
      open={isHovering}
      title={
        <div>
          RPC to <strong>{props.serviceName}</strong>: {props.operationName}
        </div>
      }
    >
      {rpc}
    </Tooltip>
  );
}

function TimeScaleSpan(props: TGraphModeProps) {
  const {
    criticalPath,
    viewEnd,
    viewStart,
    getViewedBounds,
    color,
    rpc,
    span,
    shortLabel,
  } = props;

  // group events based on timestamps
  const eventGroups = _groupBy(span.events, (event: SpanEvent) => {
    const posPercent = getViewedBounds(event.timestamp, event.timestamp).start;
    // round to the nearest 0.2%
    return toPercent(Math.round(posPercent * 500) / 500);
  });

  return (
    <div
      className="vm-time-scale__span-wrapper"
      aria-hidden
    >
      <div
        aria-label={shortLabel}
        className="vm-time-scale__bar"
        style={{
          background: color,
          left: toPercent(viewStart),
          width: toPercent(viewEnd - viewStart),
        }}
      >
        <div className="vm-time-scale__label">{shortLabel}</div>
      </div>
      {Object.keys(eventGroups).map(positionKey => (
        <Tooltip
          key={positionKey}
          placement="top-left"
          title={
            <SpanEventsList
              events={eventGroups[positionKey]}
              spanStartTime={span.startTime}
            />
          }
        >
          <div
            className="vm-time-scale__log-marker"
            style={{ left: positionKey, zIndex: 3 }}
          />
        </Tooltip>
      ))}
      {rpc && (
        <TimeScaleRpc
          viewStart={rpc.viewStart}
          viewEnd={rpc.viewEnd}
          color={rpc.color}
          operationName={rpc.operationName}
          serviceName={rpc.serviceName}
        />
      )}
      {criticalPath &&
        criticalPath.map((each, index) => {
          const critcalPathViewBounds = getViewedBounds(each.sectionStart, each.sectionEnd);
          const criticalPathViewStart = critcalPathViewBounds.start;
          const criticalPathViewEnd = critcalPathViewBounds.end;
          const key = `${each.spanID}-${index}`;

          return (
            <TimeScaleCriticalPath
              criticalPathViewStart={criticalPathViewStart}
              criticalPathViewEnd={criticalPathViewEnd}
              color={color}
              key={key}
            />
          );
        })}
    </div>
  );
}

function TimeScaleEventMarkers(props: TEventsModeProps) {
  const { color, startTime, endTime, events, selectedEvent, onSelectEvent } = props;
  const duration = endTime - startTime;

  return (
    <>
      <div
        aria-hidden
        className="vm-time-scale__bar"
        style={{
          background: color,
          left: 0,
          width: "100%",
        }}
      />
      {events.map((event, i) => {
        const offset = event.timestamp - startTime;
        const percent = duration > 0 ? Math.min(1, Math.max(0, offset / duration)) : 0;
        const isException = event.name === "exception";
        const isSelected = selectedEvent === event;

        return (
          <Tooltip
            // eslint-disable-next-line @eslint-react/no-array-index-key -- multiple events can share the same timestamp; index is a stable tie-breaker for the whole-list-replace render
            key={`${event.timestamp}-${i}`}
            placement="top-center"
            title={
              <SpanEventsList
                events={[event]}
                spanStartTime={startTime}
              />
            }
          >
            <button
              type="button"
              className={cx("vm-time-scale__log-marker", "vm-time-scale__log-marker_large", {
                "vm-time-scale__log-marker_error": isException,
                "vm-time-scale__log-marker_selected": isSelected,
              })}
              style={{ left: toPercent(percent), zIndex: 3 }}
              aria-label={event.name}
              onClick={() => onSelectEvent(event)}
            />
          </Tooltip>
        );
      })}
    </>
  );
}

const MIN_TICK_SPACING_PX = 60;

function computeTickCount(width: number, maxTicks: number): number {
  if (width <= 0) return maxTicks;
  const fit = Math.floor(width / MIN_TICK_SPACING_PX) + 1;
  return Math.max(2, Math.min(maxTicks, fit));
}

export default function TimeScale(props: TimeScaleProps) {
  const { numTicks: maxTicks, mode } = props;
  const rootRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);

  useEffect(() => {
    const el = rootRef.current;
    if (!el) return;
    const observer = new ResizeObserver(entries => {
      const entry = entries[0];
      if (entry) setWidth(entry.contentRect.width);
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const numTicks = computeTickCount(width, maxTicks);
  const ticks = Array.from({ length: numTicks }, (_, i) => i / (numTicks - 1));

  const labels: string[] | undefined = mode === "scale"
    ? ticks.map(portion => {
      const viewingDuration = props.endTime - props.startTime;
      return formatDuration(props.startTime + portion * viewingDuration);
    })
    // `startTime`/`endTime` here are absolute epoch timestamps (needed to position event
    // markers against `event.timestamp`), not a relative-to-trace-start offset like "scale"
    // mode gets - so relative labels show elapsed time since the span's own start instead of
    // adding it to `startTime` like "scale" does.
    : mode === "events"
      ? ticks.map(portion => props.absolute
        ? formatClockTime(props.startTime + portion * (props.endTime - props.startTime))
        : formatDuration(portion * (props.endTime - props.startTime)))
      : undefined;

  return (
    <div
      ref={rootRef}
      className="vm-time-scale"
    >
      <div className="vm-time-scale__axis" />
      {ticks.map((portion, i) => (
        <div
          key={portion}
          className="vm-time-scale__tick"
          style={{ left: `${portion * 100}%` }}
        >
          {labels && (
            <span className="vm-time-scale__tick-label">
              {labels[i]}
            </span>
          )}
        </div>
      ))}
      {mode === "graph" && <TimeScaleSpan {...props} />}
      {mode === "events" && <TimeScaleEventMarkers {...props} />}
    </div>
  );
}
