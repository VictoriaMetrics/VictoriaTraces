import { forwardRef, useCallback, useImperativeHandle, useState } from "react";
import { getConfigValue } from "jaeger-ui-lite/src/utils/config/get-config";
import { DEFAULT_LIMIT } from "jaeger-ui-lite/src/constants/search-form";
import TextField from "../../../Main/TextField/TextField";
import Button from "../../../Main/Button/Button";
import { RestartIcon } from "../../../Main/Icons";
import { ChildComponentHandle } from "../GlobalSettings";
import { useTracesSearchSettings } from "../../../../layouts/TracesLayout/TracesSearchSettingsContext";
import "./style.scss";

const TracesLimitConfigurator = forwardRef<ChildComponentHandle>((_props, ref) => {
  const { resultsLimit, setResultsLimit } = useTracesSearchSettings();
  const maxLimit = getConfigValue("search.maxLimit") as number;

  const [limit, setLimit] = useState(resultsLimit);
  const [error, setError] = useState("");

  const handleChange = (value: string) => {
    setLimit(value);
    const numeric = Number(value);
    setError(!value || numeric < 1 || numeric > maxLimit ? `Enter a number between 1 and ${maxLimit}` : "");
  };

  const handleReset = () => {
    handleChange(String(DEFAULT_LIMIT));
  };

  const handleApply = useCallback(() => {
    if (!error) {
      setResultsLimit(limit);
    }
  }, [limit, error, setResultsLimit]);

  useImperativeHandle(ref, () => ({ handleApply }), [handleApply]);

  return (
    <div className="vm-traces-limit-configurator">
      <div className="vm-server-configurator__title">
        Limit Results
        <div className="vm-traces-limit-configurator-title__reset">
          <Button
            variant="text"
            color="primary"
            size="small"
            startIcon={<RestartIcon/>}
            onClick={handleReset}
          >
            Reset
          </Button>
        </div>
      </div>
      <TextField
        label="Max number of traces to return"
        value={limit}
        error={error}
        type="number"
        onChange={handleChange}
        onEnter={handleApply}
      />
    </div>
  );
});

export default TracesLimitConfigurator;
