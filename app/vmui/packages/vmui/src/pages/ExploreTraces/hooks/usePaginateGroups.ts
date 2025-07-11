import { useMemo } from "react";
import { GroupTracesType } from "../../../types";

export const usePaginateGroups = (
  groupData: GroupTracesType[],
  page: number,
  rowsPerPage: number
):  GroupTracesType[] => {
  return useMemo(() => {
    if (!rowsPerPage) return groupData;

    const startIdx = (page - 1) * rowsPerPage;
    const endIdx = startIdx + rowsPerPage;
    let currentIdx = 0;
    const result: GroupTracesType[] = [];

    for (const group of groupData) {
      const groupLength = group.values.length;
      const groupStart = currentIdx;
      const groupEnd = currentIdx + groupLength;

      // Skip groups that are entirely before the current page range.
      if (groupEnd <= startIdx) {
        currentIdx = groupEnd;
        continue;
      }

      // Break if the group starts after the current page range.
      if (groupStart >= endIdx) {
        break;
      }

      // Determine the portion of the group that falls within the page range.
      const sliceStart = Math.max(0, startIdx - groupStart);
      const sliceEnd = Math.min(groupLength, endIdx - groupStart);

      // Create a partial group with the sliced traces.
      const partialGroup = {
        ...group,
        values: group.values.slice(sliceStart, sliceEnd),
      };

      result.push(partialGroup);
      currentIdx = groupEnd;

      // Exit loop if we have reached or exceeded the end index.
      if (currentIdx >= endIdx) break;
    }

    return result;
  }, [groupData, page, rowsPerPage]);
};
