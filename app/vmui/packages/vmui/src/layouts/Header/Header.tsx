import { FC, useMemo } from "preact/compat";
import { NavLink, To, useNavigate, useSearchParams } from "react-router-dom";
import router from "../../router";
import { getTenantSearchParams } from "../../hooks/useTenant";
import { getAppModeEnable, getAppModeParams } from "../../utils/app-mode";
import { LogoTracesIcon } from "../../components/Main/Icons";
import { getCssVariable } from "../../utils/theme";
import "./style.scss";
import classNames from "classnames";
import { useAppState } from "../../state/common/StateContext";
import HeaderNav from "./HeaderNav";
import SidebarHeader from "./SidebarNav/SidebarHeader";
import HeaderControls from "./HeaderControls";
import useDeviceDetect from "../../hooks/useDeviceDetect";
import useWindowSize from "../../hooks/useWindowSize";

const Logo = ({ to }: { to: To }) => <NavLink to={to}><LogoTracesIcon/></NavLink>;

const Header: FC = () => {
  const { isMobile } = useDeviceDetect();

  const windowSize = useWindowSize();
  // eslint-disable-next-line @eslint-react/exhaustive-deps -- windowSize is a resize-triggered proxy dependency; window.innerWidth is read directly (not windowSize.width) so the value is correct immediately on first render, before useWindowSize's mount effect populates its state
  const displaySidebar = useMemo(() => window.innerWidth < 1000, [windowSize]);

  const { isDarkTheme } = useAppState();
  const appModeEnable = getAppModeEnable();

  const primaryColor = useMemo(() => {
    const variable = isDarkTheme ? "color-background-block" : "color-primary";
    return getCssVariable(variable);
  }, [isDarkTheme]);

  const { background, color } = useMemo(() => {
    const { headerStyles: {
      background = appModeEnable ? "#FFF" : primaryColor,
      color = appModeEnable ? primaryColor : "#FFF",
    } = {} } = getAppModeParams();

    return { background, color };
  }, [primaryColor, appModeEnable]);

  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const homeLocation = useMemo(() => ({
    pathname: router.home,
    search: getTenantSearchParams(searchParams).toString(),
  }), [searchParams]);

  const onClickLogo = (e: MouseEvent) => {
    const { ctrlKey, metaKey } = e;
    const ctrlMetaKey = ctrlKey || metaKey;
    if (ctrlMetaKey) return; // open in new tab

    e.preventDefault();
    navigate(homeLocation);
    window.location.reload();
  };

  return <header
    className={classNames({
      "vm-header": true,
      "vm-header_app": appModeEnable,
      "vm-header_dark": isDarkTheme,
      "vm-header_sidebar": displaySidebar,
      "vm-header_mobile": isMobile
    })}
    style={{ background, color }}
  >
    {displaySidebar ? (
      <SidebarHeader
        background={background}
        color={color}
      />
    ) : (
      <>
        {!appModeEnable && (
          <div
            className="vm-header-logo"
            onClick={onClickLogo}
            style={{ color }}
          >
            {<Logo to={homeLocation}/>}
          </div>
        )}
        <HeaderNav
          color={color}
          background={background}
        />
      </>
    )}
    {displaySidebar && (
      <div
        className={classNames({
          "vm-header-logo": true,
          "vm-header-logo_mobile": true,
        })}
        onClick={onClickLogo}
        style={{ color }}
      >
        {<Logo to={homeLocation}/>}
      </div>
    )}
    <HeaderControls
      displaySidebar={displaySidebar}
      isMobile={isMobile}
    />
  </header>;
};

export default Header;
