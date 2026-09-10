import { useState, useMemo, useRef, useEffect } from "preact/compat";
import { getComparator, stableSort } from "./helpers";
import { OrderDir } from "../../types";
import TableHeaderCell from "./TableHeaderCell";
import TableCell from "./TableCell";
import TableCellActions from "./TableCell/TableCellActions";
import TableRow from "./TableRow";
import "./style.scss";
import { useTableColumnPrefs } from "./hooks/useTableColumnPrefs";
import { Size, useResizeObserver } from "../../hooks/useResizeObserver";
import { useDebounceCallback } from "../../hooks/useDebounceCallback";
import { ColumnKey, TableProps } from "./types";
import { useDragColumn } from "./hooks/useDragColumn";

const DEFAULT_COLUMN_WIDTH = 150;
const ACTIONS_COLUMN_WIDTH = 24;

const Table = <T extends object>({
  tableId,
  rows,
  columns,
  defaultOrder,
  isActiveRow,
  rowClassName,
  onClickRow,
  actionsRender,
  paginationOffset,
  applyViewColumns = () => {
  },
}: TableProps<T>) => {
  const { getColumnPrefs, updateColumnPref } = useTableColumnPrefs(tableId);

  const draggableKeys = columns.filter(col => col.options.draggable).map(col => String(col.key));

  const dragController = useDragColumn({
    axis: "x" as "x" | "y",
    arr: draggableKeys,
    onChange: (nextKeys: string[]) => {
      let i = 0;
      const draggableSet = new Set(draggableKeys);
      const merged = columns.map(col => {
        const key = String(col.key);
        return draggableSet.has(key) ? nextKeys[i++] : key;
      });
      applyViewColumns({ type: "replace", columnKeys: merged });
    }
  });

  const [orderBy, setOrderBy] = useState<ColumnKey<T>>(defaultOrder?.key || columns[0]?.key);
  const [orderDir, setOrderDir] = useState<OrderDir>(defaultOrder?.dir || "desc");

  useEffect(() => {
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- resets sort state to the default (or first column) when defaultOrder/columns change externally
    setOrderBy(defaultOrder?.key || columns[0]?.key);
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- resets sort state to the default (or first column) when defaultOrder/columns change externally
    setOrderDir(defaultOrder?.dir || "desc");
  }, [defaultOrder?.key, defaultOrder?.dir, columns]);

  const sortedList = useMemo(() => {
    const [startIndex, endIndex] = paginationOffset;
    return stableSort<T>(rows, getComparator(orderDir, orderBy)).slice(startIndex, endIndex);
  }, [rows, orderBy, orderDir, paginationOffset]);

  const sortPack = useMemo(() => ({
    key: orderBy,
    dir: orderDir,
    onChange: (key: ColumnKey<T>, orderDir: OrderDir) => {
      setOrderDir(orderDir);
      setOrderBy(key);
    },
  }), [orderBy, orderDir]);

  const tableRef = useRef<HTMLTableElement>(null);
  const [size, setSize] = useState<Size>({ width: 0, height: 0 });
  const onResizeTable = useDebounceCallback(setSize, 60);
  useResizeObserver({ ref: tableRef, onResize: onResizeTable });
  const hasCustomWidths = columns.some(col => getColumnPrefs(col.key)?.width != null);
  const tableStyle = hasCustomWidths ? {
    tableLayout: "fixed" as const,
    width: `${columns.reduce((sum, col) => sum + (getColumnPrefs(col.key)?.width ?? DEFAULT_COLUMN_WIDTH), 0)
      + (actionsRender ? ACTIONS_COLUMN_WIDTH : 0)}px`,
  } : undefined;

  const tableBody = useMemo(() => sortedList.map((row, rowIndex) => (
    <TableRow
      // eslint-disable-next-line @eslint-react/no-array-index-key -- Table<T> is generic with no guaranteed unique row id from callers; rowIndex is stable within the current sorted+paginated slice
      key={rowIndex}
      isActive={isActiveRow && isActiveRow(row as T)}
      className={rowClassName?.(row as T)}
      onClick={(e) => onClickRow && onClickRow(row as T, e)}
    >
      {columns.map((col) => (
        <TableCell
          key={String(col.key)}
          column={col}
          columnPrefs={getColumnPrefs(col.key)}
          row={row as T}
          rowIdx={rowIndex}
        />
      ))}

      {actionsRender && (
        <TableCellActions
          row={row as T}
          actionsRender={actionsRender}
        />
      )}
    </TableRow>
  )), [sortedList, columns, isActiveRow, rowClassName, onClickRow, actionsRender, getColumnPrefs]);

  return (
    <div className="vm-table-scroll">
      <table
        className="vm-table"
        ref={tableRef}
        style={tableStyle}
      >
        <thead className="vm-table-header">
          <TableRow variant="header">
            {columns.map((column, idx) => (
              <TableHeaderCell
                key={column.key}
                idx={idx}
                column={column}
                sort={sortPack}
                prefs={getColumnPrefs(column.key)}
                containerSize={size}
                dragController={dragController}
                applyViewColumns={applyViewColumns}
                onChangePref={updateColumnPref}
              />
            ))}

            {actionsRender && <th className="vm-table-cell vm-table-cell-header vm-table-cell_actions"/>}
          </TableRow>
        </thead>
        <tbody className="vm-table-body">
          {tableBody}
        </tbody>
      </table>
    </div>
  );
};

export default Table;
