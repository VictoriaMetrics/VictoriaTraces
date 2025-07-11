import React, { render } from "preact/compat";
import "./constants/dayjsPlugins";
import reportWebVitals from "./reportWebVitals";
import "./styles/style.scss";
import AppTraces from "./AppTraces";

const getAppComponent = () => {
  return <AppTraces/>;
};

const root = document.getElementById("root");
if (root) render(getAppComponent(), root);


// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
