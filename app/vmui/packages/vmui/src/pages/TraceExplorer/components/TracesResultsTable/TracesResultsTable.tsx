import { FC, useEffect, useRef } from "preact/compat";
import { useSearchParams } from "react-router-dom";
import Table from "../../../../components/Table";
import TableSettings from "../../../../components/Table/TableSettings";
import Pagination from "../../../../components/Main/Pagination";
import SelectLimit from "../../../../components/Main/Pagination/SelectLimit";
import Button from "../../../../components/Main/Button";
import Tooltip from "../../../../components/Main/Tooltip";
import { TraceIcon } from "../../../../components/Main/Icons";
import { useTableColumnView } from "../../../../components/Table/hooks/useTableColumnView";
import { useTableTracesPaginate } from "./hooks/useTableTracesPaginate";
import { ALL_TRACES_COLUMN_KEYS, useTableTracesColumns } from "./hooks/useTableTracesColumns";
import { TraceSummary } from "../../hooks/useLogsqlTracesSearch";
import { TRACES_URL_PARAMS } from "../../../../constants/logs";
import { ColumnStats } from "../../../../components/Table/types";
import useSearchParamsFromObject from "../../../../hooks/useSearchParamsFromObject";
import "./style.scss";

const tableId = "table-traces-results";
const NO_COLUMN_STATS = new Map<string, ColumnStats>();

interface Props {
  results: TraceSummary[];
  activeTraceID?: string;
  onClickRow: (row: TraceSummary) => void;
  onOpenTrace: (row: TraceSummary) => void;
}

const TracesResultsTable: FC<Props> = ({ results, activeTraceID, onClickRow, onOpenTrace }) => {
  const [searchParams] = useSearchParams();
  const { setSearchParamsFromKeys } = useSearchParamsFromObject();
  const containerRef = useRef<HTMLDivElement>(null);

  const rowsPerPageRaw = searchParams.get(TRACES_URL_PARAMS.ROWS_PER_PAGE);
  const rowsPerPageNum = rowsPerPageRaw ? Number(rowsPerPageRaw) : 100;
  const rowsPerPage = isNaN(rowsPerPageNum) ? 0 : rowsPerPageNum;

  const { viewColumnKeys, dispatchViewColumns } = useTableColumnView(
    tableId,
    ALL_TRACES_COLUMN_KEYS,
    ALL_TRACES_COLUMN_KEYS
  );

  const { page, offset, onChangePage } = useTableTracesPaginate({ rowsPerPage, containerRef });
  const { tableColumns } = useTableTracesColumns({ keys: viewColumnKeys });

  useEffect(() => {
    onChangePage(1);
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- onChangePage is recreated every render by useTableTracesPaginate; depending on it would reset to page 1 on every render, not just on new results
  }, [results]);

  const handleSetRowsPerPage = (limit: number) => {
    setSearchParamsFromKeys({ [TRACES_URL_PARAMS.ROWS_PER_PAGE]: limit || "all" });
  };

  return (
    <div
      className="vm-traces-results-table"
      ref={containerRef}
    >
      <div className="vm-traces-results-table-toolbar">
        <SelectLimit
          allowUnlimited
          limit={rowsPerPage}
          onChange={handleSetRowsPerPage}
        />
        <TableSettings
          columnKeys={ALL_TRACES_COLUMN_KEYS}
          viewColumnKeys={viewColumnKeys}
          statsByKey={NO_COLUMN_STATS}
          dispatchViewColumns={dispatchViewColumns}
        />
      </div>
      <Table
        tableId={tableId}
        rows={results}
        columns={tableColumns}
        defaultOrder={{ key: "startTime", dir: "desc" }}
        isActiveRow={row => row.traceID === activeTraceID}
        rowClassName={row => row.errorCount > 0 ? "vm-table-row_error" : undefined}
        onClickRow={onClickRow}
        paginationOffset={offset}
        applyViewColumns={dispatchViewColumns}
        actionsRender={row => (
          <Tooltip title="Open trace view">
            <Button
              variant="text"
              size="small"
              color="gray"
              startIcon={<TraceIcon/>}
              aria-label="Open trace view"
              onClick={e => {
                e.stopPropagation();
                onOpenTrace(row);
              }}
            />
          </Tooltip>
        )}
      />
      <Pagination
        currentPage={page}
        totalItems={results.length}
        itemsPerPage={rowsPerPage}
        onPageChange={onChangePage}
      />
    </div>
  );
};

export default TracesResultsTable;
