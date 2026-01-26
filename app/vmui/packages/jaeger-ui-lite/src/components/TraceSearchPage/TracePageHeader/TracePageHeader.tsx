import * as React from 'react';
import { Button, InputRef, Tooltip } from 'antd';
import _get from 'lodash/get';
import _maxBy from 'lodash/maxBy';
import { IoArrowBack, IoChevronForward, IoWarning } from 'react-icons/io5';
import { Link } from 'react-router';

import DocumentTitle from '../../../utils/documentTitle';
import SpanGraph from './SpanGraph';
import { TUpdateViewRangeTimeFunction, IViewRange, ViewRangeTimeUpdate, ETraceViewType } from '../types';
import LabeledList from '../../common/LabeledList';
import TraceName from '../../common/TraceName';
import { TNil } from '../../../types';
import { IOtelTrace } from '../../../types/otel';
import { formatDatetime, formatDuration } from '../../../utils/date';
import { getTraceLinks } from '../../../model/link-patterns';

import './TracePageHeader.css';
import ExternalLinks from '../../common/ExternalLinks';
import TraceId from '../../common/TraceId';

type TracePageHeaderEmbedProps = {
  canCollapse: boolean;
  clearSearch: () => void;
  focusUiFindMatches: () => void;
  hideMap: boolean;
  hideSummary: boolean;
  nextResult: () => void;
  onSlimViewClicked: () => void;
  onTraceViewChange: (viewType: ETraceViewType) => void;
  prevResult: () => void;
  resultCount: number;
  showArchiveButton: boolean;
  showShortcutsHelp: boolean;
  showStandaloneLink: boolean;
  disableJsonView: boolean;
  showViewOptions: boolean;
  slimView: boolean;
  textFilter: string | TNil;
  toSearch: string | null;
  trace: IOtelTrace;
  viewType: ETraceViewType;
  updateNextViewRangeTime: (update: ViewRangeTimeUpdate) => void;
  updateViewRangeTime: TUpdateViewRangeTimeFunction;
  viewRange: IViewRange;
  useOtelTerms: boolean;
};

export const HEADER_ITEMS = [
  {
    key: 'timestamp',
    label: 'Trace Start',
    renderer: (trace: IOtelTrace) => {
      const dateStr = formatDatetime(trace.startTime);
      const match = dateStr.match(/^(.+)(\.\d+)$/);
      return match ? (
        <span className="TracePageHeader--overviewItem--value">
          {match[1]}
          <span className="TracePageHeader--overviewItem--valueDetail">{match[2]}</span>
        </span>
      ) : (
        dateStr
      );
    },
  },
  {
    key: 'duration',
    label: 'Duration',
    renderer: (trace: IOtelTrace) => formatDuration(trace.duration),
  },
  {
    key: 'service-count',
    label: 'Services',
    renderer: (trace: IOtelTrace) => trace.services.length,
  },
  {
    key: 'depth',
    label: 'Depth',
    renderer: (trace: IOtelTrace) => _get(_maxBy(trace.spans as any[], 'depth'), 'depth', 0) + 1,
  },
  {
    key: 'span-count',
    label: 'Total Spans',
    renderer: (trace: IOtelTrace) => {
      const orphanCount = trace.orphanSpanCount ?? 0;
      const tooltipText = `This trace may be incomplete. ${orphanCount} span${orphanCount !== 1 ? 's' : ''} ${orphanCount !== 1 ? 'have' : 'has'} missing parent span(s).`;
      return (
        <span>
          {trace.spans.length}
          {orphanCount > 0 && (
            <>
              {' '}
              <Tooltip title={tooltipText}>
                <span className="TracePageHeader--incompleteTag">
                  <IoWarning className="TracePageHeader--incompleteIcon" />
                  Incomplete
                </span>
              </Tooltip>
            </>
          )}
        </span>
      );
    },
  },
];

export function TracePageHeaderFn(props: TracePageHeaderEmbedProps & { forwardedRef: React.Ref<InputRef> }) {
  const {
    canCollapse,
    hideMap,
    hideSummary,
    onSlimViewClicked,
    slimView,
    toSearch,
    trace,
    updateNextViewRangeTime,
    updateViewRangeTime,
    viewRange,
  } = props;

  if (!trace) {
    return null;
  }

  const links = getTraceLinks(trace);

  const summaryItems =
    !hideSummary &&
    !slimView &&
    HEADER_ITEMS.map(item => {
      const { renderer, ...rest } = item;
      return { ...rest, value: renderer(trace) };
    });

  const traceShortID = trace.traceID.slice(0, 7);

  const title = (
    <h1 className={`TracePageHeader--title ${canCollapse ? 'is-collapsible' : ''}`}>
      <TraceName traceName={trace.traceName} /> <TraceId traceId={trace.traceID} />
    </h1>
  );

  return (
    <header className="TracePageHeader">
      <DocumentTitle title={`${trace.traceEmoji} ${traceShortID}: ${trace.tracePageTitle} — Jaeger UI`} />
      <div className="TracePageHeader--titleRow">
        {toSearch && (
          <Link className="TracePageHeader--back" to={toSearch}>
            <IoArrowBack />
          </Link>
        )}
        {links && links.length > 0 && <ExternalLinks links={links} />}
        {canCollapse ? (
          <a
            className="TracePageHeader--titleLink"
            onClick={onSlimViewClicked}
            role="switch"
            aria-checked={!slimView}
          >
            <IoChevronForward className={`TracePageHeader--detailToggle ${!slimView ? 'is-expanded' : ''}`} />
            {title}
          </a>
        ) : (
          title
        )}
      </div>
      {summaryItems && <LabeledList className="TracePageHeader--overviewItems" items={summaryItems} />}
      {!hideMap && !slimView && (
        <SpanGraph
          trace={trace}
          viewRange={viewRange}
          updateNextViewRangeTime={updateNextViewRangeTime}
          updateViewRangeTime={updateViewRangeTime}
        />
      )}
    </header>
  );
}

export default React.forwardRef((props: TracePageHeaderEmbedProps, ref: React.Ref<InputRef>) => (
  <TracePageHeaderFn {...props} forwardedRef={ref} />
));
