import { FC, RefObject } from "preact/compat";
import classNames from "classnames";
import "./style.scss";
import { Size } from "../../../hooks/useResizeObserver";
import { useDragResize } from "../../../hooks/useDragResize";

type Props = {
  targetRef: RefObject<HTMLElement>; // element target which size will be changed
  minSize: number;
  dir?: 1 | -1;
  size?: Size;
  onResizeEnd: (width: number) => void;
};

const DragResizeHandle: FC<Props> = ({
  targetRef,
  minSize,
  dir = 1,
  size = {},
  onResizeEnd,
}) => {
  const { isResizing, dragOffset, onPointerDown, onPointerMove, onPointerUp } = useDragResize({
    targetRef,
    minSize,
    dir,
    axis: "x",
    onResizeEnd,
  });

  const preventNativeDrag = (e: DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  };

  // The browser still synthesizes a click from the pointerdown/pointerup pair even though
  // onPointerDown prevents default — left unstopped, it bubbles up to the header cell's
  // onClick and triggers a sort toggle right after a resize.
  const stopClick = (e: MouseEvent) => {
    e.stopPropagation();
  };

  const style: Record<string, string | undefined> = {
    height: size.height ? `${size.height}px` : undefined,
    transform: isResizing ? `translateX(${dir * dragOffset}px)` : undefined,
  };

  return (
    <div
      style={style}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={onPointerUp}
      onDragStart={preventNativeDrag}
      onClick={stopClick}
      className={classNames({
        "vm-drag-resize-handle": true,
        "vm-drag-resize-handle_revert": dir === -1,
        "vm-drag-resize-handle_resizing": isResizing,
      })}
    />
  );
};

export default DragResizeHandle;
