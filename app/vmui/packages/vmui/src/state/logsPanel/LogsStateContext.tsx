import { createContext, FC, useContext, useMemo, useReducer, Dispatch } from "react";
import { LogsAction, LogsState, initialLogsState, reducer } from "./reducer";

type LogsStateContextType = { state: LogsState, dispatch: Dispatch<LogsAction> };

export const LogsStateContext = createContext<LogsStateContextType>({} as LogsStateContextType);

// eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
export const useLogsState = (): LogsState => useContext(LogsStateContext).state;
// eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
export const useLogsDispatch = (): Dispatch<LogsAction> => useContext(LogsStateContext).dispatch;

export const LogsStateProvider: FC = ({ children }) => {
  const [state, dispatch] = useReducer(reducer, initialLogsState);

  const contextValue = useMemo(() => {
    return { state, dispatch };
  }, [state, dispatch]);

  return <LogsStateContext value={contextValue}>
    {children}
  </LogsStateContext>;
};


