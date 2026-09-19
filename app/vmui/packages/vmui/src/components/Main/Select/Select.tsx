import { FC, useCallback, useEffect, useMemo, useRef, useState, RefObject, FormEvent, MouseEvent } from "preact/compat";
import classNames from "classnames";
import { ArrowDropDownIcon, CloseIcon } from "../Icons";
import Autocomplete from "../Autocomplete";
import { useAppState } from "../../../state/common/StateContext";
import "./style.scss";
import useDeviceDetect from "../../../hooks/useDeviceDetect";
import MultipleSelectedValue from "./MultipleSelectedValue";
import useEventListener from "../../../hooks/useEventListener";
import useClickOutside from "../../../hooks/useClickOutside";

interface SelectProps {
  value: string | string[]
  list: string[]
  // Optional secondary text shown on the right side of each dropdown row (e.g. a timestamp
  // or count next to a name) - parallel to `list`, same length/order.
  metaList?: string[]
  label?: string
  placeholder?: string
  noOptionsText?: string
  clearable?: boolean
  searchable?: boolean
  autofocus?: boolean
  disabled?: boolean
  onChange: (value: string) => void

  onOpen?(open: boolean): void
}

const Select: FC<SelectProps> = ({
  value,
  list,
  metaList,
  label,
  placeholder,
  noOptionsText,
  clearable = false,
  searchable = false,
  autofocus,
  disabled,
  onChange,
  onOpen
}) => {
  const { isDarkTheme } = useAppState();
  const { isMobile } = useDeviceDetect();

  const [search, setSearch] = useState("");
  const autocompleteAnchorElRef = useRef<HTMLDivElement>(null);
  const [wrapperRef, setWrapperRef] = useState<RefObject<HTMLElement> | null>(null);
  const [openList, setOpenList] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);

  const isMultiple = Array.isArray(value);
  const selectedValues = Array.isArray(value) ? value : undefined;
  const hideInput = isMobile && isMultiple && !!selectedValues?.length;

  const textFieldValue = useMemo(() => {
    if (openList) return search;
    return Array.isArray(value) ? "" : value;
  }, [value, search, openList]);

  const autocompleteValue = useMemo(() => openList && search ? search : "", [search, openList]);

  const clearFocus = useCallback(() => {
    if (inputRef.current) {
      inputRef.current.blur();
    }
  }, []);

  const handleCloseList = useCallback(() => {
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- shared close handler, also used as an event/click-outside handler; the disabled-sync effect below is one of several callers
    setOpenList(false);
    clearFocus();
  }, [clearFocus]);

  const handleFocus = () => {
    if (disabled) return;
    setOpenList(true);
  };

  const handleBlur = () => {
    list.includes(search) && onChange(search);
  };

  const handleToggleList = (e: MouseEvent<HTMLDivElement>) => {
    if (e.target instanceof HTMLInputElement || disabled) return;
    setOpenList(prev => !prev);
  };

  const handleSelected = (val: string) => {
    setSearch("");
    onChange(val);
    if (!isMultiple) handleCloseList();
    if (isMultiple && inputRef.current) inputRef.current.focus();
  };

  const handleChange = (e: FormEvent<HTMLInputElement>) => {
    setSearch((e.target as HTMLInputElement).value);
  };

  const createHandleClick = (value: string) => (e: MouseEvent<HTMLDivElement>) => {
    handleSelected(value);
    e.stopPropagation();
  };

  const handleKeyUp = (e: KeyboardEvent) => {
    if (inputRef.current !== e.target) {
      setOpenList(false);
    }
  };

  useEffect(() => {
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- must react to openList changes from any source, including the child Autocomplete's onOpenAutocomplete={setOpenList}
    setSearch("");
    if (openList && inputRef.current) {
      inputRef.current.focus();
    }
    if (!openList) clearFocus();
  }, [openList, inputRef, clearFocus]);

  useEffect(() => {
    if (!autofocus || !inputRef.current || isMobile) return;
    inputRef.current.focus();
  }, [autofocus, inputRef, isMobile]);

  useEffect(() => {
    if (disabled) handleCloseList();
  }, [disabled, handleCloseList]);

  useEffect(() => {
    onOpen && onOpen(openList);
  }, [openList, onOpen]);

  useEventListener("keyup", handleKeyUp);
  useClickOutside(autocompleteAnchorElRef, handleCloseList, wrapperRef);

  return (
    <div
      className={classNames({
        "vm-select": true,
        "vm-select_dark": isDarkTheme,
        "vm-select_disabled": disabled
      })}
    >
      <div
        className="vm-select-input"
        onClick={handleToggleList}
        ref={autocompleteAnchorElRef}
      >
        <div className="vm-select-input-content">
          {!!selectedValues?.length && (
            <MultipleSelectedValue
              values={selectedValues}
              onRemoveItem={handleSelected}
            />
          )}
          {!hideInput && (
            <input
              value={textFieldValue}
              type="text"
              placeholder={placeholder}
              onInput={handleChange}
              onFocus={handleFocus}
              onBlur={handleBlur}
              ref={inputRef}
              readOnly={isMobile || !searchable}
              autoComplete="off"
            />
          )}
        </div>
        {label && <span className="vm-text-field__label">{label}</span>}
        {clearable && value && (
          <div
            className="vm-select-input__icon"
            onClick={createHandleClick("")}
          >
            <CloseIcon/>
          </div>
        )}
        <div
          className={classNames({
            "vm-select-input__icon": true,
            "vm-select-input__icon_open": openList
          })}
        >
          <ArrowDropDownIcon/>
        </div>
      </div>
      {!disabled && openList && (
        <Autocomplete
          label={label}
          value={autocompleteValue}
          options={list.map((el, i) => ({ value: el, meta: metaList?.[i] }))}
          anchor={autocompleteAnchorElRef}
          selected={selectedValues}
          minLength={0}
          fullWidth
          noOptionsText={noOptionsText}
          onSelect={handleSelected}
          onOpenAutocomplete={setOpenList}
          onChangeWrapperRef={setWrapperRef}
        />
      )}
    </div>
  );
};

export default Select;
