import { Dispatch, useState, useEffect, SetStateAction } from "preact/compat";
import { useSearchParams } from "react-router-dom";

const useStateSearchParams = <T>(defaultState: T, key: string): [T, Dispatch<SetStateAction<T>>] => {
  const [searchParams] = useSearchParams();
  const currentValue = searchParams.get(key) ? searchParams.get(key) as unknown as T : defaultState;
  const [state, setState] = useState<T>(currentValue);

  useEffect(() => {
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- syncs local state from the URL search param when it changes externally, without reverting independent local updates (functional updater avoids needing `state` as a dep)
    setState(prev => ((currentValue as unknown as T) !== prev ? (currentValue as unknown as T) : prev));
  }, [currentValue]);

  return [state, setState];
};

export default useStateSearchParams;
