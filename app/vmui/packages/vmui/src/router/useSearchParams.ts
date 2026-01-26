import { useLocation, useHistory } from "react-router-dom";
import { useCallback, useMemo } from "react";

/**
 * v5-compatible polyfill for react-router v6 useSearchParams
 */
export function useSearchParams(): [
  URLSearchParams,
  (next: URLSearchParams | Record<string, string>) => void
] {
  const location = useLocation();
  const history = useHistory();

  const searchParams = useMemo(() => {
    return new URLSearchParams(location.search);
  }, [location.search]);

  const setSearchParams = useCallback(
    (next: URLSearchParams | Record<string, string>) => {
      let params: URLSearchParams;

      if (next instanceof URLSearchParams) {
        params = next;
      } else {
        params = new URLSearchParams();
        Object.entries(next).forEach(([key, value]) => {
          if (value != null) {
            params.set(key, String(value));
          }
        });
      }

      history.push({
        pathname: location.pathname,
        search: params.toString(),
      });
    },
    [history, location.pathname]
  );

  return [searchParams, setSearchParams];
}
