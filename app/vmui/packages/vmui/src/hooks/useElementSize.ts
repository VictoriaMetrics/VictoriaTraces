import { useCallback, useEffect, useState } from "preact/compat";
import useEventListener from "./useEventListener";

export interface ElementSize {
  width: number
  height: number
}

const useElementSize = <T extends HTMLElement = HTMLDivElement>(): [(node: T | null) => void, ElementSize] => {
  // Mutable values like 'ref.current' aren't valid dependencies
  // because mutating them doesn't re-render the component.
  // Instead, we use a state as a ref to be reactive.
  const [ref, setRef] = useState<T | null>(null);
  const [size, setSize] = useState<ElementSize>({
    width: 0,
    height: 0,
  });

  // Prevent too many rendering using useCallback
  const handleSize = useCallback(() => {
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- offsetWidth/offsetHeight are only measurable once the node is committed to the DOM
    setSize({
      width: ref?.offsetWidth || 0,
      height: ref?.offsetHeight || 0,
    });

  }, [ref?.offsetHeight, ref?.offsetWidth]);

  useEventListener("resize", handleSize);

  useEffect(handleSize, [ref?.offsetHeight, ref?.offsetWidth, handleSize]);

  return [setRef, size];
};

export default useElementSize;
