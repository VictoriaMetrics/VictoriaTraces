import { FC, useState } from "react";
import { HashRouter, Route, Switch } from "react-router-dom";
import { CompatRouter } from 'react-router-dom-v5-compat';
import AppContextProvider from "./contexts/AppContextProvider";
import ThemeProvider from "./components/Main/ThemeProvider/ThemeProvider";
import TracesLayout from "./layouts/TracesLayout/TracesLayout";
import "./constants/markedPlugins";
import JaegerRoutesInVmui from "./jaeger/JaegerRoutesInVmui";

const App: FC = () => {
  const [loadedTheme, setLoadedTheme] = useState(false);

  return (
    <HashRouter>
      <CompatRouter>
        <AppContextProvider>
          <>
            <ThemeProvider onLoaded={setLoadedTheme} />
            {loadedTheme && (
              <Switch>
                <Route
                  path="/"
                  render={() => (
                    <TracesLayout>
                      <JaegerRoutesInVmui />
                    </TracesLayout>
                  )}
                />
              </Switch>
            )}
          </>
        </AppContextProvider>
      </CompatRouter>
    </HashRouter>
  );
};

export default App;
