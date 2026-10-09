import { LogHits } from "../../../api/types";

export const OTHER_HITS_LABEL = "other";

export const calculateTotalHits = (hits: LogHits[]): number => {
  return hits.reduce((acc, item) => acc + (item.total || 0), 0);
};

// Sorts by the given numeric key descending, keeping the "other" series last.
export const sortLogHits = <T extends { label?: string }>(key: keyof T) => (a: T, b: T): number => {
  if (a.label === OTHER_HITS_LABEL) return 1;
  if (b.label === OTHER_HITS_LABEL) return -1;

  const aValue = a[key] as unknown as number;
  const bValue = b[key] as unknown as number;

  return bValue - aValue;
};
