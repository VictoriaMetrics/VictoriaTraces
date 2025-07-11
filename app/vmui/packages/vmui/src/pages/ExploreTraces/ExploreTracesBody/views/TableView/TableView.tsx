import React, { FC, useMemo, useState } from "preact/compat";
import DownloadTracesButton from "../../../DownloadTracesButton/DownloadTracesButton";
import { createPortal } from "preact/compat";
import "./style.scss";
import { ViewProps } from "../../types";
import useStateSearchParams from "../../../../../hooks/useStateSearchParams";
import TableTraces from "../../TableTraces";
import SelectLimit from "../../../../../components/Main/Pagination/SelectLimit/SelectLimit";
import TableSettings from "../../../../../components/Table/TableSettings/TableSettings";
import useSearchParamsFromObject from "../../../../../hooks/useSearchParamsFromObject";
import EmptyTraces from "../components/EmptyTraces/EmptyTraces";
import { useCallback } from "react";

const MemoizedTableView = React.memo(TableTraces);

const TableView: FC<ViewProps> = ({ data, settingsRef }) => {
  const { setSearchParamsFromKeys } = useSearchParamsFromObject();
  const [displayColumns, setDisplayColumns] = useState<string[]>([]);
  const [rowsPerPage, setRowsPerPage] = useStateSearchParams(100, "rows_per_page");

  const columns = useMemo(() => {
    const keys = new Set<string>();
    for (const item of data) {
      for (const key in item) {
        keys.add(key);
      }
    }
    return Array.from(keys).sort((a,b) => a.localeCompare(b));
  }, [data]);

  const handleSetRowsPerPage = (limit: number) => {
    setRowsPerPage(limit);
    setSearchParamsFromKeys({ rows_per_page: limit });
  };

  const getTraces = useCallback(() => data, [data]);

  const renderSettings = () => {
    if (!settingsRef.current) return null;

    return createPortal(
      <div className="vm-table-view__settings">
        <SelectLimit
          limit={rowsPerPage}
          onChange={handleSetRowsPerPage}
        />
        <div className="vm-table-view__settings-buttons">
          {data.length > 0 && <DownloadTracesButton getTraces={getTraces} />}
          <TableSettings
            columns={columns}
            selectedColumns={displayColumns}
            onChangeColumns={setDisplayColumns}
          />
        </div>
      </div>,
      settingsRef.current
    );
  };

  if (!data.length) return <EmptyTraces />;

  return (
    <>
      {renderSettings()}
      <MemoizedTableView
        traces={data}
        displayColumns={displayColumns}
        tableCompact={false}
        columns={columns}
        rowsPerPage={Number(rowsPerPage)}
      />
    </>
  );
};

export default TableView;
