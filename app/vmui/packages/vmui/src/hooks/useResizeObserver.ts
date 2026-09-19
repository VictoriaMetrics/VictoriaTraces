import { useEffect, useRef, useState } from "react";

import type { RefObject } from "react";

import { useIsMounted } from "./useIsMounted";

export type Size = {
  width: number | undefined
  height: number | undefined
}

type UseResizeObserverOptions<T extends HTMLElement = HTMLElement> = {
  ref: RefObject<T>
  onResize?: (size: Size) => void
  box?: "border-box" | "content-box" | "device-pixel-content-box"
}

const initialSize: Size = {
  width: undefined,
  height: undefined,
};

type ObserverEntryCallback = (entry: ResizeObserverEntry) => void;

// A `new ResizeObserver()` per call site means the browser runs a separate measure/notify
// pass per observer whenever layout changes — a window resize touches many observed
// elements across the page at once, so that's many passes instead of one. A single shared
// observer, dispatching each entry to whichever call site(s) registered that target,
// collapses that to one pass regardless of how many components use this hook.
let sharedObserver: ResizeObserver | null = null;
const callbacksByTarget = new Map<Element, Set<ObserverEntryCallback>>();

function getSharedObserver(): ResizeObserver | null {
  if (typeof window === "undefined" || !("ResizeObserver" in window)) return null;
  if (!sharedObserver) {
    sharedObserver = new ResizeObserver(entries => {
      entries.forEach(entry => {
        callbacksByTarget.get(entry.target)?.forEach(cb => cb(entry));
      });
    });
  }
  return sharedObserver;
}

export function useResizeObserver<T extends HTMLElement = HTMLElement>(
  options: UseResizeObserverOptions<T>,
): Size {
  const { ref, box = "content-box" } = options;
  const [size, setSize] = useState<Size>(initialSize);
  const isMounted = useIsMounted();
  const previousSizeRef = useRef<Size>({ ...initialSize });
  const onResizeRef = useRef<((size: Size) => void) | undefined>(undefined);
  onResizeRef.current = options.onResize;

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const observer = getSharedObserver();
    if (!observer) return;

    const handleEntry: ObserverEntryCallback = (entry) => {
      const boxProp = "borderBoxSize";

      const newWidth = extractSize(entry, boxProp, "inlineSize");
      const newHeight = extractSize(entry, boxProp, "blockSize");

      const hasChanged =
        previousSizeRef.current.width !== newWidth ||
        previousSizeRef.current.height !== newHeight;

      if (hasChanged) {
        const newSize: Size = { width: newWidth, height: newHeight };
        previousSizeRef.current.width = newWidth;
        previousSizeRef.current.height = newHeight;

        if (onResizeRef.current) {
          onResizeRef.current(newSize);
        } else {
          if (isMounted()) {
            setSize(newSize);
          }
        }
      }
    };

    let callbacks = callbacksByTarget.get(el);
    if (!callbacks) {
      callbacks = new Set();
      callbacksByTarget.set(el, callbacks);
    }
    callbacks.add(handleEntry);
    observer.observe(el, { box });

    return () => {
      callbacks.delete(handleEntry);
      if (callbacks.size === 0) {
        observer.unobserve(el);
        callbacksByTarget.delete(el);
      }
    };
  }, [box, ref, isMounted]);

  return size;
}

/** @private */
type BoxSizesKey = keyof Pick<
  ResizeObserverEntry,
  "borderBoxSize" | "contentBoxSize" | "devicePixelContentBoxSize"
>

function extractSize(
  entry: ResizeObserverEntry,
  box: BoxSizesKey,
  sizeType: keyof ResizeObserverSize,
): number | undefined {
  if (!entry[box]) {
    if (box === "contentBoxSize") {
      return entry.contentRect[sizeType === "inlineSize" ? "width" : "height"];
    }
    return undefined;
  }

  return Array.isArray(entry[box])
    ? entry[box][0][sizeType]
    : // @ts-expect-error Support Firefox's non-standard behavior
    (entry[box][sizeType] as number);
}
