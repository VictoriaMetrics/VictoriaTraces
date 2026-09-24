import { FC, useState } from "preact/compat";
import { HashRouter, Route, Routes } from "react-router-dom";
import AppContextProvider from "./contexts/AppContextProvider";
import ThemeProvider from "./components/Main/ThemeProvider";
import TraceExplorer from "./pages/TraceExplorer";
import TraceById from "./pages/TraceExplorer/TraceById";
import TracesLayout from "./layouts/TracesLayout";
import PreviewIcons from "./components/Main/Icons/PreviewIcons";
import AllButtonsPreview from "./components/Main/Button/AllButtonsPreview";

const isDev = import.meta.env.DEV;

const App: FC = () => {
  const [loadedTheme, setLoadedTheme] = useState(false);

  return <>
    <HashRouter>
      <AppContextProvider>
        <>
          <ThemeProvider onLoaded={setLoadedTheme}/>
          {loadedTheme && (
            <Routes>
              <Route
                path={"/"}
                element={<TracesLayout/>}
              >
                <Route
                  path={"/"}
                  element={<TraceExplorer/>}
                />
                <Route
                  path={"/trace"}
                  element={<TraceById/>}
                />

                {isDev && (
                  <>
                    <Route
                      path="/icons"
                      element={<PreviewIcons />}
                    />
                    <Route
                      path="/buttons"
                      element={<AllButtonsPreview />}
                    />
                  </>
                )}
              </Route>
            </Routes>
          )}
        </>
      </AppContextProvider>
    </HashRouter>
  </>;
};

export default App;
