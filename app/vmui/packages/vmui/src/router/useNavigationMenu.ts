import { processNavigationItems } from "./utils";
import { getTracesNavigation } from "./navigation";

const getNavigationMenu = () => {
  const menu = getTracesNavigation();
  return processNavigationItems(menu);
};

export default getNavigationMenu;


