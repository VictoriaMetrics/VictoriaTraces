import { FC, useCallback, useState } from "preact/compat";
import TextField from "../../Main/TextField";
import { useEffect } from "react";
import { TRACES_MAX_LIMIT } from "../../../constants/logs";
import { formatNumber } from "../../../utils/number";

type Props = {
  limit: number;
  onChangeLimit: (val: number) => void;
  onPressEnter: () => void;
  onError?: (error: boolean) => void;
}

const isValidLimit = (number: number) => {
  if (isNaN(number) || number <= 0) {
    return {
      isValid: false,
      errorMsg: "Number must be > 0"
    };
  } else if (number > TRACES_MAX_LIMIT) {
    return {
      isValid: false,
      errorMsg: `Max limit is ${formatNumber(TRACES_MAX_LIMIT)}`
    };
  }

  return {
    isValid: true,
    errorMsg: ""
  };
};

const TracesLimitInput: FC<Props> = ({ limit, onChangeLimit, onPressEnter, onError }) => {
  const [errorLimit, setErrorLimit] = useState("");
  const [limitInput, setLimitInput] = useState(limit);

  const handleChangeLimit = useCallback((val: string) => {
    const number = +val;
    setLimitInput(number);
    const { isValid, errorMsg } = isValidLimit(number);
    if (!isValid) {
      setErrorLimit(errorMsg);
    } else {
      setErrorLimit("");
      onChangeLimit(number);
    }
  }, [onChangeLimit]);

  useEffect(() => {
    onError && onError(Boolean(errorLimit));
  }, [errorLimit, onError]);

  const handlePressEnter = useCallback(() => {
    if (errorLimit) return;
    onPressEnter();
  }, [errorLimit, onPressEnter]);

  useEffect(() => {
    const { isValid, errorMsg } = isValidLimit(limit);
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- syncs the input echo/error state to the `limit` prop (including validating it on mount); must run on prop change, not just user typing
    isValid ? setErrorLimit("") : setErrorLimit(errorMsg);
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- see above
    setLimitInput(limit);
  }, [limit]);

  return (
    <TextField
      label="Limit"
      type="number"
      value={limitInput}
      error={errorLimit}
      onChange={handleChangeLimit}
      onEnter={handlePressEnter}
    />
  );
};

export default TracesLimitInput;
