import { FC, useEffect, useMemo, useRef } from "preact/compat";
import { GraphOptions, GRAPH_STYLES } from "../types";
import Switch from "../../../Main/Switch/Switch";
import "./style.scss";
import useStateSearchParams from "../../../../hooks/useStateSearchParams";
import { useSearchParams } from "react-router-dom";
import Button from "../../../Main/Button/Button";
import { SettingsIcon } from "../../../Main/Icons";
import Tooltip from "../../../Main/Tooltip/Tooltip";
import Popper from "../../../Main/Popper/Popper";
import Modal from "../../../Main/Modal/Modal";
import useBoolean from "../../../../hooks/useBoolean";
import useDeviceDetect from "../../../../hooks/useDeviceDetect";

const STACKED_PARAM = "stacked";
const STACKED_DEFAULT = "false";
const FILL_PARAM = "fill";
const FILL_DEFAULT = "true";

interface Props {
  onChange: (options: GraphOptions) => void;
}

const BarHitsOptions: FC<Props> = ({ onChange }) => {
  const { isMobile } = useDeviceDetect();
  const [searchParams, setSearchParams] = useSearchParams();
  const optionsButtonRef = useRef<HTMLDivElement>(null);
  const {
    value: openOptions,
    toggle: toggleOpenOptions,
    setFalse: handleCloseOptions,
  } = useBoolean(false);

  const [stacked, setStacked] = useStateSearchParams(STACKED_DEFAULT, STACKED_PARAM);
  const [fill, setFill] = useStateSearchParams(FILL_DEFAULT, FILL_PARAM);

  const options: GraphOptions = useMemo(() => ({
    graphStyle: GRAPH_STYLES.BAR,
    stacked: stacked === "true",
    fill: fill === "true",
  }), [stacked, fill]);

  const updateParam = (key: string, value: string, defaultValue: string) => {
    value === defaultValue ? searchParams.delete(key) : searchParams.set(key, value);
    setSearchParams(searchParams, { replace: true });
  };

  const handleChangeFill = (val: boolean) => {
    setFill(`${val}`);
    updateParam(FILL_PARAM, `${val}`, FILL_DEFAULT);
  };

  const handleChangeStacked = (val: boolean) => {
    setStacked(`${val}`);
    updateParam(STACKED_PARAM, `${val}`, STACKED_DEFAULT);
  };

  useEffect(() => {
    onChange(options);
  }, [options, onChange]);

  const settings = (
    <div className="vm-bar-hits-options-settings">
      <div className="vm-bar-hits-options-settings-item">
        <Switch
          label={"Stacked"}
          value={stacked === "true"}
          onChange={handleChangeStacked}
        />
      </div>
      <div className="vm-bar-hits-options-settings-item">
        <Switch
          label={"Fill"}
          value={fill === "true"}
          onChange={handleChangeFill}
        />
      </div>
    </div>
  );

  return (
    <div className="vm-bar-hits-options">
      <div ref={optionsButtonRef}>
        <Tooltip title="Graph settings">
          <Button
            variant="text"
            color="primary"
            startIcon={<SettingsIcon/>}
            onClick={toggleOpenOptions}
            aria-label="graph settings"
          />
        </Tooltip>
      </div>
      {isMobile ? (
        openOptions && (
          <Modal
            title={"Graph settings"}
            onClose={handleCloseOptions}
          >
            {settings}
          </Modal>
        )
      ) : (
        <Popper
          open={openOptions}
          placement="bottom-right"
          onClose={handleCloseOptions}
          buttonRef={optionsButtonRef}
          title={"Graph settings"}
        >
          {settings}
        </Popper>
      )}
    </div>
  );
};

export default BarHitsOptions;
