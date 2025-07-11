export const getDefaultServer = (tenantId?: string): string => {
  return window.location.href.replace(/(\/(select\/)?vmui\/.*|\/#\/.*)/, "");
};
