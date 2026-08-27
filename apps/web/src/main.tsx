import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { AppShell } from "@documi/app-shell";
import {
  createServerAdministrationGateway,
  createServerDocumentGateway,
  createServerOrganizationGateway,
  setDefaultAccessTokenProvider,
} from "@documi/api-client";
import "@documi/ui/styles.css";
import { configureWebAuthentication } from "./auth";

async function start() {
  const root = document.getElementById("root");
  const accessToken = await configureWebAuthentication();
  setDefaultAccessTokenProvider(accessToken);
  const documents = createServerDocumentGateway({ baseUrl: window.location.origin });
  const administration = createServerAdministrationGateway({ baseUrl: window.location.origin });
  const organizations = createServerOrganizationGateway({ baseUrl: window.location.origin });

  if (!root) throw new Error("Missing #root element");

  createRoot(root).render(
    <StrictMode>
      <AppShell platform={{ kind: "web", documents, administration, organizations }} />
    </StrictMode>,
  );
}

void start();
