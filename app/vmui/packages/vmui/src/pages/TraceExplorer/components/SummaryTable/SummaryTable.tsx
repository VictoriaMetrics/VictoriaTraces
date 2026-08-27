import { Fragment, ReactNode } from "preact/compat";

import "./style.scss";

type SummaryTableItem = { key: string; label: ReactNode; value: ReactNode };

type SummaryTableProps = {
  items: SummaryTableItem[];
};

export default function SummaryTable({ items }: SummaryTableProps) {
  return (
    <div className="vm-trace-info-drawer-summary">
      {items.map(({ key, label, value }) => (
        <Fragment key={key}>
          <span className="vm-trace-info-drawer-summary__label">{label}</span>
          <span className="vm-trace-info-drawer-summary__value">{value}</span>
        </Fragment>
      ))}
    </div>
  );
}
