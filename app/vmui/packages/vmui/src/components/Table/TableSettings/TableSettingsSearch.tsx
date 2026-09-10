import { FC } from "preact/compat";
import { KeyboardIcon, SearchIcon } from "../../Main/Icons";
import "./style.scss";
import TextField from "../../Main/TextField";
import Button from "../../Main/Button";
import Tooltip from "../../Main/Tooltip";
import TableSettingsHotKeys from "./TableSettingsHotKeys";

interface Props {
  searchColumn: string;
  onChange: (value: string) => void;
}

const TableSettingsSearch: FC<Props> = ({
  searchColumn,
  onChange
}) => {
  return (
    <div className="vm-table-settings-section-search">
      <TextField
        placeholder={"Search columns"}
        startIcon={<SearchIcon/>}
        value={searchColumn}
        onChange={onChange}
        type="search"
      />

      <Tooltip title={<TableSettingsHotKeys/>}>
        <Button
          startIcon={<KeyboardIcon/>}
          variant="text"
          color="gray"
          aria-label="Search keyboard shortcuts"
        />
      </Tooltip>
    </div>
  );
};

export default TableSettingsSearch;
