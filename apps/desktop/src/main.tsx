import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { AppShell } from "@documi/app-shell";
import "@documi/ui/styles.css";
import { invoke } from "@tauri-apps/api/core";
import type { DocumentGateway } from "@documi/app-shell";

const localDocuments: DocumentGateway = {
  initializeWorkspace: () => invoke("local_workspace_initialize"),
  list: (workspaceId) => invoke("local_documents_list", { workspaceId }),
  createNative: (request) => invoke("local_document_create", { request }),
  get: (documentId) => invoke("local_document_get", { documentId }),
  saveNative: (request) => invoke("local_document_save", { request }),
  versions: (documentId) => invoke("local_document_versions", { documentId }),
};

const root = document.getElementById("root");
if (!root) throw new Error("Missing #root element");
createRoot(root).render(
  <StrictMode>
    <AppShell
      platform={{
        kind: "desktop",
        documents: localDocuments,
        checkLibreOffice: () => invoke("check_libreoffice"),
      }}
    />
  </StrictMode>,
);
