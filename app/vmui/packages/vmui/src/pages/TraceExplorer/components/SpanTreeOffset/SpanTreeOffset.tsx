import React from "react";
import cx from "classnames";

import Tooltip from "../../../../components/Main/Tooltip";
import { OtelSpan } from "../../types";

import "./style.scss";

type TProps = {
  span: OtelSpan;
  color: string;
  isChildrenExpanded: boolean;
  onChildrenToggled: (e: React.MouseEvent<HTMLButtonElement>) => void;
};

const SpanTreeOffset: React.FC<TProps> = ({
  span,
  color,
  isChildrenExpanded,
  onChildrenToggled,
}) => {
  const { hasChildren, depth } = span;

  return (
    <span
      className={`vm-span-tree-offset ${hasChildren ? "is-parent" : ""}`}
    >
      {depth > 0 && (
        <div className="vm-span-tree-offset__indent-guide-wrapper">
          <span
            className="vm-span-tree-offset__indent-guide"
            style={{ marginLeft: `${(depth - 1) * 16}px` }}
          />
        </div>
      )}
      {hasChildren ? (
        <Tooltip title={isChildrenExpanded ? "Collapse children" : "Expand children"}>
          <button
            className="vm-span-tree-offset__icon-wrapper is-toggle"
            data-testid="icon-wrapper"
            aria-label={isChildrenExpanded ? "Collapse children" : "Expand children"}
            onClick={onChildrenToggled}
          >
            <span
              className={cx("vm-span-tree-offset__dot", { "is-collapsed": !isChildrenExpanded })}
              style={isChildrenExpanded ? { backgroundColor: color } : { borderColor: color }}
            />
          </button>
        </Tooltip>
      ) : (
        <span
          className="vm-span-tree-offset__icon-wrapper"
          data-testid="icon-wrapper"
        >
          <span
            className="vm-span-tree-offset__dot"
            style={{ backgroundColor: color }}
          />
        </span>
      )}
    </span>
  );
};

export default SpanTreeOffset;
