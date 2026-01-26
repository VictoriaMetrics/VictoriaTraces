import { createContext, FC, useContext, useMemo, useReducer, useEffect, Dispatch } from "react";
import { QueryAction, QueryState, initialQueryState, reducer } from "./reducer";
import { setQueriesToStorage } from "../../components/QueryHistory/utils";
import { saveToStorage } from "../../utils/storage";

type QueryStateContextType = { state: QueryState, dispatch: Dispatch<QueryAction> };

export const QueryStateContext = createContext<QueryStateContextType>({} as QueryStateContextType);

export const useQueryState = (): QueryState => useContext(QueryStateContext).state;
export const useQueryDispatch = (): Dispatch<QueryAction> => useContext(QueryStateContext).dispatch;

export const QueryStateProvider: FC = ({ children }) => {
  const [state, dispatch] = useReducer(reducer, initialQueryState);

  const contextValue = useMemo(() => {
    return { state, dispatch };
  }, [state, dispatch]);

  useEffect(() => {
    // 每次 queryHistory 变化，落盘
    setQueriesToStorage('LOGS_QUERY_HISTORY', state.queryHistory);
  }, [state.queryHistory]);

  useEffect(() => {
    // 每次 autocomplete 变化，落盘
    saveToStorage("AUTOCOMPLETE", state.autocomplete);
  }, [state.autocomplete]);

  return <QueryStateContext.Provider value={contextValue}>
    {children}
  </QueryStateContext.Provider>;
};


