import { FC, useState } from "react";
import { HashRouter, Route, Routes } from "react-router";
import AppContextProvider from "./contexts/AppContextProvider";
import ThemeProvider from "./components/Main/ThemeProvider/ThemeProvider";
import TracesLayout from "./layouts/TracesLayout/TracesLayout";
import "./constants/markedPlugins";
import JaegerRoutesInVmui from "./jaeger/JaegerRoutesInVmui";

const App: FC = () => {
  const [loadedTheme, setLoadedTheme] = useState(false);

  return (
    <HashRouter>
      <AppContextProvider>
        <>
          <ThemeProvider onLoaded={setLoadedTheme} />
          {loadedTheme && (
            <Routes>
              <Route
                path="/*"
                element={
                  <TracesLayout>
                    <JaegerRoutesInVmui />
                  </TracesLayout>
                }
              />
            </Routes>
          )}
        </>
      </AppContextProvider>
    </HashRouter>
  );
};

export default App;
