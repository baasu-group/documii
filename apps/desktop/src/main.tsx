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
  createOffice: (workspaceId, family, title, idempotencyKey) =>
    invoke("local_office_document_create", {
      request: { workspaceId, family, title, idempotencyKey },
    }),
  uploadOffice: async (workspaceId, file, idempotencyKey) => {
    const extension = file.name.split(".").pop()?.toLowerCase();
    const family = extension === "docx" ? "text" : extension === "xlsx" ? "spreadsheet" : extension === "pptx" ? "presentation" : undefined;
    if (!family) throw new Error("unsupported_office_file");
    const content = Array.from(new Uint8Array(await file.arrayBuffer()));
    return invoke("local_office_document_import", {
      request: { workspaceId, family, title: file.name.replace(/\.[^.]+$/, ""), fileName: file.name, content, idempotencyKey },
    });
  },
  openInLibreOffice: (documentId) => invoke("open_document_in_libreoffice", { documentId }),
  saveLibreOfficeWorkingCopy: (workingCopyId, idempotencyKey) =>
    invoke("save_libreoffice_working_copy", { request: { workingCopyId, idempotencyKey } }),
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
        checkLibreOfficeExtension: () => invoke("check_documi_libreoffice_extension"),
        installLibreOfficeExtension: (confirmed) =>
          invoke("install_documi_libreoffice_extension", { confirmed }),
      }}
    />
  </StrictMode>,
);
