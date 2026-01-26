import { ApiError } from './api-error';

export type LoadingTraceArchive = { isLoading: true };
export type SuccessfulTraceArchive = { isAcknowledged: false; isArchived: true };
export type ErrorTraceArchive = { error: ApiError; isAcknowledged: false; isArchived: false; isError: true };
export type AcknowledgedTraceArchive = Omit<SuccessfulTraceArchive | ErrorTraceArchive, 'isAcknowledged'> & {
  isAcknowledged: true;
};

export type TraceArchive =
  | LoadingTraceArchive
  | SuccessfulTraceArchive
  | ErrorTraceArchive
  | AcknowledgedTraceArchive;

export type TracesArchive = Record<string, TraceArchive>;
