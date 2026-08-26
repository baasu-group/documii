import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { AppShell } from "@documi/app-shell";
import "@documi/ui/styles.css";

const root = document.getElementById("root");

if (!root) throw new Error("Missing #root element");

createRoot(root).render(
  <StrictMode>
    <AppShell platform="web" />
  </StrictMode>,
);
