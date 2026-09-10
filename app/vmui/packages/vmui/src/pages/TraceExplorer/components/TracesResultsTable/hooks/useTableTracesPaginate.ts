import { RefObject, useEffect, useState } from "preact/compat";

type Options = {
  rowsPerPage: number;
  containerRef: RefObject<HTMLElement>;
}

export const useTableTracesPaginate = ({ rowsPerPage, containerRef }: Options) => {
  const [page, setPage] = useState(1);

  const startOffset = rowsPerPage > 0 ? (page - 1) * rowsPerPage : 0;
  const endOffset = rowsPerPage > 0 ? page * rowsPerPage : Infinity;
  const offset: [number, number] = [startOffset, endOffset];

  const onChangePage = (newPage: number) => {
    setPage(newPage);
    if (containerRef.current) {
      const y = containerRef.current.getBoundingClientRect().top + window.scrollY - 50;
      if (y < window.scrollY) window.scrollTo({ top: y });
    }
  };

  useEffect(() => {
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- resets to page 1 whenever the page size changes, since the current page offset would otherwise be invalid
    setPage(1);
  }, [rowsPerPage]);

  return {
    page,
    offset,
    onChangePage
  };
};
