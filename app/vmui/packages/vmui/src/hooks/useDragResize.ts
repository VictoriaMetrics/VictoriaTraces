import { useRef, useState } from "preact/hooks";
import { RefObject } from "preact/compat";
import { borderBoxToContentSize } from "../utils/dom-geometry";

type Axis = "x" | "y";
type Direction = 1 | -1;

type Options = {
  targetRef: RefObject<HTMLElement>;
  minSize?: number;
  axis?: Axis;
  dir?: Direction;
  onResizeEnd: (sizePx: number) => void;
};

export function useDragResize({
  targetRef,
  minSize = 80,
  axis = "x",
  dir = 1,
  onResizeEnd,
}: Options) {
  const [isResizing, setIsResizing] = useState(false);
  const [dragOffset, setDragOffset] = useState(0);

  const startPosRef = useRef(0);
  const startSizeRef = useRef(0);
  const dragOffsetRef = useRef(0);
  const isResizingRef = useRef(false);
  const justFinishedRef = useRef(false);
  const captureElRef = useRef<HTMLElement | null>(null);
  const pointerIdRef = useRef<number | null>(null);

  const getClientPos = (e: PointerEvent) => (axis === "x" ? e.clientX : e.clientY);

  const getCurrentSize = (el: HTMLElement) => {
    const rect = el.getBoundingClientRect();
    return axis === "x" ? rect.width : rect.height;
  };

  // Some macOS trackpad drag gestures deliver the terminating release to whatever's
  // visually under the cursor instead of the pointer-captured handle - this both misfires
  // that element's own click (e.g. toggling sort) and, since finishResize below never runs,
  // leaves the drag stuck. Swallow any click for as long as a resize is in flight, plus the
  // one trailing synthesized click the browser dispatches right after release.
  const suppressStrayClick = (e: MouseEvent) => {
    if (!isResizingRef.current && !justFinishedRef.current) return;
    e.preventDefault();
    e.stopPropagation();
  };

  const finishResize = () => {
    if (!isResizingRef.current) return;

    const target = targetRef.current;
    const el = captureElRef.current;
    const pointerId = pointerIdRef.current;
    if (el && pointerId != null && el.hasPointerCapture?.(pointerId)) {
      el.releasePointerCapture(pointerId);
    }

    if (target) {
      const finalBorderBox = Math.max(minSize, startSizeRef.current + dragOffsetRef.current);
      const finalSize = borderBoxToContentSize(target, finalBorderBox, axis);
      onResizeEnd(finalSize);
    }

    isResizingRef.current = false;
    setIsResizing(false);
    setDragOffset(0);
    dragOffsetRef.current = 0;

    window.removeEventListener("pointerup", finishResize);
    window.removeEventListener("pointercancel", finishResize);

    justFinishedRef.current = true;
    setTimeout(() => {
      justFinishedRef.current = false;
    }, 0);
  };

  // Pointer capture routes all subsequent pointermove/pointerup events to this same
  // element regardless of what's under the cursor, so dragging survives the cursor
  // leaving the (thin) handle and doesn't need a full-screen overlay as a safety net.
  const onPointerDown = (pointerDownEvent: PointerEvent) => {
    const target = targetRef.current;
    if (!target) return;

    pointerDownEvent.preventDefault();
    pointerDownEvent.stopPropagation();

    const handleEl = pointerDownEvent.currentTarget as HTMLElement;
    handleEl?.setPointerCapture(pointerDownEvent.pointerId);
    captureElRef.current = handleEl;
    pointerIdRef.current = pointerDownEvent.pointerId;

    isResizingRef.current = true;
    setIsResizing(true);
    setDragOffset(0);
    dragOffsetRef.current = 0;

    startPosRef.current = getClientPos(pointerDownEvent);
    startSizeRef.current = getCurrentSize(target);

    // Fallback in case the terminating pointerup/pointercancel doesn't reach the captured
    // element (see suppressStrayClick above) - without this the drag would never conclude.
    window.addEventListener("pointerup", finishResize);
    window.addEventListener("pointercancel", finishResize);
    window.addEventListener("click", suppressStrayClick, { capture: true, once: true });
  };

  const onPointerMove = (pointerMoveEvent: PointerEvent) => {
    if (!isResizingRef.current) return;

    const rawOffset = (getClientPos(pointerMoveEvent) - startPosRef.current) * dir;
    const minOffset = minSize - startSizeRef.current;
    const nextOffset = Math.max(rawOffset, minOffset);

    dragOffsetRef.current = nextOffset;
    setDragOffset(nextOffset);
  };

  return {
    isResizing,
    dragOffset,
    onPointerDown,
    onPointerMove,
    onPointerUp: finishResize,
  };
}
