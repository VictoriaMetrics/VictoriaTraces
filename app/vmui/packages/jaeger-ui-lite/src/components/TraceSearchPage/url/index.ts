import queryString from 'query-string';

import prefixUrl from '../../../utils/prefix-url';

import { TNil } from '../../../types';
import { TUrlState } from '../../SearchTracePage/url';

export const MAX_LENGTH = 7000;

export const ROUTE_PATH = prefixUrl('/search-trace');

export function getUrl(id: string, uiFind?: string): string {
  const traceUrl = prefixUrl(`/search-trace/${id}`);
  if (!uiFind) return traceUrl;

  return `${traceUrl}?${queryString.stringify({ uiFind })}`;
}

export function getSearchUrl(query?: TUrlState) {
  const searchUrl = prefixUrl(`/search-trace`);
  if (!query) return searchUrl;

  const { traceID, spanLinks, ...rest } = query;
  let ids = traceID;
  if (spanLinks && traceID) {
    ids = (Array.isArray(traceID) ? traceID : [traceID]).filter((id: string) => !spanLinks[id]);
  }
  const stringifyArg = {
    ...rest,
    span:
      spanLinks &&
      Object.keys(spanLinks).reduce((res: string[], trace: string) => {
        return [...res, `${spanLinks[trace]}@${trace}`];
      }, []),
    traceID: ids && ids.length ? ids : undefined,
  };

  const fullUrl = `${searchUrl}?${queryString.stringify(stringifyArg)}`;
  if (fullUrl.length <= MAX_LENGTH) return fullUrl;

  const truncated = fullUrl.slice(0, MAX_LENGTH + 1);
  if (truncated[MAX_LENGTH] === '&') return truncated.slice(0, -1);

  return truncated.slice(0, truncated.lastIndexOf('&'));
}

export function getLocation(id: string, state: Record<string, string> | TNil, uiFind?: string) {
  return {
    state,
    pathname: getUrl(id),
    search: uiFind && queryString.stringify({ uiFind }),
  };
}
