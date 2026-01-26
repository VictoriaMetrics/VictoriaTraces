import * as PreactCompat from "preact/compat";

// preact/compat's type declarations use `export =`, which TypeScript forbids
// re-exporting via `export * from`. Named runtime values are re-exported
// individually instead; type-only symbols (ReactNode, FC, etc.) don't need a
// runtime binding here since "react"'s *types* resolve straight to
// preact/compat via tsconfig "paths" and get elided from the compiled JS.
export const {
  Children,
  Component,
  Fragment,
  PureComponent,
  StrictMode,
  Suspense,
  SuspenseList,
  cloneElement,
  createContext,
  createElement,
  createRef,
  forwardRef,
  isValidElement,
  lazy,
  memo,
  startTransition,
  unstable_batchedUpdates,
  useCallback,
  useContext,
  useDebugValue,
  useDeferredValue,
  useEffect,
  useId,
  useImperativeHandle,
  useInsertionEffect,
  useLayoutEffect,
  useMemo,
  useReducer,
  useRef,
  useState,
  useSyncExternalStore,
  useTransition,
  version,
} = PreactCompat;

export default PreactCompat;

/**
 * react-router v8 imports useOptimistic from "react" at module scope even in
 * declarative (non-Data-Router) mode, but preact/compat has no shim for it.
 * This mirrors React's real semantics closely enough for code paths that
 * don't actually rely on it.
 */
export function useOptimistic<State, Action = State>(
  state: State,
  updateFn?: (state: State, action: Action) => State
): [State, (action: Action) => void] {
  const [optimisticState, setOptimisticState] = useState(state);

  useEffect(() => {
    setOptimisticState(state);
  }, [state]);

  const addOptimistic = useCallback((action: Action) => {
    setOptimisticState(prev => (updateFn ? updateFn(prev, action) : (action as unknown as State)));
  }, [updateFn]);

  return [optimisticState, addOptimistic];
}

type PromiseRecord<T> =
  | { status: "pending" }
  | { status: "fulfilled"; value: T }
  | { status: "rejected"; error: unknown };

const promiseRecords = new WeakMap<Promise<unknown>, PromiseRecord<unknown>>();

/**
 * react-router v8's RSC support module imports `use` from "react" at module
 * scope; this app never triggers that (client-only, no RSC), but a real
 * implementation keeps the import from dangling.
 */
export function use<T>(resource: Promise<T> | PreactCompat.Context<T>): T {
  if (resource && typeof (resource as Promise<T>).then === "function") {
    const promise = resource as Promise<T>;
    let record = promiseRecords.get(promise) as PromiseRecord<T> | undefined;

    if (!record) {
      record = { status: "pending" };
      promiseRecords.set(promise, record);
      promise.then(
        value => { promiseRecords.set(promise, { status: "fulfilled", value }); },
        error => { promiseRecords.set(promise, { status: "rejected", error }); }
      );
    }

    if (record.status === "pending") throw promise;
    if (record.status === "rejected") throw record.error;
    return record.value;
  }

  // `use`, unlike other hooks, is explicitly allowed to be called conditionally.
  // eslint-disable-next-line @eslint-react/rules-of-hooks
  return useContext(resource as PreactCompat.Context<T>);
}
