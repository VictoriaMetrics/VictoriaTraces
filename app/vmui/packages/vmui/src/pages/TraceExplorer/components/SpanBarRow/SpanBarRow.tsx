import React, { useCallback } from "react";
import cx from "classnames";
import { ArrowBackIcon, ErrorIcon, FocusIcon, UnfocusIcon } from "../../../../components/Main/Icons";
import Button from "../../../../components/Main/Button";
import Tooltip from "../../../../components/Main/Tooltip";
import TimelineRow from "../TimelineRow";
import { formatDuration } from "../../../../utils/date";
import SpanTreeOffset from "../SpanTreeOffset";
import TimeScale from "../TimeScale";

import { CriticalPathSection, OtelSpan, TNil } from "../../types";
import { ViewedBoundsFunctionType } from "../../utils";

import "./style.scss";

type SpanBarRowProps = {
  className?: string;
  color: string;
  criticalPath: CriticalPathSection[];
  nameColumnWidth: number;
  isSelected: boolean;
  isIsolatedRoot: boolean;
  isChildrenExpanded: boolean;
  isMatchingFilter: boolean;
  onSelectSpan: (spanID: string) => void;
  onIsolateSpan: (spanID: string) => void;
  onChildrenToggled: (spanID: string) => void;
  numTicks: number;
  rpc?:
    | {
      viewStart: number;
      viewEnd: number;
      color: string;
      operationName: string;
      serviceName: string;
    }
    | TNil;
  noInstrumentedServer?:
    | {
      color: string;
      serviceName: string;
    }
    | TNil;
  hasOwnError: boolean;
  getViewedBounds: ViewedBoundsFunctionType;
  span: OtelSpan;
};

/**
 * Wrapped in `React.memo` below since a trace can render thousands of these
 * rows, and most re-renders only affect a small subset of spans.
 */
const SpanBarRow: React.FC<SpanBarRowProps> = ({
  className,
  color,
  criticalPath,
  nameColumnWidth,
  isSelected,
  isIsolatedRoot,
  isChildrenExpanded,
  isMatchingFilter,
  numTicks,
  rpc = null,
  noInstrumentedServer,
  hasOwnError,
  getViewedBounds,
  span,
  onSelectSpan,
  onIsolateSpan,
  onChildrenToggled,
}) => {
  const _selectSpan = useCallback(() => {
    onSelectSpan(span.spanID);
  }, [onSelectSpan, span.spanID]);

  const _isolateSpan = useCallback((e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    onIsolateSpan(span.spanID);
  }, [onIsolateSpan, span.spanID]);

  const _childrenToggle = useCallback((e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    onChildrenToggled(span.spanID);
  }, [onChildrenToggled, span.spanID]);

  const {
    duration,
    name: operationName,
    resource: { serviceName },
  } = span;
  const label = formatDuration(duration);
  const viewBounds = getViewedBounds(span.startTime, span.endTime);
  const viewStart = viewBounds.start;
  const viewEnd = viewBounds.end;

  return (
    <TimelineRow
      className={cx("vm-span-bar-row", className, {
        "is-selected": isSelected,
        "is-matching-filter": isMatchingFilter,
      })}
    >
      <TimelineRow.Cell
        className="vm-span-bar-row__name-column"
        width={nameColumnWidth}
      >
        <SpanTreeOffset
          span={span}
          color={color}
          isChildrenExpanded={isChildrenExpanded}
          onChildrenToggled={_childrenToggle}
        />
        <button
          type="button"
          className="vm-span-bar-row__name"
          onClick={_selectSpan}
          style={{ borderColor: color }}
        >
          <span className="vm-endpoint-name">{rpc ? rpc.operationName : operationName}</span>
          <span className="vm-span-svc-name">
            {hasOwnError && <span className="vm-span-bar-row__error-icon"><ErrorIcon /></span>}
            {serviceName}{" "}
            {rpc && (
              <span>
                <span className="vm-span-bar-row__arrow-forward-icon"><ArrowBackIcon /></span>{" "}
                <i
                  className="vm-span-bar-row__rpc-color-marker"
                  style={{ background: rpc.color }}
                />
                {rpc.serviceName}
              </span>
            )}
            {noInstrumentedServer && (
              <span>
                <span className="vm-span-bar-row__arrow-forward-icon"><ArrowBackIcon /></span>{" "}
                <i
                  className="vm-span-bar-row__rpc-color-marker"
                  style={{ background: noInstrumentedServer.color }}
                />
                {noInstrumentedServer.serviceName}
              </span>
            )}
          </span>
        </button>
        <Tooltip title={isIsolatedRoot ? "Show full trace" : "Isolate this span as root"}>
          <Button
            className="vm-span-bar-row__isolate-button"
            variant="text"
            size="small"
            color="gray"
            startIcon={isIsolatedRoot ? <UnfocusIcon /> : <FocusIcon />}
            aria-label={isIsolatedRoot ? "Show full trace" : "Isolate this span as root"}
            onClick={_isolateSpan}
          />
        </Tooltip>
      </TimelineRow.Cell>
      <TimelineRow.Cell
        className="vm-span-bar-row__view"
        style={{ cursor: "pointer" }}
        width="auto"
        onClick={_selectSpan}
      >
        <TimeScale
          mode="graph"
          numTicks={numTicks}
          criticalPath={criticalPath}
          rpc={rpc}
          viewStart={viewStart}
          viewEnd={viewEnd}
          getViewedBounds={getViewedBounds}
          color={color}
          shortLabel={label}
          span={span}
        />
      </TimelineRow.Cell>
    </TimelineRow>
  );
};

export default React.memo(SpanBarRow);
