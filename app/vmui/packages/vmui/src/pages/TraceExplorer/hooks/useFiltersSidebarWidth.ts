import { useCallback, useState } from "preact/compat";

// Kept in-memory only (no persistence) to avoid touching the shared storage-key registry.
export function useFiltersSidebarWidth() {
  const [width, setWidth] = useState(0);
  const clearWidth = useCallback(() => setWidth(0), []);
  return { width, setWidth, clearWidth };
}
