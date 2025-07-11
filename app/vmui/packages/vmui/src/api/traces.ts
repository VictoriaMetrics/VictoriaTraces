export const getTracesUrl = (server: string): string =>
  `${server}/select/logsql/query`;

export const getTraceHitsUrl = (server: string): string =>
  `${server}/select/logsql/hits`;
