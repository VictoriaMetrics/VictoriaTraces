import { FC, useMemo } from "preact/compat";
import { Traces } from "../../../api/types";
import "./style.scss";
import classNames from "classnames";
import GroupTracesFieldRow from "./GroupTracesFieldRow";
import { useLocalStorageBoolean } from "../../../hooks/useLocalStorageBoolean";

interface Props {
  trace: Traces;
  hideGroupButton?: boolean;
}

const GroupTracesFields: FC<Props> = ({ trace, hideGroupButton }) => {
  const sortedFields = useMemo(() => {
    return Object.entries(trace)
      .sort(([aKey], [bKey]) => aKey.localeCompare(bKey));
  }, [trace]);

  const [disabledHovers] = useLocalStorageBoolean("TRACES_DISABLED_HOVERS");

  return (
    <div
      className={classNames({
        "vm-group-traces-row-fields": true,
        "vm-group-traces-row-fields_interactive": !disabledHovers
      })}
    >
      <table>
        <tbody>
          {sortedFields.map(([key, value]) => (
            <GroupTracesFieldRow
              key={key}
              field={key}
              value={value}
              hideGroupButton={hideGroupButton}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default GroupTracesFields;
