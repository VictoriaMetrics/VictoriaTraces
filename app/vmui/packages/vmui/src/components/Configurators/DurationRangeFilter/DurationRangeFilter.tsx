import { FC, useMemo, useRef, useState } from "react";
import { Slider } from "antd";
import { FilterIcon, ArrowDownIcon } from "../../Main/Icons";
import Button from "../../Main/Button/Button";
import Popper from "../../Main/Popper/Popper";
import useDeviceDetect from "../../../hooks/useDeviceDetect";
import useBoolean from "../../../hooks/useBoolean";
import { getAppModeEnable } from "../../../utils/app-mode";
import { useTracesSearchSettings } from "../../../layouts/TracesLayout/TracesSearchSettingsContext";
import classNames from "classnames";
import "./style.scss";

// A handful of marks on a log-ish scale rather than a continuous range — trace
// durations span many orders of magnitude, so a linear slider would be unusable.
const MARKS_MS: (number | null)[] = [0, 1, 10, 100, 1000, 10_000, 60_000, null];
const MAX_POSITION = MARKS_MS.length - 1;

function formatDurationMs(ms: number): string {
  if (ms === 0) return "0ms";
  return ms < 1000 ? `${ms}ms` : `${ms / 1000}s`;
}

const marks = MARKS_MS.reduce((acc, ms, position) => {
  acc[position] = ms === null ? `${formatDurationMs(MARKS_MS[position - 1] as number)}+` : formatDurationMs(ms);
  return acc;
}, {} as Record<number, string>);

function minDurationFromPosition(position: number): string | null {
  if (position === 0) return null;
  return formatDurationMs(MARKS_MS[position] as number);
}

function maxDurationFromPosition(position: number): string | null {
  const ms = MARKS_MS[position];
  return ms === null ? null : formatDurationMs(ms);
}

function positionFromMinDuration(value: string | null): number {
  if (!value) return 0;
  const index = MARKS_MS.findIndex(ms => ms !== null && formatDurationMs(ms) === value);
  return index === -1 ? 0 : index;
}

function positionFromMaxDuration(value: string | null): number {
  if (!value) return MAX_POSITION;
  const index = MARKS_MS.findIndex(ms => ms !== null && formatDurationMs(ms) === value);
  return index === -1 ? MAX_POSITION : index;
}

const DurationRangeFilter: FC = () => {
  const appModeEnable = getAppModeEnable();
  const { isMobile } = useDeviceDetect();
  const { durationRange, setDurationRange } = useTracesSearchSettings();

  const {
    value: openOptions,
    toggle: toggleOpenOptions,
    setFalse: setCloseOptions,
  } = useBoolean(false);

  const buttonRef = useRef<HTMLDivElement>(null);

  const [range, setRange] = useState<[number, number]>(() => [
    positionFromMinDuration(durationRange.minDuration),
    positionFromMaxDuration(durationRange.maxDuration),
  ]);

  const isUnlimited = range[0] === 0 && range[1] === MAX_POSITION;

  const textValue = useMemo(() => {
    if (isUnlimited) return "any";
    return `${marks[range[0]]} – ${marks[range[1]]}`;
  }, [isUnlimited, range]);

  const commitRange = (next: [number, number]) => {
    setDurationRange({
      minDuration: minDurationFromPosition(next[0]),
      maxDuration: maxDurationFromPosition(next[1]),
    });
  };

  const handleClose = () => {
    commitRange(range);
    setCloseOptions();
  };

  return (
    <div className="vm-duration-range-filter" ref={buttonRef}>
      {isMobile ? (
        <div className="vm-mobile-option" onClick={toggleOpenOptions}>
          <span className="vm-mobile-option__icon"><FilterIcon/></span>
          <div className="vm-mobile-option-text">
            <span className="vm-mobile-option-text__label">Duration</span>
            <span className="vm-mobile-option-text__value">{textValue}</span>
          </div>
          <span className="vm-mobile-option__arrow"><ArrowDownIcon/></span>
        </div>
      ) : (
        <Button
          className={appModeEnable ? "" : "vm-header-button"}
          variant="contained"
          color="primary"
          startIcon={<FilterIcon/>}
          onClick={toggleOpenOptions}
        >
          Duration: {textValue}
        </Button>
      )}
      <Popper
        open={openOptions}
        placement="bottom-right"
        onClose={handleClose}
        buttonRef={buttonRef}
        title={isMobile ? "Filter traces by duration" : undefined}
      >
        <div
          className={classNames({
            "vm-duration-range-filter-popper": true,
            "vm-duration-range-filter-popper_mobile": isMobile,
          })}
        >
          <div className="vm-duration-range-filter-popper__label">Duration range: {textValue}</div>
          <Slider
            range
            min={0}
            max={MAX_POSITION}
            step={1}
            marks={marks}
            value={range}
            tooltip={{ formatter: position => marks[position ?? 0] }}
            onChange={(value: number[]) => setRange([value[0], value[1]])}
          />
          <div className="vm-duration-range-filter-popper-info">
            <p>
              Filters search results to only traces whose total duration falls within this range.
              Left at <code>0</code> and <code>{marks[MAX_POSITION]}</code> the filter is unlimited.
            </p>
          </div>
        </div>
      </Popper>
    </div>
  );
};

export default DurationRangeFilter;
