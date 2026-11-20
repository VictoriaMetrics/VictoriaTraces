import { LogHits } from "../../../api/types";

export const OTHER_HITS_LABEL = "other";

export const calculateTotalHits = (hits: LogHits[]): number => {
  return hits.reduce((acc, item) => acc + (item.total || 0), 0);
};

// Returns the next set of hidden series labels after a legend click on `target`:
// a plain click isolates the series (or shows everything back when it is already the only visible one),
// a multi-select click toggles just that series.
export const toggleHiddenLabels = (
  labels: string[],
  hidden: ReadonlySet<string>,
  target: string,
  isMultiSelect: boolean
): Set<string> => {
  if (isMultiSelect) {
    const next = new Set(hidden);
    if (!next.delete(target)) next.add(target);
    return next;
  }

  const isOnlyTargetVisible = labels.every(label => label === target || hidden.has(label));
  return isOnlyTargetVisible ? new Set() : new Set(labels.filter(label => label !== target));
};

// Sorts by the given numeric key descending, keeping the "other" series last.
export const sortLogHits = <T extends { label?: string }>(key: keyof T) => (a: T, b: T): number => {
  if (a.label === OTHER_HITS_LABEL) return 1;
  if (b.label === OTHER_HITS_LABEL) return -1;

  const aValue = a[key] as unknown as number;
  const bValue = b[key] as unknown as number;

  return bValue - aValue;
};
