import * as React from "react";
import cx from "classnames";

import "./style.scss";

type TimelineRowProps = {
  children: React.ReactNode;
  className?: string;
} & Omit<React.HTMLAttributes<HTMLDivElement>, "className">;

export default function TimelineRow({ children, className, ...rest }: TimelineRowProps) {
  return (
    <div
      className={cx("vm-timeline-row", className)}
      {...rest}
    >
      {children}
    </div>
  );
}

type TimelineRowCellProps = {
  children: React.ReactNode;
  className?: string;
  // A pixel width fixes the cell's size; "auto" makes it flex to fill the remaining row width.
  width: number | "auto";
  style?: object;
  onClick?: (event: React.MouseEvent<HTMLDivElement>) => void;
};

// eslint-disable-next-line @eslint-react/no-forward-ref -- preact/compat still requires forwardRef; a plain function component does not receive 'ref' as a prop here
const TimelineRowCell = React.forwardRef<HTMLDivElement, TimelineRowCellProps>(
  ({ children, className, width, style = {}, onClick = () => {} }, ref) => {
    const mergedStyle = width === "auto"
      ? { ...style, position: "relative", flex: "1 1 auto", minWidth: 0 }
      : { ...style, position: "relative", flex: "none", width: `${width}px` };
    return (
      <div
        ref={ref}
        className={className}
        style={mergedStyle}
        onClick={onClick}
      >
        {children}
      </div>
    );
  }
);

TimelineRowCell.displayName = "TimelineRow.Cell";

TimelineRow.Cell = TimelineRowCell;
