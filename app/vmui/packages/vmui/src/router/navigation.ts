import router, { routerOptions } from "./index";

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

/**
 * VictoriaTraces navigation menu
 */
export const getTracesNavigation = (): NavigationItem[] => [
  {
    label: routerOptions[router.home].title,
    value: router.home,
  },
];
