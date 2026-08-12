import React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import "./stil.css";

const wurzel = document.getElementById("root");
if (!wurzel) {
  throw new Error("Element #root fehlt");
}

createRoot(wurzel).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
