import { useCallback } from "react";
import { useHistory } from "react-router-dom";

export type NavigateOptions = {
  replace?: boolean;
  state?: any;
};

export type NavigateFunction = (
  to: string | number | { pathname: string; search?: string; hash?: string },
  options?: NavigateOptions
) => void;

export function useNavigate(): NavigateFunction {
  const history = useHistory();

  const navigate = useCallback<NavigateFunction>(
    (to, options = {}) => {
      if (typeof to === "number") {
        history.go(to);
        return;
      }

      const { replace = false, state } = options;

      if (replace) {
        history.replace(to as any, state);
      } else {
        history.push(to as any, state);
      }
    },
    [history]
  );

  return navigate;
}
