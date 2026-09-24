import { createContext, FC, useContext, useMemo, useReducer, Dispatch } from "preact/compat";
import { QueryAction, QueryState, initialQueryState, reducer } from "./reducer";

type QueryStateContextType = { state: QueryState, dispatch: Dispatch<QueryAction> };

export const QueryStateContext = createContext<QueryStateContextType>({} as QueryStateContextType);

// eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
export const useQueryState = (): QueryState => useContext(QueryStateContext).state;
// eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
export const useQueryDispatch = (): Dispatch<QueryAction> => useContext(QueryStateContext).dispatch;

export const QueryStateProvider: FC = ({ children }) => {
  const [state, dispatch] = useReducer(reducer, initialQueryState);

  const contextValue = useMemo(() => {
    return { state, dispatch };
  }, [state, dispatch]);

  return <QueryStateContext value={contextValue}>
    {children}
  </QueryStateContext>;
};


