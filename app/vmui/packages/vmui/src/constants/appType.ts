export enum AppType {
  victoriatraces = "victoriatraces",
}

export const APP_TYPE = import.meta.env.VITE_APP_TYPE;
export const APP_TYPE_TRACES = APP_TYPE === AppType.victoriatraces;


