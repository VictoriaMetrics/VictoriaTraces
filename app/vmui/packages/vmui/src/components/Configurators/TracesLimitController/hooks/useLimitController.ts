import { getFromStorage, saveToStorage } from "../../../../utils/storage";
import { useSearchParams } from "react-router-dom";
import { TRACES_DEFAULT_LIMIT, TRACES_MAX_LIMIT, TRACES_URL_PARAMS } from "../../../../constants/logs";
import { useCallback } from "react";
import { useMemo } from "preact/compat";

const isValidLimit = (value: number): boolean => {
  return Number.isInteger(value) && value > 0 && value <= TRACES_MAX_LIMIT;
};

export const useLimitController = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const rawLimitFromParams = searchParams.get(TRACES_URL_PARAMS.LIMIT);

  const limit = useMemo(() => {
    // 1. Try URL param
    const paramsLimit = Number(rawLimitFromParams);
    if (isValidLimit(paramsLimit)) return paramsLimit;

    // 2. Try session storage
    const storageLimit = Number(getFromStorage("TRACES_LIMIT"));
    if (isValidLimit(storageLimit)) return storageLimit;

    // 3. Fallback
    return TRACES_DEFAULT_LIMIT;
  }, [rawLimitFromParams]);

  const setLimit = useCallback((nextLimit: number) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev);
      next.set(TRACES_URL_PARAMS.LIMIT, `${nextLimit}`);
      return next;
    });

    saveToStorage("TRACES_LIMIT", `${nextLimit}`);
  }, [setSearchParams]);

  return {
    limit,
    setLimit,
  };
};
