import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { AppShell } from "@documi/app-shell";
import "@documi/ui/styles.css";
import { invoke } from "@tauri-apps/api/core";

const root = document.getElementById("root");
if (!root) throw new Error("Missing #root element");
createRoot(root).render(
  <StrictMode>
    <AppShell platform={{ kind: "desktop", checkLibreOffice: () => invoke("check_libreoffice") }} />
  </StrictMode>,
);
