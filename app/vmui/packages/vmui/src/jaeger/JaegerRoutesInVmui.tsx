// src/jaeger/JaegerRoutesInVmui.tsx
import React from "react";
import { Provider } from "react-redux";
import { Route, Redirect, Switch } from "react-router-dom";

// Jaeger UI components
import NotFound from "jaeger-ui-lite/src/components/App/NotFound";
import Page from "jaeger-ui-lite/src/components/App/Page";
import TraceSearchPage from "jaeger-ui-lite/src/components/TraceSearchPage";
import { ROUTE_PATH as traceSearchPath } from "jaeger-ui-lite/src/components/TraceSearchPage/url";

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
  return (
    <Provider store={store as any}>
      {/* @ts-ignore */}
      <Page>
        <Switch>
          <Route path={traceSearchPath}>
            <TraceSearchPage />
          </Route>
          <Route exact path="/">
            <Redirect to={traceSearchPath} />
          </Route>
          <Route exact path={prefixUrl()}>
            <Redirect to={traceSearchPath} />
          </Route>
          <Route exact path={prefixUrl("/")}>
            <Redirect to={traceSearchPath} />
          </Route>

          <Route>
            <NotFound error="Page not found" />
          </Route>
        </Switch>
      </Page>
    </Provider>
  );
}
