import { LogHits } from "../../../api/types";

export const OTHER_HITS_LABEL = "other";

export const calculateTotalHits = (hits: LogHits[]): number => {
  return hits.reduce((acc, item) => acc + (item.total || 0), 0);
};

// Stable series identity: labels can collide, e.g. a real "other" service and the aggregate.
export const getHitKey = (hit: Pick<LogHits, "fields">): string => JSON.stringify(hit.fields || {});

// Legend click: plain isolates the series or restores all, multi-select toggles just it.
export const toggleHiddenSeries = (
  keys: string[],
  hidden: ReadonlySet<string>,
  target: string,
  isMultiSelect: boolean
): Set<string> => {
  if (isMultiSelect) {
    const next = new Set(hidden);
    if (!next.delete(target)) next.add(target);
    return next;
  }

  const isOnlyTargetVisible = keys.every(key => key === target || hidden.has(key));
  return isOnlyTargetVisible ? new Set() : new Set(keys.filter(key => key !== target));
};

// Drops hidden keys of gone series; returns the same set when nothing changes.
export const pruneHiddenSeries = (hidden: Set<string>, keys: string[]): Set<string> => {
  const present = new Set(keys);
  const next = new Set([...hidden].filter(key => present.has(key)));
  return next.size === hidden.size ? hidden : next;
};

// Sorts by the given numeric key descending, keeping the "other" series last.
export const sortLogHits = <T extends { isOther?: boolean }>(key: keyof T) => (a: T, b: T): number => {
  if (a.isOther !== b.isOther) return a.isOther ? 1 : -1;

  const aValue = a[key] as unknown as number;
  const bValue = b[key] as unknown as number;

  return bValue - aValue;
};
