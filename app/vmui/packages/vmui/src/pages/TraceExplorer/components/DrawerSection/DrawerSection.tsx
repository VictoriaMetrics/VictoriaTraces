import { ReactNode } from "preact/compat";
import "./style.scss";

interface Props {
  title: ReactNode;
  count?: number;
  children: ReactNode;
}

export default function DrawerSection({ title, count, children }: Props) {
  return (
    <div className="vm-trace-info-drawer-section">
      <div className="vm-trace-info-drawer-section__title">
        {title}
        {count !== undefined && (
          <span className="vm-trace-info-drawer-section__count"> ({count})</span>
        )}
      </div>
      <div className="vm-trace-info-drawer-section__body">
        {children}
      </div>
    </div>
  );
}
