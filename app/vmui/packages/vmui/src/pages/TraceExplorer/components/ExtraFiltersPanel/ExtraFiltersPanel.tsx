import { FC, useMemo } from "preact/compat";
import { CloseIcon } from "../../../../components/Main/Icons";
import Tooltip from "../../../../components/Main/Tooltip";
import { DURATION_MAX_FIELD, DURATION_MIN_FIELD, ExtraFilter } from "../../hooks/useExtraFilters";
import { buildTagClause, formatDurationForInput } from "../../utils";
import "./style.scss";

interface Props {
  extraFilters: ExtraFilter[];
  onRemove: (field: string, value: string) => void;
}

function getLabel(filter: ExtraFilter): string {
  if (filter.field === DURATION_MIN_FIELD) return `duration >= ${formatDurationForInput(Number(filter.value))}`;
  if (filter.field === DURATION_MAX_FIELD) return `duration <= ${formatDurationForInput(Number(filter.value))}`;
  return buildTagClause(filter.field, filter.value);
}

const ExtraFiltersPanel: FC<Props> = ({ extraFilters, onRemove }) => {
  const items = useMemo(() => extraFilters.map(f => ({
    ...f,
    label: getLabel(f),
  })), [extraFilters]);

  if (!items.length) return null;

  return (
    <div className="vm-extra-filters-panel">
      {items.map(item => (
        <Tooltip
          key={`${item.field}:${item.value}`}
          title={item.value}
        >
          <div className="vm-extra-filters-panel-item">
            <div className="vm-extra-filters-panel-item__label">{item.label}</div>
            <div
              className="vm-extra-filters-panel-item__remove"
              onClick={() => onRemove(item.field, item.value)}
            >
              <CloseIcon/>
            </div>
          </div>
        </Tooltip>
      ))}
    </div>
  );
};

export default ExtraFiltersPanel;
