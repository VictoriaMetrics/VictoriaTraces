import { getFromStorage, saveToStorage } from "../../utils/storage";
import { TracesFiledValues } from "../../api/types";
import { AUTOCOMPLETE_LIMITS } from "../../constants/queryAutocomplete";

export interface TracesState {
  markdownParsing: boolean;
  ansiParsing: boolean;
  autocompleteCache: Map<string, TracesFiledValues[]>;
}

export type TracesAction =
  | { type: "SET_MARKDOWN_PARSING", payload: boolean }
  | { type: "SET_ANSI_PARSING", payload: boolean }
  | { type: "SET_AUTOCOMPLETE_CACHE", payload: { key: string, value: TracesFiledValues[] } }


export const initialTracesState: TracesState = {
  markdownParsing: getFromStorage("TRACES_MARKDOWN") === "true",
  ansiParsing: getFromStorage("TRACES_ANSI") === "true",
  autocompleteCache: new Map<string, TracesFiledValues[]>(),
};

export function reducer(state: TracesState, action: TracesAction): TracesState {
  switch (action.type) {
    case "SET_MARKDOWN_PARSING":
      saveToStorage("TRACES_MARKDOWN", `${ action.payload}`);
      return {
        ...state,
        markdownParsing: action.payload
      };
    case "SET_ANSI_PARSING":
      saveToStorage("TRACES_ANSI", `${ action.payload}`);
      return {
        ...state,
        ansiParsing: action.payload
      };
    case "SET_AUTOCOMPLETE_CACHE": {
      if (state.autocompleteCache.size >= AUTOCOMPLETE_LIMITS.cacheLimit) {
        const firstKey = state.autocompleteCache.keys().next().value;
        firstKey && state.autocompleteCache.delete(firstKey);
      }
      state.autocompleteCache.set(action.payload.key, action.payload.value);

      return {
        ...state,
        autocompleteCache: state.autocompleteCache,
      };
    }
    default:
      throw new Error();
  }
}
