import { FC, useEffect } from "react";
import Header from "../Header/Header";
import { useLocation } from "react-router";
import "./style.scss";
import { getAppModeEnable } from "../../utils/app-mode";
import classNames from "classnames";
import Footer from "../Footer/Footer";
import router, { routerOptions } from "../../router";
import useDeviceDetect from "../../hooks/useDeviceDetect";
import ControlsTracesLayout from "./ControlsTracesLayout";
import { footerLinksToLogs } from "../../constants/footerLinks";
import { TracesSearchSettingsProvider } from "./TracesSearchSettingsContext";

const LogsLayout: FC<{ children?: any }> = ({ children }) => {
  const appModeEnable = getAppModeEnable();
  const { isMobile } = useDeviceDetect();
  const { pathname } = useLocation();

  const setDocumentTitle = () => {
    const defaultTitle = "UI for VictoriaTraces";
    const routeTitle = routerOptions[router.home]?.title;

    document.title = routeTitle
      ? `${routeTitle} - ${defaultTitle}`
      : defaultTitle;
  };

  useEffect(setDocumentTitle, [pathname]);

  return (
    <TracesSearchSettingsProvider>
      <section className="vm-trace-container">
        <Header controlsComponent={ControlsTracesLayout} />

        <div
          className={classNames({
            "vm-trace-container-body": true,
            "vm-trace-container-body_mobile": isMobile,
            "vm-trace-container-body_app": appModeEnable,
          })}
        >
          {children}
        </div>

        {!appModeEnable && <Footer links={footerLinksToLogs} />}
      </section>
    </TracesSearchSettingsProvider>
  );
};

export default LogsLayout;
