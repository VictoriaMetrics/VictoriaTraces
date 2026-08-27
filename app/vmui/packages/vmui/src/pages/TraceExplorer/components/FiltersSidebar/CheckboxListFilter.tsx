import { FC, ReactNode } from "preact/compat";
import Checkbox from "../../../../components/Main/Checkbox";
import { SpinnerIcon } from "../../../../components/Main/Icons";

interface Props {
  values: string[];
  selected: string[];
  onToggle: (value: string) => void;
  loading?: boolean;
  emptyText?: string;
  renderIcon?: (value: string) => ReactNode;
}

const CheckboxListFilter: FC<Props> = ({
  values,
  selected,
  onToggle,
  loading = false,
  emptyText = "No values found",
  renderIcon,
}) => {
  return (
    <div className="vm-filters-checkbox-list">
      {loading && (
        <div className="vm-filters-checkbox-list__status vm-filters-checkbox-list__status_loading">
          <SpinnerIcon/>
        </div>
      )}
      {!loading && !values.length && (
        <div className="vm-filters-checkbox-list__status">{emptyText}</div>
      )}
      {!loading && !!values.length && (
        <div className="vm-filters-checkbox-list__items">
          {values.map(value => (
            <div
              key={value}
              className="vm-filters-checkbox-list__item"
              onClick={() => onToggle(value)}
            >
              <Checkbox
                size="small"
                checked={selected.includes(value)}
                color={selected.includes(value) ? "primary" : "gray"}
              />
              {renderIcon?.(value)}
              <span className="vm-filters-checkbox-list__item-label">{value}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default CheckboxListFilter;
