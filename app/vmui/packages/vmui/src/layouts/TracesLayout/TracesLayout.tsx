import Header from "../Header/Header";
import React, { FC, useEffect } from "preact/compat";
import { Outlet, useLocation } from "react-router-dom";
import "../MainLayout/style.scss";
import { getAppModeEnable } from "../../utils/app-mode";
import classNames from "classnames";
import Footer from "../Footer/Footer";
import router, { routerOptions } from "../../router";
import useDeviceDetect from "../../hooks/useDeviceDetect";
import ControlsTracesLayout from "./ControlsTracesLayout";
import useFetchDefaultTimezone from "../../hooks/useFetchDefaultTimezone";
import { footerLinksToTraces } from "../../constants/footerLinks";

const TracesLayout: FC = () => {
  const appModeEnable = getAppModeEnable();
  const { isMobile } = useDeviceDetect();
  const { pathname } = useLocation();

  useFetchDefaultTimezone();

  const setDocumentTitle = () => {
    const defaultTitle = "vmui for VictoriaTraces";
    const routeTitle = routerOptions[router.traces]?.title;
    document.title = routeTitle ? `${routeTitle} - ${defaultTitle}` : defaultTitle;
  };

  useEffect(setDocumentTitle, [pathname]);

  return <section className="vm-container">
    <Header controlsComponent={ControlsTracesLayout}/>
    <div
      className={classNames({
        "vm-container-body": true,
        "vm-container-body_mobile": isMobile,
        "vm-container-body_app": appModeEnable
      })}
    >
      <Outlet/>
    </div>
    {!appModeEnable && <Footer links={footerLinksToTraces}/>}
  </section>;
};

export default TracesLayout;
