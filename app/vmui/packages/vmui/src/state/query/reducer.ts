import { getFromStorage, saveToStorage } from "../../utils/storage";
import { getOverrideValue } from "../../components/Configurators/GlobalSettings/QueryTimeOverride";

export interface QueryState {
  queryHasTimeFilter: boolean;
  executeQueryTrigger: number;
  autocomplete: boolean;
  autocompleteQuick: boolean;
}

export type QueryAction =
  | { type: "SET_QUERY_HAS_TIME_FILTER", payload: boolean }
  | { type: "RUN_QUERY"}
  | { type: "TOGGLE_AUTOCOMPLETE" }
  | { type: "SET_AUTOCOMPLETE_QUICK", payload: boolean }

export const initialQueryState: QueryState = {
  queryHasTimeFilter: false,
  executeQueryTrigger: 0,
  autocomplete: getFromStorage("AUTOCOMPLETE") as boolean || false,
  autocompleteQuick: false,
};

export function reducer(state: QueryState, action: QueryAction): QueryState {
  switch (action.type) {
    case "SET_QUERY_HAS_TIME_FILTER":
      return {
        ...state,
        queryHasTimeFilter: getOverrideValue() ? action.payload : false
      };
    case "RUN_QUERY":
      return {
        ...state,
        executeQueryTrigger: state.executeQueryTrigger + 1
      };
    case "TOGGLE_AUTOCOMPLETE":
      saveToStorage("AUTOCOMPLETE", !state.autocomplete);
      return {
        ...state,
        autocomplete: !state.autocomplete
      };
    case "SET_AUTOCOMPLETE_QUICK":
      return {
        ...state,
        autocompleteQuick: action.payload
      };
    default:
      throw new Error();
  }
}
