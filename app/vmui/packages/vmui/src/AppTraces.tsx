import React, { FC, useState } from "preact/compat";
import { HashRouter, Route, Routes } from "react-router-dom";
import AppContextProvider from "./contexts/AppContextProvider";
import ThemeProvider from "./components/Main/ThemeProvider/ThemeProvider";
import ExploreTraces from "./pages/ExploreTraces/ExploreTraces";
import TracesLayout from "./layouts/TracesLayout/TracesLayout";
import "./constants/markedPlugins";

const AppTraces: FC = () => {
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
                  element={<ExploreTraces/>}
                />
              </Route>
            </Routes>
          )}
        </>
      </AppContextProvider>
    </HashRouter>
  </>;
};

export default AppTraces;
