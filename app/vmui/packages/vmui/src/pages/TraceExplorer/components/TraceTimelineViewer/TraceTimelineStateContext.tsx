import { createContext, FC, useContext, useMemo, useReducer, Dispatch, ReactNode } from "preact/compat";
import { TraceTimelineAction, newInitialState, reducer } from "./reducer";
import { TTraceTimeline } from "../../types";

type TraceTimelineStateContextType = { state: TTraceTimeline, dispatch: Dispatch<TraceTimelineAction> };

export const TraceTimelineStateContext = createContext<TraceTimelineStateContextType>(
  {} as TraceTimelineStateContextType
);

// eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
export const useTraceTimelineState = (): TTraceTimeline => useContext(TraceTimelineStateContext).state;
export const useTraceTimelineDispatch = (): Dispatch<TraceTimelineAction> =>
  // eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
  useContext(TraceTimelineStateContext).dispatch;

export const TraceTimelineStateProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const [state, dispatch] = useReducer(reducer, undefined, newInitialState);

  const contextValue = useMemo(() => ({ state, dispatch }), [state, dispatch]);

  return (
    <TraceTimelineStateContext value={contextValue}>
      {children}
    </TraceTimelineStateContext>
  );
};
