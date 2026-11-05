import { useMemo } from "preact/compat";
import { useSearchParams } from "react-router-dom";

export const getTenantSearchParams = (source: URLSearchParams): URLSearchParams => {
  const params = new URLSearchParams();
  const accountID = source.get("accountID");
  const projectID = source.get("projectID");
  if (accountID) params.set("accountID", accountID);
  if (projectID) params.set("projectID", projectID);
  return params;
};

export const buildTraceLocation = (
  source: URLSearchParams,
  traceID: string,
  spanID?: string,
): { pathname: string, search: string } => {
  const params = getTenantSearchParams(source);
  params.set("trace_id", traceID);
  if (spanID) params.set("span_id", spanID);
  return { pathname: "/trace", search: params.toString() };
};

export const useTenant = () => {
  const [searchParams] = useSearchParams();

  const accountID = searchParams.get("accountID") || "0";
  const projectID = searchParams.get("projectID") || "0";

  return useMemo(() => ({
    AccountID: accountID,
    ProjectID: projectID,
  }), [accountID, projectID]);
};
