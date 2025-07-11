import React, { FC, useCallback, useEffect, useMemo, useState } from "preact/compat";
import "./style.scss";
import { Traces } from "../../../api/types";
import Accordion from "../../../components/Main/Accordion/Accordion";
import { groupByMultipleKeys } from "../../../utils/array";
import Tooltip from "../../../components/Main/Tooltip/Tooltip";
import GroupTracesItem from "./GroupTracesItem";
import Button from "../../../components/Main/Button/Button";
import { CollapseIcon, ExpandIcon } from "../../../components/Main/Icons";
import { useSearchParams } from "react-router-dom";
import { getStreamPairs } from "../../../utils/traces";
import GroupTracesConfigurators
  from "../../../components/TracesConfigurators/GroupTracesConfigurators/GroupTracesConfigurators";
import GroupTracesHeader from "./GroupTracesHeader";
import { TRACES_DISPLAY_FIELDS, TRACES_GROUP_BY, TRACES_URL_PARAMS } from "../../../constants/traces";
import Pagination from "../../../components/Main/Pagination/Pagination";
import SelectLimit from "../../../components/Main/Pagination/SelectLimit/SelectLimit";
import { usePaginateGroups } from "../hooks/usePaginateGroups";
import { GroupTracesType } from "../../../types";
import useDeviceDetect from "../../../hooks/useDeviceDetect";
import DownloadTracesButton from "../DownloadTracesButton/DownloadTracesButton";
import { hasSortPipe } from "../../../components/Configurators/QueryEditor/TracesQL/utils/sort";

interface Props {
  traces: Traces[];
  settingsRef: React.RefObject<HTMLElement>;
}

const GroupTraces: FC<Props> = ({ traces, settingsRef }) => {
  const { isMobile } = useDeviceDetect();
  const [searchParams, setSearchParams] = useSearchParams();

  const query = searchParams.get("query") || "";
  const queryHasSort = hasSortPipe(query);

  const [page, setPage] = useState(1);
  const [expandGroups, setExpandGroups] = useState<boolean[]>([]);

  const groupBy = searchParams.get(TRACES_URL_PARAMS.GROUP_BY) || TRACES_GROUP_BY;
  const displayFieldsString = searchParams.get(TRACES_URL_PARAMS.DISPLAY_FIELDS) || TRACES_DISPLAY_FIELDS;
  const displayFields = useMemo(() => displayFieldsString.split(","), [displayFieldsString]);

  const rowsPerPageRaw = Number(searchParams.get(TRACES_URL_PARAMS.ROWS_PER_PAGE));
  const rowsPerPage = isNaN(rowsPerPageRaw) ? 0 : rowsPerPageRaw;

  const expandAll = useMemo(() => expandGroups.every(Boolean), [expandGroups]);

  const groupData: GroupTracesType[] = useMemo(() => {
    return groupByMultipleKeys(traces, [groupBy]).map((item) => {
      const streamValue = item.values[0]?.[groupBy] || "";
      const pairs = getStreamPairs(streamValue);

      // VictoriaTraces sends rows oldest → newest when the query has no `| sort` pipe,
      // so we reverse the array to put the newest entries first.
      // If a sort is already specified, keep the original order.
      const values = queryHasSort ? item.values : item.values.toReversed();

      return {
        keys: item.keys,
        keysString: item.keys.join(""),
        values,
        pairs,
        total: values.length,
      };
    }).sort((a, b) => b.total - a.total); // groups sorting
  }, [traces, groupBy, queryHasSort]);

  const paginatedGroups = usePaginateGroups(groupData, page, rowsPerPage);

  const handleToggleExpandAll = useCallback(() => {
    setExpandGroups(new Array(groupData.length).fill(!expandAll));
  }, [expandAll, groupData.length]);

  const handleChangeExpand = useCallback((i: number) => (value: boolean) => {
    setExpandGroups((prev) => {
      const newExpandGroups = [...prev];
      newExpandGroups[i] = value;
      return newExpandGroups;
    });
  }, []);

  const handleSetRowsPerPage = (limit?: number) => {
    if (limit) {
      searchParams.set(TRACES_URL_PARAMS.ROWS_PER_PAGE, String(limit));
    } else {
      searchParams.delete(TRACES_URL_PARAMS.ROWS_PER_PAGE);
    }

    setSearchParams(searchParams);
  };

  const handlePageChange = (newPage: number) => {
    setPage(newPage);
    window.scrollTo({ top: 0 });
  };

  const getTraces = useCallback(() => traces, [traces]);

  useEffect(() => {
    setExpandGroups(new Array(groupData.length).fill(!isMobile));
  }, [groupData]);

  useEffect(() => {
    setPage(1);
  }, [rowsPerPage]);

  return (
    <>
      <div className="vm-group-traces">
        {paginatedGroups.map((group, groupN) => (
          <div
            className="vm-group-traces-section"
            key={group.keysString}
          >
            <Accordion
              defaultExpanded={expandGroups[groupN]}
              onChange={handleChangeExpand(groupN)}
              title={(
                <GroupTracesHeader
                  group={group}
                  index={groupN}
                />
              )}
            >
              <div className="vm-group-traces-section-rows">
                {group.values.map((log, rowN) => (
                  <GroupTracesItem
                    key={`${groupN}_${rowN}_${log._time}`}
                    log={log}
                    displayFields={displayFields}
                  />
                ))}
              </div>
            </Accordion>
          </div>
        ))}

        <Pagination
          currentPage={page}
          totalItems={traces.length}
          itemsPerPage={rowsPerPage || Infinity}
          onPageChange={handlePageChange}
        />
      </div>


      {settingsRef.current && React.createPortal((
        <div className="vm-group-traces-header">
          <div className="vm-explore-traces-body-header__log-info">
            Total groups: <b>{groupData.length}</b>
          </div>
          <SelectLimit
            allowUnlimited
            limit={rowsPerPage}
            onChange={handleSetRowsPerPage}
          />
          <Tooltip title={expandAll ? "Collapse All" : "Expand All"}>
            <Button
              variant="text"
              startIcon={expandAll ? <CollapseIcon/> : <ExpandIcon/>}
              onClick={handleToggleExpandAll}
              ariaLabel={expandAll ? "Collapse All" : "Expand All"}
            />
          </Tooltip>
          <DownloadTracesButton getTraces={getTraces}/>
          <GroupTracesConfigurators traces={traces}/>
        </div>
      ), settingsRef.current)}
    </>
  );
};

export default GroupTraces;
