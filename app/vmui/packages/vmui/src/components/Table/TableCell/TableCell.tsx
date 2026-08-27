import { memo } from "preact/compat";
import classNames from "classnames";
import "./style.scss";
import { ColumnPrefs } from "../hooks/useTableColumnPrefs";
import { type Column } from "../types";

interface Props<T> {
  row: T;
  rowIdx: number;
  column: Column<T>;
  columnPrefs?: ColumnPrefs;
}

const TableCell = <T extends object>({
  row,
  rowIdx,
  column,
  columnPrefs = {}
}: Props<T>) => {
  const { width, wrapped } = columnPrefs;

  const className = classNames({
    "vm-table-cell": true,
    "vm-table-cell_wrapped": wrapped,
    [`${column.className}`]: column.className
  });

  // Applied on the <td> itself (not just the content div below) so an explicit resize
  // width wins over a column's own width CSS (e.g. the stretchy last column's width:100%)
  // — inline style always beats a class rule for the same property.
  const style = width ? { width: `${width}px` } : undefined;

  return (
    <td
      className={className}
      style={style}
    >
      <div
        className="vm-table-cell__content"
        style={width ? { width: `${width}px`, maxWidth: "none" } : undefined}
      >
        {column.render ? column.render(row, rowIdx) : row[column.key] ?? "-"}
      </div>
    </td>
  );
};

export default memo(TableCell) as typeof TableCell;
