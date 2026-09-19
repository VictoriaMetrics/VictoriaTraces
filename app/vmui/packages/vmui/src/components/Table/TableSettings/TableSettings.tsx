import { createPortal, FC } from "preact/compat";
import Button from "../../Main/Button";
import "./style.scss";
import Tooltip from "../../Main/Tooltip";
import { TuneIcon } from "../../Main/Icons";
import useBoolean from "../../../hooks/useBoolean";
import TableSettingsDrawer from "./TableSettingsDrawer";
import { ViewColumnsAction } from "../hooks/useTableColumnView";
import { ColumnStats } from "../types";

export interface TableSettingsProps {
  columnKeys: string[];
  viewColumnKeys: string[];
  statsByKey: Map<string, ColumnStats>
  dispatchViewColumns: (action: ViewColumnsAction) => void;
}

const title = "Table settings";

const TableSettings: FC<TableSettingsProps> = (props) => {
  const {
    value: isOpenSettings,
    toggle: toggleOpenSettings,
    setFalse: handleClose,
  } = useBoolean(false);

  return (
    <>
      <Tooltip title={title}>
        <Button
          variant="text"
          startIcon={<TuneIcon/>}
          onClick={toggleOpenSettings}
          aria-label={title}
        />
      </Tooltip>

      {isOpenSettings && createPortal(
        <TableSettingsDrawer
          {...props}
          title={title}
          onClose={handleClose}
        />
        , document.body)}
    </>
  );
};

export default TableSettings;
