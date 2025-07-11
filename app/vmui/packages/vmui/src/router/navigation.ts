import router, { routerOptions } from "./index";
import { getTenantIdFromUrl } from "../utils/tenants";

export enum NavigationItemType {
  internalLink,
  externalLink,
}

export interface NavigationItem {
  label?: string,
  value?: string,
  hide?: boolean
  submenu?: NavigationItem[],
  type?: NavigationItemType,
}

interface NavigationConfig {
  serverUrl: string,
  isEnterpriseLicense: boolean,
  showPredefinedDashboards: boolean,
  showAlertLink: boolean,
}

/**
 * Special case for alert link
 */
const getAlertLink = (url: string, showAlertLink: boolean) => {
  // see more https://docs.victoriametrics.com/victoriametrics/cluster-victoriametrics/#vmalert
  const isCluster = !!getTenantIdFromUrl(url);
  const value = isCluster ? `${url}/vmalert` : url.replace(/\/prometheus$/, "/vmalert");
  return {
    label: "Alerts",
    value,
    type: NavigationItemType.externalLink,
    hide: !showAlertLink,
  };
};

/**
 * Submenu for Tools tab
 */
const getToolsNav = (isEnterpriseLicense: boolean) => [
  { value: router.trace },
  { value: router.queryAnalyzer },
  { value: router.withTemplate },
  { value: router.relabel },
  { value: router.downsamplingDebug, hide: !isEnterpriseLicense },
  { value: router.retentionDebug, hide: !isEnterpriseLicense },
];

/**
 * Submenu for Explore tab
 */
const getExploreNav = () => [
  { value: router.metrics },
  { value: router.cardinality },
  { value: router.topQueries },
  { value: router.activeQueries },
];

/**
 * VictoriaTraces navigation menu
 */
export const getTracesNavigation = (): NavigationItem[] => [
  {
    label: routerOptions[router.traces].title,
    value: router.home,
  },
];
