import { createContext, FC, useContext, useMemo, useReducer, Dispatch } from "preact/compat";
import { Action, AppState, initialState, reducer } from "./reducer";
import { getQueryStringValue } from "../../utils/query-string";

type StateContextType = { state: AppState, dispatch: Dispatch<Action> };

export const StateContext = createContext<StateContextType>({} as StateContextType);

// eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
export const useAppState = (): AppState => useContext(StateContext).state;
// eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
export const useAppDispatch = (): Dispatch<Action> => useContext(StateContext).dispatch;

export const initialPrepopulatedState = Object.entries(initialState)
  .reduce((acc, [key, value]) => ({
    ...acc,
    [key]: getQueryStringValue(key) || value
  }), {}) as AppState;

export const AppStateProvider: FC = ({ children }) => {
  const [state, dispatch] = useReducer(reducer, initialPrepopulatedState);

  const contextValue = useMemo(() => {
    return { state, dispatch };
  }, [state, dispatch]);

  return <StateContext value={contextValue}>
    {children}
  </StateContext>;
};


