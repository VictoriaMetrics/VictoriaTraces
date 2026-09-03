import { FC, ReactNode } from "preact/compat";
import classNames from "classnames";
import "./style.scss";

interface Props {
  children: ReactNode
  variant?: "header" | "body";
  isActive?: boolean;
  className?: string;
  onClick?: (e: MouseEvent) => void;
}

const TableRow: FC<Props> = ({
  children,
  variant = "body",
  isActive,
  className: extraClassName,
  onClick,
}) => {
  const isHeader = variant === "header";

  const className = classNames({
    "vm-table-row": true,
    "vm-table-row_header": isHeader,
    "vm-table-row_active": isActive,
    "vm-table-row_clickable": Boolean(onClick)
  }, extraClassName);

  return (
    <tr
      className={className}
      onClick={onClick}
    >
      {children}
    </tr>
  );
};

export default TableRow;
