import { processNavigationItems } from "./utils";
import { getTracesNavigation } from "./navigation";

const useNavigationMenu = () => {
  const menu = getTracesNavigation();
  return processNavigationItems(menu);
};

export default useNavigationMenu;


