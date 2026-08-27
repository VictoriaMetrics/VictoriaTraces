import { forwardRef, useCallback, useEffect, useImperativeHandle, useState } from "preact/compat";
import { ErrorTypes } from "../../../../types";
import TextField from "../../../Main/TextField";
import { isValidHttpUrl } from "../../../../utils/url";
import Button from "../../../Main/Button";
import { StorageIcon } from "../../../Main/Icons";
import Tooltip from "../../../Main/Tooltip";
import { getFromStorage, removeFromStorage, saveToStorage } from "../../../../utils/storage";
import useBoolean from "../../../../hooks/useBoolean";
import { ChildComponentHandle } from "../GlobalSettings";
import { useAppDispatch, useAppState } from "../../../../state/common/StateContext";

interface ServerConfiguratorProps {
  onClose: () => void;
}

const tooltipSave = {
  enable: "Enable to save the modified server URL to local storage, preventing reset upon page refresh.",
  disable: "Disable to stop saving the server URL to local storage, reverting to the default URL on page refresh."
};

// eslint-disable-next-line @eslint-react/no-forward-ref -- preact/compat still requires forwardRef; a plain function component does not receive 'ref' as a prop here
const ServerConfigurator = forwardRef<ChildComponentHandle, ServerConfiguratorProps>(({ onClose }, ref) => {
  const { serverUrl: stateServerUrl } = useAppState();
  const dispatch = useAppDispatch();

  const {
    value: enabledStorage,
    toggle: handleToggleStorage,
  } = useBoolean(!!getFromStorage("SERVER_URL"));

  const [serverUrl, setServerUrl] = useState(stateServerUrl);
  const [error, setError] = useState("");

  const handleChange = (val: string) => {
    const value = val || "";
    setServerUrl(value);
    setError("");
  };

  const handleApply = useCallback(() => {
    dispatch({ type: "SET_SERVER", payload: serverUrl });
    onClose();
  }, [serverUrl, dispatch, onClose]);

  useEffect(() => {
    // derives a validation error from the applied server URL; setError("") on edit (see handleChange) keeps it from showing while the user is typing
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- syncs validation error to the external stateServerUrl, not to local edits
    if (!stateServerUrl) setError(ErrorTypes.emptyServer);
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- syncs validation error to the external stateServerUrl, not to local edits
    if (!isValidHttpUrl(stateServerUrl)) setError(ErrorTypes.validServer);
  }, [stateServerUrl]);

  useEffect(() => {
    if (enabledStorage) {
      saveToStorage("SERVER_URL", serverUrl);
    } else {
      removeFromStorage(["SERVER_URL"]);
    }
  }, [enabledStorage, serverUrl]);

  useEffect(() => {
    // the tenant selector can change the serverUrl
    if (stateServerUrl === serverUrl) return;
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- syncs local editable value to the external stateServerUrl (e.g. tenant selector changes it)
    setServerUrl(stateServerUrl);
    // 'serverUrl' intentionally excluded: including it would re-run this on every keystroke and revert the user's in-progress edit back to stateServerUrl
    // eslint-disable-next-line @eslint-react/exhaustive-deps
  }, [stateServerUrl]);

  useImperativeHandle(ref, () => ({ handleApply }), [handleApply]);

  return (
    <div>
      <div className="vm-server-configurator__title">
        Server URL
      </div>
      <div className="vm-server-configurator-url">
        <TextField
          autofocus
          value={serverUrl}
          error={error}
          onChange={handleChange}
          onEnter={handleApply}
          inputmode="url"
        />
        <Tooltip title={enabledStorage ? tooltipSave.disable : tooltipSave.enable}>
          <Button
            className="vm-server-configurator-url__button"
            variant="text"
            color={enabledStorage ? "primary" : "gray"}
            onClick={handleToggleStorage}
            startIcon={<StorageIcon/>}
          />
        </Tooltip>
      </div>
    </div>
  );
});

export default ServerConfigurator;
