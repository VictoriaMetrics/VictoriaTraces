import { FC, useEffect, useState } from "preact/compat";
import CalendarHeader from "./CalendarHeader";
import CalendarBody from "./CalendarBody";
import YearsList from "./YearsList";
import { DATE_FORMAT, DATE_TIME_FORMAT } from "../../../../constants/date";
import "./style.scss";
import useDeviceDetect from "../../../../hooks/useDeviceDetect";
import classNames from "classnames";
import MonthsList from "./MonthsList";
import Button from "../../Button";
import { getNowInTimezone, type VmDate } from "../../../../utils/time";

interface DatePickerProps {
  date: VmDate
  format?: string
  onChange: (date: string) => void
}

enum CalendarTypeView {
  "days",
  "months",
  "years"
}

const Calendar: FC<DatePickerProps> = ({
  date,
  format = DATE_TIME_FORMAT,
  onChange,
}) => {
  const { isMobile } = useDeviceDetect();

  const [viewType, setViewType] = useState<CalendarTypeView>(CalendarTypeView.days);
  const [viewDate, setViewDate] = useState(date);
  const [selectDate, setSelectDate] = useState(date);

  const today = getNowInTimezone();
  const viewDateIsToday = today.format(DATE_FORMAT) === viewDate.format(DATE_FORMAT);

  const toggleDisplayYears = () => {
    setViewType(prev => prev === CalendarTypeView.years ? CalendarTypeView.days : CalendarTypeView.years);
  };

  const handleChangeViewDate = (date: VmDate) => {
    setViewDate(date);
    setViewType(prev => prev === CalendarTypeView.years ? CalendarTypeView.months : CalendarTypeView.days);
  };

  const handleChangeSelectDate = (date: VmDate) => {
    setSelectDate(date);
  };

  const handleToday = () => {
    setViewDate(today);
  };

  useEffect(() => {
    if (selectDate.format() === date.format()) return;
    onChange(selectDate.format(format));
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- must only fire on user-driven selectDate changes; including date/format/onChange would re-fire onChange with the stale selectDate when the date prop changes externally (the effect below resets selectDate then), and on every parent re-render that passes a new onChange reference
  }, [selectDate]);

  useEffect(() => {
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- resets internal view/select state to match the externally controlled date prop
    setViewDate(date);
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- resets internal view/select state to match the externally controlled date prop
    setSelectDate(date);
  }, [date]);

  return (
    <div
      className={classNames({
        "vm-calendar": true,
        "vm-calendar_mobile": isMobile,
      })}
    >
      <CalendarHeader
        viewDate={viewDate}
        onChangeViewDate={handleChangeViewDate}
        toggleDisplayYears={toggleDisplayYears}
        showArrowNav={viewType === CalendarTypeView.days}
      />
      {viewType === CalendarTypeView.days && (
        <CalendarBody
          viewDate={viewDate}
          selectDate={selectDate}
          onChangeSelectDate={handleChangeSelectDate}
        />
      )}
      {viewType === CalendarTypeView.years && (
        <YearsList
          viewDate={viewDate}
          onChangeViewDate={handleChangeViewDate}
        />
      )}
      {viewType === CalendarTypeView.months && (
        <MonthsList
          selectDate={selectDate}
          viewDate={viewDate}
          onChangeViewDate={handleChangeViewDate}
        />
      )}
      {!viewDateIsToday && (viewType === CalendarTypeView.days) && (
        <div className="vm-calendar-footer">
          <Button
            variant="text"
            size="small"
            onClick={handleToday}
          >
            Show today
          </Button>
        </div>
      )}
    </div>
  );
};

export default Calendar;
