import { useMemo } from "preact/compat";
import { useSearchParams } from "react-router-dom";
import { ACCOUNT_ID_URL_PARAM, PROJECT_ID_URL_PARAM } from "../constants/tenant";
import { SPAN_ID_URL_PARAM, TRACE_ID_URL_PARAM } from "../constants/trace";
import router from "../router";

export const getTenantSearchParams = (source: URLSearchParams): URLSearchParams => {
  const params = new URLSearchParams();
  const accountID = source.get(ACCOUNT_ID_URL_PARAM);
  const projectID = source.get(PROJECT_ID_URL_PARAM);
  if (accountID) params.set(ACCOUNT_ID_URL_PARAM, accountID);
  if (projectID) params.set(PROJECT_ID_URL_PARAM, projectID);
  return params;
};

export const buildTraceLocation = (
  source: URLSearchParams,
  traceID: string,
  spanID?: string,
): { pathname: string, search: string } => {
  const params = getTenantSearchParams(source);
  params.set(TRACE_ID_URL_PARAM, traceID);
  if (spanID) params.set(SPAN_ID_URL_PARAM, spanID);
  return { pathname: router.trace, search: params.toString() };
};

export const useTenant = () => {
  const [searchParams] = useSearchParams();

  const accountID = searchParams.get(ACCOUNT_ID_URL_PARAM) || "0";
  const projectID = searchParams.get(PROJECT_ID_URL_PARAM) || "0";

  return useMemo(() => ({
    AccountID: accountID,
    ProjectID: projectID,
  }), [accountID, projectID]);
};
