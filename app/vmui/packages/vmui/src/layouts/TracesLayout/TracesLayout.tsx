import { FC, useEffect } from "preact/compat";
import Header from "../Header";
import { matchPath, Outlet, useLocation } from "react-router-dom";
import "./style.scss";
import { getAppModeEnable } from "../../utils/app-mode";
import classNames from "classnames";
import Footer from "../Footer";
import { RouterOptions, routerOptions } from "../../router";
import useDeviceDetect from "../../hooks/useDeviceDetect";
import { footerLinksToTraces } from "../../constants/footerLinks";
import WebStorageCheck from "../../components/WebStorageCheck";
import { migrateStorageToPrefixedKeys } from "../../utils/storage";
import { useAppState } from "../../state/common/StateContext";
import {
  useBrowserTabSync
} from "../../components/Configurators/GlobalSettings/BrowserTabController/hooks/useBrowserTabSync";

const TracesLayout: FC = () => {
  const appModeEnable = getAppModeEnable();
  const { isMobile } = useDeviceDetect();
  const { pathname } = useLocation();
  const { isDarkTheme } = useAppState();
  useBrowserTabSync();

  const setDocumentTitle = () => {
    const matchedEntry = Object.entries(routerOptions).find(([path]) => {
      return matchPath(path, pathname);
    });

    const routeTitle =  (matchedEntry?.[1] as RouterOptions)?.title;
    const defaultTitle = "UI for VictoriaTraces";
    document.title = routeTitle ? `${routeTitle} - ${defaultTitle}` : defaultTitle;
  };

  useEffect(setDocumentTitle, [pathname]);

  useEffect(() => {
    const migrateStorage = migrateStorageToPrefixedKeys();
    if (migrateStorage.removed.length || migrateStorage.migrated.length) {
      console.info(migrateStorage);
    }
  }, []);

  return <section
    className={classNames({
      "vm-container": true,
      "vm-container_dark": isDarkTheme
    })}
  >
    <Header/>
    <div
      id="vm-body"
      className={classNames({
        "vm-container-body": true,
        "vm-container-body_mobile": isMobile,
        "vm-container-body_app": appModeEnable
      })}
    >
      <Outlet/>
    </div>
    {!appModeEnable && <Footer links={footerLinksToTraces}/>}

    <WebStorageCheck/>
  </section>;
};

export default TracesLayout;
