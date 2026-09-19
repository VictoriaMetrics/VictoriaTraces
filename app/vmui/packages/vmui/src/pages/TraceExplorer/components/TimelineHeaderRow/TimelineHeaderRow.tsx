import { useRef, useState } from "react";

import TimeScale from "../TimeScale";
import TimelineRow from "../TimelineRow";
import Button from "../../../../components/Main/Button";
import DragResizeHandle from "../../../../components/Main/DragResizeHandle";
import TextField from "../../../../components/Main/TextField";
import Tooltip from "../../../../components/Main/Tooltip";
import { CloseIcon, SearchIcon } from "../../../../components/Main/Icons";
import { formatDatetime } from "../../../../utils/date";
import { OtelSpan, ViewRangeTime } from "../../types";

import "./style.scss";

const MIN_NAME_COLUMN_WIDTH = 120;

type TimelineHeaderRowProps = {
  duration: number;
  startTime: number;
  nameColumnWidth: number;
  numTicks: number;
  onColummWidthChange: (width: number) => void;
  viewRangeTime: ViewRangeTime;
  selectedSpan?: OtelSpan;
  onCancelSelection: () => void;
  searchText: string;
  onSearchTextChange: (text: string) => void;
};

export default function TimelineHeaderRow(props: TimelineHeaderRowProps) {
  const {
    duration,
    startTime,
    nameColumnWidth,
    numTicks,
    onColummWidthChange,
    viewRangeTime,
    selectedSpan,
    onCancelSelection,
    searchText,
    onSearchTextChange,
  } = props;
  const [viewStart, viewEnd] = viewRangeTime.current;
  const tickStartTime = (viewStart * duration) as OtelSpan["startTime"];
  const tickEndTime = (viewEnd * duration) as OtelSpan["endTime"];
  const nameColumnRef = useRef<HTMLDivElement>(null);
  const [isSearchOpen, setIsSearchOpen] = useState(false);

  const closeSearch = () => {
    setIsSearchOpen(false);
    onSearchTextChange("");
  };

  return (
    <TimelineRow className="vm-timeline-header-row">
      <TimelineRow.Cell
        ref={nameColumnRef}
        className="vm-timeline-header-row__name-column"
        width={nameColumnWidth}
      >
        {isSearchOpen ? (
          <TextField
            autofocus
            placeholder="Search this trace"
            value={searchText}
            onChange={onSearchTextChange}
            onKeyDown={e => {
              if (e.key === "Escape") closeSearch();
            }}
            type="search"
          />
        ) : selectedSpan ? (
          <Button
            className="vm-timeline-header-row__selected-span"
            variant="text"
            size="small"
            color="gray"
            endIcon={<CloseIcon />}
            aria-label="Clear span selection"
            onClick={onCancelSelection}
          >
            <span className="vm-timeline-header-row__selected-span-label">
              {selectedSpan.resource.serviceName} / {selectedSpan.name}
            </span>
          </Button>
        ) : (
          <span className="vm-timeline-header-row__title">
            <span className="vm-timeline-header-row__title-label">Start time</span>
            <span className="vm-timeline-header-row__title-value">{formatDatetime(startTime)}</span>
          </span>
        )}
        <Tooltip title={isSearchOpen ? "Close search" : "Search this trace"}>
          <Button
            className="vm-timeline-header-row__search-toggle"
            variant="text"
            size="small"
            color="gray"
            startIcon={isSearchOpen ? <CloseIcon /> : <SearchIcon />}
            aria-label={isSearchOpen ? "Close search" : "Search this trace"}
            onClick={() => (isSearchOpen ? closeSearch() : setIsSearchOpen(true))}
          />
        </Tooltip>
      </TimelineRow.Cell>
      <div className="vm-timeline-header-row__resize-handle">
        <DragResizeHandle
          targetRef={nameColumnRef}
          minSize={MIN_NAME_COLUMN_WIDTH}
          dir={1}
          onResizeEnd={onColummWidthChange}
        />
      </div>
      <TimelineRow.Cell
        className="vm-timeline-header-row__ticks-column"
        width="auto"
      >
        <TimeScale
          mode="scale"
          numTicks={numTicks}
          startTime={tickStartTime}
          endTime={tickEndTime}
        />
      </TimelineRow.Cell>
    </TimelineRow>
  );
}
