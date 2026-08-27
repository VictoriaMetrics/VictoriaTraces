import { createContext, FC, useContext, useMemo, useReducer, Dispatch } from "preact/compat";
import { TimeAction, TimeState, initialTimeState, reducer } from "./reducer";

type TimeStateContextType = { state: TimeState, dispatch: Dispatch<TimeAction> };

export const TimeStateContext = createContext<TimeStateContextType>({} as TimeStateContextType);

// eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
export const useTimeState = (): TimeState => useContext(TimeStateContext).state;
// eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
export const useTimeDispatch = (): Dispatch<TimeAction> => useContext(TimeStateContext).dispatch;

export const TimeStateProvider: FC = ({ children }) => {
  const [state, dispatch] = useReducer(reducer, initialTimeState);

  const contextValue = useMemo(() => {
    return { state, dispatch };
  }, [state, dispatch]);

  return <TimeStateContext value={contextValue}>
    {children}
  </TimeStateContext>;
};


