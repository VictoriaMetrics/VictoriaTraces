import { ComponentType } from "react";
import { Provider } from "react-redux";
import { Route, Navigate, Routes } from "react-router";
import { useTimeState } from "../state/time/TimeStateContext";
import { useTracesSearchSettings } from "../layouts/TracesLayout/TracesSearchSettingsContext";

// Jaeger UI components
import NotFound from "jaeger-ui-lite/src/components/App/NotFound";
import Page from "jaeger-ui-lite/src/components/App/Page";
import TraceSearchPage from "jaeger-ui-lite/src/components/TraceSearchPage";
import { ROUTE_PATH as traceSearchPath } from "jaeger-ui-lite/src/components/TraceSearchPage/url";
import { TimeRange, DurationRange } from "jaeger-ui-lite/src/components/TraceSearchPage/SearchForm";

const TraceSearchPageWithSettings = TraceSearchPage as ComponentType<{
  timeRange: TimeRange;
  durationRange: DurationRange;
  resultsLimit: string;
}>;

// Jaeger runtime init
import JaegerAPI, { DEFAULT_API_ROOT } from "jaeger-ui-lite/src/api/jaeger";
import processScripts from "jaeger-ui-lite/src/utils/config/process-scripts";
import prefixUrl from "jaeger-ui-lite/src/utils/prefix-url";
import { store } from "jaeger-ui-lite/src/utils/configure-store";

// Jaeger styles
import "jaeger-ui-lite/src/components/common/vars.css";
import "jaeger-ui-lite/src/components/common/utils.css";
import "antd/dist/reset.css";
// import "jaeger-ui-lite/src/components/App/index.css";
import './ub.css'


JaegerAPI.apiRoot = DEFAULT_API_ROOT;
processScripts();

export default function JaegerRoutesInVmui() {
  const { period } = useTimeState();
  const { durationRange, resultsLimit } = useTracesSearchSettings();

  const timeRange = {
    start: Math.round(period.start * 1e6),
    end: Math.round(period.end * 1e6),
  };

  return (
    <Provider store={store as any}>
      {/* @ts-ignore */}
      <Page>
        <Routes>
          <Route
            path={traceSearchPath}
            element={
              <TraceSearchPageWithSettings
                timeRange={timeRange}
                durationRange={durationRange}
                resultsLimit={resultsLimit}
              />
            }
          />
          <Route path="/" element={<Navigate to={traceSearchPath} replace />} />
          <Route path={prefixUrl()} element={<Navigate to={traceSearchPath} replace />} />
          <Route path={prefixUrl("/")} element={<Navigate to={traceSearchPath} replace />} />

          <Route path="*" element={<NotFound error="Page not found" />} />
        </Routes>
      </Page>
    </Provider>
  );
}
