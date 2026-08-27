import { useRef, useState } from "preact/compat";
import useEventListener from "./useEventListener";
import { useResizeObserver } from "./useResizeObserver";

interface WindowSize {
  width: number
  height: number
}

const useWindowSize = (): WindowSize => {
  const [windowSize, setWindowSize] = useState<WindowSize>(() => ({
    width: window.innerWidth,
    height: window.innerHeight,
  }));

  const handleSize = () => {
    setWindowSize({
      width: window.innerWidth,
      height: window.innerHeight,
    });
  };

  useEventListener("resize", handleSize);

  const htmlRef = useRef(document.documentElement);
  useResizeObserver({ ref: htmlRef, onResize: handleSize });

  return windowSize;
};

export default useWindowSize;
