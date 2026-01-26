import { createContext, FC, useContext, useMemo, useState } from "react";
import { DEFAULT_LIMIT } from "jaeger-ui-lite/src/constants/search-form";

export interface DurationRange {
  minDuration: string | null;
  maxDuration: string | null;
}

interface TracesSearchSettingsContextType {
  durationRange: DurationRange;
  setDurationRange: (range: DurationRange) => void;
  resultsLimit: string;
  setResultsLimit: (limit: string) => void;
}

const defaultDurationRange: DurationRange = { minDuration: null, maxDuration: null };

export const TracesSearchSettingsContext = createContext<TracesSearchSettingsContextType>({
  durationRange: defaultDurationRange,
  setDurationRange: () => {},
  resultsLimit: String(DEFAULT_LIMIT),
  setResultsLimit: () => {},
});

export const useTracesSearchSettings = (): TracesSearchSettingsContextType => useContext(TracesSearchSettingsContext);

export const TracesSearchSettingsProvider: FC = ({ children }) => {
  const [durationRange, setDurationRange] = useState<DurationRange>(defaultDurationRange);
  const [resultsLimit, setResultsLimit] = useState<string>(String(DEFAULT_LIMIT));

  const contextValue = useMemo(() => ({
    durationRange,
    setDurationRange,
    resultsLimit,
    setResultsLimit,
  }), [durationRange, resultsLimit]);

  return (
    <TracesSearchSettingsContext.Provider value={contextValue}>
      {children}
    </TracesSearchSettingsContext.Provider>
  );
};
