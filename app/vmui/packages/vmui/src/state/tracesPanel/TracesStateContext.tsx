import React, { createContext, FC, useContext, useMemo, useReducer } from "preact/compat";
import { TracesAction, TracesState, initialTracesState, reducer } from "./reducer";
import { Dispatch } from "react";

type TracesStateContextType = { state: TracesState, dispatch: Dispatch<TracesAction> };

export const TracesStateContext = createContext<TracesStateContextType>({} as TracesStateContextType);

export const useTracesState = (): TracesState => useContext(TracesStateContext).state;
export const useTracesDispatch = (): Dispatch<TracesAction> => useContext(TracesStateContext).dispatch;

export const TracesStateProvider: FC = ({ children }) => {
  const [state, dispatch] = useReducer(reducer, initialTracesState);

  const contextValue = useMemo(() => {
    return { state, dispatch };
  }, [state, dispatch]);

  return <TracesStateContext.Provider value={contextValue}>
    {children}
  </TracesStateContext.Provider>;
};


