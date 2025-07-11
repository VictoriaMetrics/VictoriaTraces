import React, { FC, useMemo, useRef, useState } from "preact/compat";
import "./style.scss";
import Table from "../../../components/Table/Table";
import { Traces } from "../../../api/types";
import Pagination from "../../../components/Main/Pagination/Pagination";
import { useEffect } from "react";

interface TableTracesProps {
  traces: Traces[];
  displayColumns: string[];
  tableCompact: boolean;
  columns: string[];
  rowsPerPage: number;
}

const getColumnClass = (key: string) => {
  switch (key) {
    case "_time":
      return "vm-table-cell_traces-time";
    default:
      return "vm-table-cell_traces";
  }
};

const compactColumns = [{
  key: "_vmui_data",
  title: "Data",
  className: "vm-table-cell_traces vm-table-cell_pre"
}];

const TableTraces: FC<TableTracesProps> = ({ traces, displayColumns, tableCompact, columns, rowsPerPage }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [page, setPage] = useState(1);

  const rows = useMemo(() => {
    return traces.map((trace) => {
      const _vmui_data = JSON.stringify(trace, null, 2);
      return { ...trace, _vmui_data };
    }) as Traces[];
  }, [traces]);

  const tableColumns = useMemo(() => {
    return columns.map((key) => ({
      key: key as keyof Traces,
      title: key,
      className: getColumnClass(key),
    }));
  }, [columns]);


  const filteredColumns = useMemo(() => {
    if (tableCompact) return compactColumns;
    if (!displayColumns?.length) return [];
    return tableColumns.filter(c => displayColumns.includes(c.key as string));
  }, [tableColumns, displayColumns, tableCompact]);

  const paginationOffset = useMemo(() => {
    const startIndex = (page - 1) * rowsPerPage;
    const endIndex = startIndex + rowsPerPage;
    return { startIndex, endIndex };
  }, [page, rowsPerPage]);

  const handlePageChange = (newPage: number) => {
    setPage(newPage);
    if (containerRef.current) {
      const y = containerRef.current.getBoundingClientRect().top + window.scrollY - 50;
      window.scrollTo({ top: y });
    }
  };

  useEffect(() => {
    setPage(1);
  }, [traces, rowsPerPage]);

  return (
    <>
      <div ref={containerRef}>
        <Table
          rows={rows}
          columns={filteredColumns}
          defaultOrderBy={"_time"}
          defaultOrderDir={"desc"}
          copyToClipboard={"_vmui_data"}
          paginationOffset={paginationOffset}
        />
      </div>
      <Pagination
        currentPage={page}
        totalItems={rows.length}
        itemsPerPage={rowsPerPage}
        onPageChange={handlePageChange}
      />
    </>
  );
};

export default TableTraces;
