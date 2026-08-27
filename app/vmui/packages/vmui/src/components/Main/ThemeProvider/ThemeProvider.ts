import { FC, useEffect, useState } from "preact/compat";
import { getContrastColor } from "../../../utils/color";
import { getCssVariable, isSystemDark, setCssVariable } from "../../../utils/theme";
import { AppParams, getAppModeEnable, getAppModeParams } from "../../../utils/app-mode";
import { getFromStorage } from "../../../utils/storage";
import { darkPalette, lightPalette } from "../../../constants/palette";
import { Theme } from "../../../types";
import { useAppDispatch, useAppState } from "../../../state/common/StateContext";
import useSystemTheme from "../../../hooks/useSystemTheme";
import useWindowSize from "../../../hooks/useWindowSize";

interface ThemeProviderProps {
  onLoaded: (val: boolean) => void
}

const colorVariables = [
  "primary",
  "secondary",
  "error",
  "warning",
  "info",
  "success",
];

export const ThemeProvider: FC<ThemeProviderProps> = ({ onLoaded }) => {

  const appModeEnable = getAppModeEnable();
  const { palette: paletteAppMode = {} } = getAppModeParams();
  const { theme } = useAppState();
  const isDarkTheme = useSystemTheme();
  const dispatch = useAppDispatch();
  const windowSize = useWindowSize();

  const [palette, setPalette] = useState(() => ({
    [Theme.dark]: darkPalette,
    [Theme.light]: lightPalette,
    [Theme.system]: isSystemDark() ? darkPalette : lightPalette
  }));

  const setScrollbarSize = () => {
    const { innerWidth, innerHeight } = window;
    const { clientWidth, clientHeight } = document.documentElement;
    setCssVariable("scrollbar-width", `${innerWidth - clientWidth}px`);
    setCssVariable("scrollbar-height", `${innerHeight - clientHeight}px`);
    setCssVariable("vh", `${innerHeight * 0.01}px`);
  };

  const setContrastText = () => {
    colorVariables.forEach((variable, i) => {
      const color = getCssVariable(`color-${variable}`);
      const text = getContrastColor(color);
      setCssVariable(`${variable}-text`, text);

      if (i === colorVariables.length - 1) {
        dispatch({ type: "SET_DARK_THEME" });
        onLoaded(true);
      }
    });
  };

  const setAppModePalette = () => {
    colorVariables.forEach(variable => {
      const colorFromAppMode = paletteAppMode[variable as keyof AppParams["palette"]];
      if (colorFromAppMode) setCssVariable(`color-${variable}`, colorFromAppMode);
    });

    setContrastText();
  };

  const setTheme = () => {
    const theme = (getFromStorage("THEME") || Theme.system) as Theme;
    const result = palette[theme];
    Object.entries(result).forEach(([variable, value]) => {
      setCssVariable(variable, value);
    });
    setContrastText();

    if (appModeEnable) setAppModePalette();
  };

  const updatePalette = () => {
    const newSystemPalette = isSystemDark() ? darkPalette : lightPalette;
    if (palette[Theme.system] === newSystemPalette) {
      setTheme();
      return;
    }
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- syncs the "system" palette entry to the OS theme, which can change independently of this component's props/state
    setPalette(prev => ({
      ...prev,
      [Theme.system]: newSystemPalette
    }));
  };

  useEffect(() => {
    setScrollbarSize();
    setTheme();
    // 'setTheme' intentionally excluded: it's a plain (non-memoized) function recreated every render that closes over 'palette', which is already the dependency driving this effect; adding it would re-run the effect (and re-fire onLoaded/dispatch inside) on every render instead of only when the palette actually changes
    // eslint-disable-next-line @eslint-react/exhaustive-deps
  }, [palette]);

  useEffect(setScrollbarSize, [windowSize]);
  // 'setTheme' intentionally excluded: same non-memoized-closure reason as above
  // eslint-disable-next-line @eslint-react/exhaustive-deps
  useEffect(updatePalette, [theme, isDarkTheme, palette]);

  useEffect(() => {
    if (appModeEnable) {
      dispatch({ type: "SET_THEME", payload: Theme.light });
    }
  }, [appModeEnable, dispatch]);

  return null;
};

export default ThemeProvider;
