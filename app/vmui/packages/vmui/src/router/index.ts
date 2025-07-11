import { APP_TYPE_TRACES } from "../constants/appType";

const router = {
  home: "/",
  dashboards: "/dashboards",
  cardinality: "/cardinality",
  topQueries: "/top-queries",
  trace: "/trace",
  withTemplate: "/expand-with-exprs",
  traces: "/traces",
  activeQueries: "/active-queries",
  icons: "/icons",
  query: "/query",
  rawQuery: "/raw-query",
  downsamplingDebug: "/downsampling-filters-debug",
  retentionDebug: "/retention-filters-debug",
};

export interface RouterOptionsHeader {
  tenant?: boolean,
  stepControl?: boolean,
  timeSelector?: boolean,
  executionControls?: boolean,
  globalSettings?: boolean,
  cardinalityDatePicker?: boolean
}

export interface RouterOptions {
  title?: string,
  header: RouterOptionsHeader
}

const routerOptionsDefault = {
  header: {
    tenant: true,
    stepControl: !APP_TYPE_TRACES,
    timeSelector: !APP_TYPE_TRACES,
    executionControls: !APP_TYPE_TRACES,
  }
};

export const routerOptions: { [key: string]: RouterOptions } = {
  [router.home]: {
    title: "Query",
    ...routerOptionsDefault
  },
  [router.rawQuery]: {
    title: "Raw query",
    ...routerOptionsDefault
  },
  [router.cardinality]: {
    title: "Explore cardinality",
    header: {
      tenant: true,
      cardinalityDatePicker: true,
    }
  },
  [router.topQueries]: {
    title: "Top queries",
    header: {
      tenant: true,
    }
  },
  [router.trace]: {
    title: "Trace analyzer",
    header: {}
  },
  [router.dashboards]: {
    title: "Dashboards",
    ...routerOptionsDefault,
  },
  [router.withTemplate]: {
    title: "WITH templates",
    header: {}
  },
  [router.relabel]: {
    title: "Metric relabel debug",
    header: {}
  },
  [router.traces]: {
    title: "Traces Explorer",
    header: {}
  },
  [router.activeQueries]: {
    title: "Active Queries",
    header: {}
  },
  [router.icons]: {
    title: "Icons",
    header: {}
  },
  [router.query]: {
    title: "Query",
    ...routerOptionsDefault
  },
  [router.downsamplingDebug]: {
    title: "Downsampling filters debug",
    header: {}
  },
  [router.retentionDebug]: {
    title: "Retention filters debug",
    header: {}
  }
};

export default router;
