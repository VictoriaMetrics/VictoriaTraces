import React, { FC } from "preact/compat";
import Switch from "../../Main/Switch/Switch";
import useDeviceDetect from "../../../hooks/useDeviceDetect";
import { useTracesDispatch, useTracesState } from "../../../state/tracesPanel/TracesStateContext";

const TraceParsingSwitches: FC = () => {
  const { isMobile } = useDeviceDetect();
  const { markdownParsing, ansiParsing } = useTracesState();
  const dispatch = useTracesDispatch();

  const handleChangeMarkdownParsing = (val: boolean) => {
    dispatch({ type: "SET_MARKDOWN_PARSING", payload: val });

    if (ansiParsing) {
      dispatch({ type: "SET_ANSI_PARSING", payload: false });
    }
  };

  const handleChangeAnsiParsing = (val: boolean) => {
    dispatch({ type: "SET_ANSI_PARSING", payload: val });

    if (markdownParsing) {
      dispatch({ type: "SET_MARKDOWN_PARSING", payload: false });
    }
  };

  return (
    <>
      <div className="vm-group-traces-configurator-item">
        <Switch
          label={"Enable markdown parsing"}
          value={markdownParsing}
          onChange={handleChangeMarkdownParsing}
          fullWidth={isMobile}
        />
        <div className="vm-group-traces-configurator-item__info">
          Toggle this switch to enable or disable the Markdown formatting for trace entries.
          Enabling this will parse trace texts to Markdown.
        </div>
      </div>
      <div className="vm-group-traces-configurator-item">
        <Switch
          label={"Enable ANSI parsing"}
          value={ansiParsing}
          onChange={handleChangeAnsiParsing}
          fullWidth={isMobile}
        />
        <div className="vm-group-traces-configurator-item__info">
          Toggle this switch to enable or disable ANSI escape sequence parsing for trace entries.
          Enabling this will interpret ANSI codes to render colored trace output.
        </div>
      </div>
    </>
  );
};

export default TraceParsingSwitches;
