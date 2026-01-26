import queryString from 'query-string';
import { NavigateFunction } from 'react-router-dom-v5-compat';
import { History as RouterHistory, Location } from 'history';

import { TNil } from '../types';

export default function updateUiFind({
  history,
  navigate,
  location,
  trackFindFunction,
  uiFind,
}: {
  history?: RouterHistory;
  navigate?: NavigateFunction;
  location: Location;
  trackFindFunction?: (uiFind: string | TNil) => void;
  uiFind?: string | TNil;
}) {
  const parsed = queryString.parse(location.search);
  const traceId = parsed.traceId;

  // If we are in trace view, do not touch URL search params.
  if (traceId) {
    if (trackFindFunction) trackFindFunction(uiFind);
    return;
  }

  const { uiFind: _oldUiFind, ...queryParams } = parsed;
  if (trackFindFunction) trackFindFunction(uiFind);
  if (uiFind) (queryParams as Record<string, string>).uiFind = uiFind;

  const nextSearch = `?${queryString.stringify(queryParams)}`;

  if (navigate) {
    navigate({ pathname: location.pathname, search: nextSearch }, { replace: true });
  } else if (history) {
    history.replace({ ...location, search: nextSearch });
  }
}
