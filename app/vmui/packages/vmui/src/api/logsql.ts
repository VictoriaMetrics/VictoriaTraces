export const getLogsqlQueryUrl = (server: string): string =>
  `${server}/select/logsql/query`;

export const getLogsqlHitsUrl = (server: string): string =>
  `${server}/select/logsql/hits`;
