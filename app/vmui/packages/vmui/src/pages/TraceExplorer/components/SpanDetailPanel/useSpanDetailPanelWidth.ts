import { useState } from "preact/compat";

// In-memory only (no persistence).
export function useSpanDetailPanelWidth() {
  const [width, setWidth] = useState(0);
  return { width, setWidth };
}
