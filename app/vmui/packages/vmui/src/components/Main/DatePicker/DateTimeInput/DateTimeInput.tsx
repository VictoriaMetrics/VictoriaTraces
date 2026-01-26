import { FC, useEffect, useRef, useState, RefObject, ChangeEvent, KeyboardEvent } from "react";
import { CalendarIcon } from "../../Icons";
import DatePicker from "../DatePicker";
import Button from "../../Button/Button";
import { DATE_TIME_FORMAT } from "../../../../constants/date";
import InputMask from "react-input-mask";
import { IMaskInput } from "react-imask";
import dayjs from "dayjs";
import classNames from "classnames";
import "./style.scss";

const formatStringDate = (val: string) => {
  return dayjs(val).isValid() ? dayjs.tz(val).format(DATE_TIME_FORMAT) : val;
};

interface DateTimeInputProps {
  value?: string;
  label: string;
  pickerLabel: string;
  pickerRef: RefObject<HTMLDivElement>;
  onChange: (date: string) => void;
  onEnter: () => void;
}

const DateTimeInput: FC<DateTimeInputProps> = ({
  value = "",
  label,
  pickerLabel,
  pickerRef,
  onChange,
  onEnter
}) => {
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [inputRef, setInputRef] = useState<HTMLInputElement | null>(null);

  const [maskedValue, setMaskedValue] = useState(formatStringDate(value));
  const [focusToTime, setFocusToTime] = useState(false);
  const [awaitChangeForEnter, setAwaitChangeForEnter] = useState(false);
  const error = dayjs(maskedValue).isValid() ? "" : "Invalid date format";

  const handleMaskedChangeValue = (value: string) => {
    setMaskedValue(value);
  };

  const handleBlur = () => {
    onChange(maskedValue);
  };

  const handleKeyUp = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      onChange(maskedValue);
      setAwaitChangeForEnter(true);
    }
  };

  const handleChangeDate = (val: string) => {
    setMaskedValue(val);
    setFocusToTime(true);
  };

  useEffect(() => {
    const newValue = formatStringDate(value);
    if (newValue !== maskedValue) {
      setMaskedValue(newValue);
    }

    if (awaitChangeForEnter) {
      onEnter();
      setAwaitChangeForEnter(false);
    }
  }, [value]);

  useEffect(() => {
    if (focusToTime && inputRef) {
      inputRef.focus();
      inputRef.setSelectionRange(11, 11);
      setFocusToTime(false);
    }
  }, [focusToTime]);

  return (
    <div
      className={classNames({
        "vm-date-time-input": true,
        "vm-date-time-input_error": error
      })}
    >
      <label>{label}</label>
      <IMaskInput
        // 你原来的 mask 等价写法：0 表示数字
        mask={"0000-00-00 00:00:00"}
        // 想要“maskChar={null}”那种体验（不显示占位符），用 lazy
        lazy={true}
        // placeholder 仍然可以保留
        placeholder="YYYY-MM-DD HH:mm:ss"

        // 受控值：用 unmasked 或 value 都行；这里用 value（带分隔符）
        value={maskedValue}

        // 你的 inputRef：react-imask 用 inputRef
        inputRef={setInputRef}

        tabIndex={1}
        autoCapitalize="none"
        inputMode="numeric"

        // react-imask 推荐用 onAccept（值变化时触发）
        onAccept={(value) => {
          // value 是格式化后的字符串：YYYY-MM-DD HH:mm:ss
          // 你原来的 handleMaskedChange 如果吃 event，就改成吃 string
          handleMaskedChangeValue(String(value));
        }}

        // 这些原生事件可以直接挂
        onBlur={handleBlur}
        onKeyUp={handleKeyUp}
      />
      {error && (
        <span className="vm-date-time-input__error-text">{error}</span>
      )}
      <div
        className="vm-date-time-input__icon"
        ref={wrapperRef}
      >
        <Button
          variant="text"
          color="gray"
          size="small"
          startIcon={<CalendarIcon />}
          ariaLabel="calendar"
        />
      </div>
      <DatePicker
        label={pickerLabel}
        ref={pickerRef}
        date={maskedValue}
        onChange={handleChangeDate}
        targetRef={wrapperRef}
      />
    </div>
  );
};

export default DateTimeInput;
