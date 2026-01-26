import React from "react";
import { createRoot } from "react-dom/client";

import "./constants/dayjsPlugins";
import "./styles/style.scss";
import App from "./App";

const container = document.getElementById("root");

if (!container) {
  throw new Error("Root container missing in index.html");
}

const root = createRoot(container);
root.render(<App />);