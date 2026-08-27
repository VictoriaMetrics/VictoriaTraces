import { FC, createPortal, useCallback, useMemo, useRef, useState } from "preact/compat";
import { useLocation, useNavigate } from "react-router-dom";
import classNames from "classnames";

import Button from "../../../../components/Main/Button";
import { CloseIcon, ErrorIcon, SpinnerIcon, TraceIcon } from "../../../../components/Main/Icons";
import DragResizeHandle from "../../../../components/Main/DragResizeHandle";
import { Size, useResizeObserver } from "../../../../hooks/useResizeObserver";
import useEventListener from "../../../../hooks/useEventListener";
import useDeviceDetect from "../../../../hooks/useDeviceDetect";
import { useAppState } from "../../../../state/common/StateContext";
import { getFromStorage, saveToStorage } from "../../../../utils/storage";
import { formatDatetime, formatDuration } from "../../../../utils/date";
import colorGenerator from "../../../../utils/color-generator";
import AttributesTable from "../AttributesTable";
import SummaryTable from "../SummaryTable";
import DrawerSection from "../DrawerSection";
import DurationDistribution from "../DurationDistribution";
import { useDurationDistribution } from "../../hooks/useDurationDistribution";
import { Attribute, Microseconds } from "../../types";
import { ERROR_STATUS_CODE, TraceSummary } from "../../hooks/useLogsqlTracesSearch";
import { SpanLogLine } from "../../hooks/useLogsqlTracesSearch";
import "./style.scss";

export interface TraceInfoDrawerProps {
  trace: TraceSummary;
  spans: SpanLogLine[];
  periodStart: bigint;
  periodEnd: bigint;
  onClose: () => void;
  onViewFullTrace: () => void;
}

const WIDTH_STORAGE_KEY = "TRACE_INFO_DRAWER_WIDTH";
const ATTRIBUTE_KEY_PREFIXES = ["resource_attr:", "span_attr:"];
const SKIPPED_ATTRIBUTE_KEYS = new Set(["resource_attr:service.name"]);
const MAX_ATTRIBUTES = 10;

const getDefaultWidth = () => {
  const widthFromStorage = getFromStorage(WIDTH_STORAGE_KEY);
  return widthFromStorage ? Number(widthFromStorage) : 0;
};

const getRootSpan = (spans: SpanLogLine[]): SpanLogLine | undefined => {
  const withoutParent = spans.find(span => !span.parent_span_id);
  if (withoutParent) return withoutParent;

  return spans.reduce<SpanLogLine | undefined>((earliest, span) => {
    if (!earliest) return span;
    return Number(span.start_time_unix_nano) < Number(earliest.start_time_unix_nano) ? span : earliest;
  }, undefined);
};

const getAttributeItems = (rootSpan?: SpanLogLine): Attribute[] => {
  if (!rootSpan) return [];

  return Object.entries(rootSpan)
    .filter((entry): entry is [string, string] => {
      const [key, value] = entry;
      return typeof value === "string" &&
        !SKIPPED_ATTRIBUTE_KEYS.has(key) &&
        ATTRIBUTE_KEY_PREFIXES.some(prefix => key.startsWith(prefix));
    })
    .slice(0, MAX_ATTRIBUTES)
    .map(([key, value]) => ({
      key: key.replace(/^(resource_attr:|span_attr:)/, ""),
      value,
    }));
};

const getServiceName = (span: SpanLogLine) => span["resource_attr:service.name"] || "unknown";

interface ServiceBreakdownItem {
  service: string;
  spanCount: number;
  totalDurationUs: number;
}

const getServiceBreakdown = (spans: SpanLogLine[]): ServiceBreakdownItem[] => {
  const byService = new Map<string, { spanCount: number; totalDurationNs: number }>();

  spans.forEach(span => {
    const service = getServiceName(span);
    const entry = byService.get(service) || { spanCount: 0, totalDurationNs: 0 };
    entry.spanCount += 1;
    entry.totalDurationNs += Number(span.duration) || 0;
    byService.set(service, entry);
  });

  return Array.from(byService.entries())
    .map(([service, { spanCount, totalDurationNs }]) => ({
      service,
      spanCount,
      totalDurationUs: totalDurationNs / 1000,
    }))
    .sort((a, b) => b.totalDurationUs - a.totalDurationUs);
};

interface CriticalSpanResult {
  span: SpanLogLine;
  criticalDurationUs: number;
}

const getCriticalSpan = (spans: SpanLogLine[]): CriticalSpanResult | undefined => {
  if (!spans.length) return undefined;

  const spanIds = new Set(spans.map(s => s.span_id));
  const childrenBySpanId = new Map<string, SpanLogLine[]>();
  const roots: SpanLogLine[] = [];

  spans.forEach(span => {
    if (span.parent_span_id && spanIds.has(span.parent_span_id)) {
      const list = childrenBySpanId.get(span.parent_span_id) || [];
      list.push(span);
      childrenBySpanId.set(span.parent_span_id, list);
    } else {
      roots.push(span);
    }
  });

  const startNs = (s: SpanLogLine) => Number(s.start_time_unix_nano);
  const endNs = (s: SpanLogLine) => Number(s.end_time_unix_nano);
  const criticalNsBySpanId = new Map<string, number>();

  const addCritical = (spanId: string, ns: number) => {
    criticalNsBySpanId.set(spanId, (criticalNsBySpanId.get(spanId) || 0) + ns);
  };

  const walk = (span: SpanLogLine, sectionEnd: number) => {
    const children = (childrenBySpanId.get(span.span_id) || [])
      .slice()
      .sort((a, b) => endNs(b) - endNs(a));

    let cursor = sectionEnd;
    let i = 0;
    while (cursor > startNs(span)) {
      while (i < children.length && endNs(children[i]) > cursor) i++;
      const child = children[i];
      if (!child) {
        addCritical(span.span_id, cursor - startNs(span));
        break;
      }
      if (endNs(child) < cursor) {
        addCritical(span.span_id, cursor - endNs(child));
      }
      walk(child, endNs(child));
      cursor = startNs(child);
      i++;
    }
  };

  roots.forEach(root => walk(root, endNs(root)));

  let bestSpanId: string | undefined;
  let bestNs = -1;
  criticalNsBySpanId.forEach((ns, spanId) => {
    if (ns > bestNs) {
      bestNs = ns;
      bestSpanId = spanId;
    }
  });
  if (!bestSpanId) return undefined;

  const span = spans.find(s => s.span_id === bestSpanId);
  return span ? { span, criticalDurationUs: bestNs / 1000 } : undefined;
};

const TraceInfoDrawer: FC<TraceInfoDrawerProps> = ({
  trace,
  spans,
  periodStart,
  periodEnd,
  onClose,
  onViewFullTrace,
}) => {
  const navigate = useNavigate();
  const location = useLocation();
  const { isMobile } = useDeviceDetect();
  const { isDarkTheme: isDarkThemeState } = useAppState();
  const isDarkTheme = isDarkThemeState ?? true;
  const { distribution: durationDistribution, isLoading: isDurationLoading } =
    useDurationDistribution(trace.operation, periodStart, periodEnd);

  const [containerWidth, setContainerWidth] = useState(() => getDefaultWidth());

  const containerRef = useRef<HTMLDivElement>(null);
  const [size, setSize] = useState<Size>({
    width: containerRef?.current?.offsetWidth,
    height: containerRef?.current?.offsetHeight,
  });
  useResizeObserver({ ref: containerRef, onResize: setSize });

  const rootSpan = useMemo(() => getRootSpan(spans), [spans]);
  const attributeItems = useMemo(() => getAttributeItems(rootSpan), [rootSpan]);
  const errorSpans = useMemo(() => spans.filter(span => span.status_code === ERROR_STATUS_CODE), [spans]);
  const serviceBreakdown = useMemo(() => getServiceBreakdown(spans), [spans]);
  const criticalSpan = useMemo(() => getCriticalSpan(spans), [spans]);

  const summaryItems = useMemo(() => [
    { key: "service", label: "Service", value: trace.service || "—" },
    { key: "operation", label: "Operation", value: trace.operation || "—" },
    { key: "duration", label: "Duration", value: formatDuration(trace.duration as Microseconds) },
    { key: "spanCount", label: "Spans", value: trace.spanCount },
    { key: "startTime", label: "Start time", value: formatDatetime(trace.startTime as Microseconds) },
  ], [trace]);

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Escape") onClose();
  };

  const handleResizeWidth = (width: number) => {
    setContainerWidth(width);
    saveToStorage(WIDTH_STORAGE_KEY, String(width));
  };

  const handlePopstate = useCallback(() => {
    navigate(location, { replace: true });
    onClose();
  }, [navigate, location, onClose]);

  useEventListener("popstate", handlePopstate);
  useEventListener("keydown", handleKeyDown);

  return createPortal(
    <div
      className={classNames("vm-trace-info-drawer-wrapper", { "vm-trace-info-drawer-wrapper_mobile": isMobile })}
    >
      <div
        className={classNames("vm-trace-info-drawer", { "vm-trace-info-drawer_mobile": isMobile })}
        style={containerWidth && !isMobile ? { width: `${containerWidth}px` } : undefined}
        ref={containerRef}
      >
        <div className="vm-trace-info-drawer-header">
          <div className="vm-trace-info-drawer-header__title">Trace preview</div>

          <Button
            variant="text"
            onClick={onClose}
            startIcon={<CloseIcon/>}
            aria-label="Close trace details"
          />
        </div>

        <div className="vm-trace-info-drawer-content">
          <SummaryTable items={summaryItems} />

          <Button
            variant="outlined"
            fullWidth
            startIcon={<TraceIcon/>}
            onClick={onViewFullTrace}
          >
            View full trace
          </Button>

          {durationDistribution ? (
            <DrawerSection title="Trace Duration">
              <DurationDistribution
                distribution={durationDistribution}
                operation={trace.operation}
                currentDurationNs={Number(rootSpan?.duration) || 0}
              />
            </DrawerSection>
          ) : isDurationLoading && (
            <DrawerSection title="Trace Duration">
              <div className="vm-trace-info-drawer-duration__loading">
                <SpinnerIcon/>
              </div>
            </DrawerSection>
          )}

          <DrawerSection
            title="Services"
            count={serviceBreakdown.length}
          >
            <div className="vm-trace-info-drawer-services__list">
              {serviceBreakdown.map((item, i) => (
                <div
                  className="vm-trace-info-drawer-services__row"
                  key={item.service}
                >
                  <span className="vm-trace-info-drawer-services__name-cell">
                    <span
                      className="vm-trace-info-drawer-services__dot"
                      style={{ backgroundColor: colorGenerator.getColorByKey(item.service, isDarkTheme) }}
                    />
                    <span className="vm-trace-info-drawer-services__name">
                      {item.service}
                      {i === 0 && serviceBreakdown.length > 1 && (
                        <span className="vm-trace-info-drawer-services__longest">Longest</span>
                      )}
                    </span>
                  </span>
                  <span className="vm-trace-info-drawer-services__spans">
                    {item.spanCount} span{item.spanCount === 1 ? "" : "s"}
                  </span>
                  <span className="vm-trace-info-drawer-services__duration">
                    {formatDuration(item.totalDurationUs as Microseconds)}
                  </span>
                </div>
              ))}
            </div>
          </DrawerSection>

          {criticalSpan && (
            <DrawerSection title="Critical span">
              <div className="vm-trace-info-drawer-critical__row">
                <span
                  className="vm-trace-info-drawer-critical__dot"
                  style={{ backgroundColor: colorGenerator.getColorByKey(getServiceName(criticalSpan.span), isDarkTheme) }}
                />
                <div className="vm-trace-info-drawer-critical__content">
                  <div className="vm-trace-info-drawer-critical__name">{criticalSpan.span.name}</div>
                  <div className="vm-trace-info-drawer-critical__service">
                    {getServiceName(criticalSpan.span)}
                  </div>
                </div>
                <div className="vm-trace-info-drawer-critical__values">
                  <div className="vm-trace-info-drawer-critical__duration">
                    {formatDuration(criticalSpan.criticalDurationUs as Microseconds)}
                  </div>
                  <div className="vm-trace-info-drawer-critical__caption">
                    span duration {formatDuration((Number(criticalSpan.span.duration) / 1000) as Microseconds)}
                  </div>
                </div>
              </div>
            </DrawerSection>
          )}

          <DrawerSection
            title="Errors"
            count={errorSpans.length}
          >
            {errorSpans.length ? (
              <div className="vm-trace-info-drawer-errors__list">
                {errorSpans.map(span => (
                  <div
                    className="vm-trace-info-drawer-errors__row"
                    key={span.span_id}
                  >
                    <span className="vm-trace-info-drawer-errors__icon"><ErrorIcon/></span>
                    <div className="vm-trace-info-drawer-errors__content">
                      <div className="vm-trace-info-drawer-errors__name">{span.name}</div>
                      <div className="vm-trace-info-drawer-errors__service">
                        {span["resource_attr:service.name"]}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="vm-trace-info-drawer-errors__empty">No errors in this trace.</div>
            )}
          </DrawerSection>

          {!!attributeItems.length && (
            <DrawerSection
              title="Attributes"
              count={attributeItems.length}
            >
              <AttributesTable data={attributeItems} />
            </DrawerSection>
          )}
        </div>
      </div>

      <DragResizeHandle
        targetRef={containerRef}
        minSize={320}
        dir={-1}
        size={size}
        onResizeEnd={handleResizeWidth}
      />
    </div>,
    document.body
  );
};

export default TraceInfoDrawer;
