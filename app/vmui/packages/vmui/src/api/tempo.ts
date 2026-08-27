export const getTempoTraceByIdUrl = (server: string, traceId: string): string =>
  `${server}/select/tempo/api/v2/traces/${encodeURIComponent(traceId)}`;
