import React, { FC, memo, useMemo } from "preact/compat";
import { Traces } from "../../../api/types";
import "./style.scss";
import useBoolean from "../../../hooks/useBoolean";
import { ArrowDownIcon, CopyIcon } from "../../../components/Main/Icons";
import classNames from "classnames";
import { useTracesState } from "../../../state/tracesPanel/TracesStateContext";
import dayjs from "dayjs";
import { useTimeState } from "../../../state/time/TimeStateContext";
import { marked } from "marked";
import { useSearchParams } from "react-router-dom";
import { TRACES_DATE_FORMAT, TRACES_URL_PARAMS } from "../../../constants/traces";
import { parseAnsiToHtml } from "../../../utils/ansiParser";
import GroupTracesFields from "./GroupTracesFields";
import { useLocalStorageBoolean } from "../../../hooks/useLocalStorageBoolean";
import Button from "../../../components/Main/Button/Button";
import Tooltip from "../../../components/Main/Tooltip/Tooltip";
import { useCallback, useEffect, useState } from "react";
import useCopyToClipboard from "../../../hooks/useCopyToClipboard";

interface Props {
  trace: Traces;
  displayFields?: string[];
  hideGroupButton?: boolean;
  onItemClick?: (trace: Traces) => void;
}

const GroupTracesItem: FC<Props> = ({ trace, displayFields = ["_msg"], onItemClick, hideGroupButton }) => {
  const {
    value: isOpenFields,
    toggle: toggleOpenFields,
  } = useBoolean(false);
  const [copied, setCopied] = useState<boolean>(false);
  const copyToClipboard = useCopyToClipboard();

  const [searchParams] = useSearchParams();
  const { markdownParsing, ansiParsing } = useTracesState();
  const { timezone } = useTimeState();

  const noWrapLines = searchParams.get(TRACES_URL_PARAMS.NO_WRAP_LINES) === "true";
  const dateFormat = searchParams.get(TRACES_URL_PARAMS.DATE_FORMAT) || TRACES_DATE_FORMAT;

  const formattedTime = useMemo(() => {
    if (!trace._time) return "";
    return dayjs(trace._time).tz().format(dateFormat);
  }, [trace._time, timezone, dateFormat]);

  const formattedMarkdown = useMemo(() => {
    if (!markdownParsing || !trace._msg) return "";
    return marked(trace._msg.replace(/```/g, "\n```\n")) as string;
  }, [trace._msg, markdownParsing]);

  const hasFields = Object.keys(trace).length > 0;

  const displayMessage = useMemo(() => {
    const values: (string | React.ReactNode)[] = [];

    if (!hasFields) {
      values.push("-");
    }

    if (displayFields.some(field => trace[field])) {
      displayFields.filter(field => trace[field]).forEach((field) => {
        const value = field === "_msg" && ansiParsing ? parseAnsiToHtml(trace[field]) : trace[field];
        values.push(value);
      });
    } else {
      Object.entries(trace).forEach(([key, value]) => {
        values.push(`${key}: ${value}`);
      });
    }

    return values;
  }, [trace, hasFields, displayFields, ansiParsing]);

  const [disabledHovers] = useLocalStorageBoolean("TRACES_DISABLED_HOVERS");

  const handleClick = () => {
    toggleOpenFields();
    onItemClick?.(trace);
  };

  const handleCopy = useCallback(async (e: Event) => {
    e.stopPropagation();
    if (copied) return;
    try {
      await copyToClipboard(JSON.stringify(trace, null, 2));
      setCopied(true);
    } catch (e) {
      console.error(e);
    }
  }, [copied, copyToClipboard]);

  useEffect(() => {
    if (copied === null) return;
    const timeout = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timeout);
  }, [copied]);

  return (
    <div className="vm-group-traces-row">
      <div
        className={classNames({
          "vm-group-traces-row-content": true,
          "vm-group-traces-row-content_interactive": !disabledHovers,
        })}
        onClick={handleClick}
      >
        <Tooltip title={copied ? "Copied" : "Copy to clipboard"}>
          <Button
            className="vm-group-traces-row-content__copy-row"
            variant="text"
            color="gray"
            size="small"
            startIcon={<CopyIcon/>}
            onClick={handleCopy}
            ariaLabel="copy to clipboard"
          />
        </Tooltip>
        {hasFields && (
          <div
            className={classNames({
              "vm-group-traces-row-content__arrow": true,
              "vm-group-traces-row-content__arrow_open": isOpenFields,
            })}
          >
            <ArrowDownIcon/>
          </div>
        )}
        <div
          className={classNames({
            "vm-group-traces-row-content__time": true,
            "vm-group-traces-row-content__time_missing": !formattedTime
          })}
        >
          {formattedTime || "timestamp missing"}
        </div>
        <div
          className={classNames({
            "vm-group-traces-row-content__msg": true,
            "vm-group-traces-row-content__msg_empty-msg": !trace._msg,
            "vm-group-traces-row-content__msg_missing": !displayMessage,
            "vm-group-traces-row-content__msg_single-line": noWrapLines,
          })}
          dangerouslySetInnerHTML={formattedMarkdown ? { __html: formattedMarkdown } : undefined}
        >
          {displayMessage.map((msg, i) => (
            <span
              className="vm-group-traces-row-content__sub-msg"
              key={`${msg}_${i}`}
            >
              {msg}
            </span>
          ))}
        </div>
      </div>
      {hasFields && isOpenFields && <GroupTracesFields
        hideGroupButton={hideGroupButton}
        trace={trace}
      />}
    </div>
  );
};

export default memo(GroupTracesItem);
