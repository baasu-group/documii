import type { EditorProviderStatus } from "@documi/shared-types";
import { useEffect, useState } from "react";

import type {
  EditorPlatform,
  LibreOfficeExtensionStatus,
  LibreOfficeStatus,
} from "./editor-platform";
import { useEditorProviders } from "./use-editor-providers";
import { WorkspaceAdministration } from "./workspace-administration";

const labels: Record<EditorProviderStatus["id"], string> = {
  native: "Native Documi Editor",
  collabora: "Collabora Online",
  onlyoffice: "ONLYOFFICE",
  "libreoffice-desktop": "LibreOffice Desktop",
  "google-docs": "Google Docs",
  "google-sheets": "Google Sheets",
  "google-slides": "Google Slides",
};

export function EditorSettings({ platform }: { platform: EditorPlatform }) {
  const { providers, error } = useEditorProviders(platform.kind === "desktop");
  const [libreOffice, setLibreOffice] = useState<LibreOfficeStatus>();
  const [libreOfficeExtension, setLibreOfficeExtension] =
    useState<LibreOfficeExtensionStatus>();
  const [installingExtension, setInstallingExtension] = useState(false);
  const [section, setSection] = useState<"editors" | "access" | "audit">("editors");
  useEffect(() => {
    if (platform.kind === "desktop")
      Promise.all([
        platform.checkLibreOffice?.().then(setLibreOffice),
        platform.checkLibreOfficeExtension?.().then(setLibreOfficeExtension),
      ]).catch(() => setLibreOffice({ installed: false }));
  }, [platform]);

  return (
    <section>
      <h1 className="m-0 text-4xl font-semibold">Workspace settings</h1>
      <p className="text-black/55">
        {platform.kind === "desktop"
          ? "Local availability comes from fixed desktop capability checks."
          : "Availability comes from the server. Secret values are never returned here."}
      </p>
      {error && (
        <p role="alert" className="text-red-800">
          {error}
        </p>
      )}
      <nav className="mt-6 flex gap-2" aria-label="Workspace settings sections">
        {(["editors", "access", "audit"] as const).map((value) => (
          <button
            key={value}
            type="button"
            onClick={() => setSection(value)}
            className={`rounded-lg px-4 py-2 text-sm font-semibold ${section === value ? "bg-forest text-white" : "bg-white text-black/55"}`}
          >
            {value === "editors" ? "Document editors" : value === "access" ? "Access" : "Audit"}
          </button>
        ))}
      </nav>
      {section === "editors" && (
        <div className="mt-6 grid gap-3">
          {providers.map((provider) => {
            const local =
              provider.id === "libreoffice-desktop" && platform.kind === "desktop"
                ? libreOffice
                : undefined;
            const available = local?.installed ?? provider.available;
            return (
              <article
                key={provider.id}
                className="rounded-xl border border-black/[0.08] bg-white p-5"
              >
                <div className="flex items-center justify-between">
                  <strong>{labels[provider.id]}</strong>
                  <span className={available ? "text-forest" : "text-black/45"}>
                    {available ? "Available" : "Unavailable"}
                  </span>
                </div>
                {!available &&
                provider.id === "libreoffice-desktop" &&
                platform.kind === "desktop" ? (
                  <p className="mb-0 text-sm text-black/55">
                    LibreOffice is not installed. It is optional and only required for “Open in
                    LibreOffice”. See the installation instructions.
                  </p>
                ) : null}
                {local?.version && (
                  <p className="mb-0 text-sm text-black/55">Version {local.version}</p>
                )}
                {provider.id === "libreoffice-desktop" && platform.kind === "desktop" && local?.installed && (
                  <div className="mt-3 rounded-lg bg-paper p-3 text-sm">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span>
                        Documi extension: {libreOfficeExtension?.installed ? "Installed" : "Not installed"}
                        {libreOfficeExtension?.version ? ` · ${libreOfficeExtension.version}` : ""}
                      </span>
                      {!libreOfficeExtension?.installed && (
                        <button
                          type="button"
                          disabled={installingExtension || !libreOfficeExtension?.packageAvailable}
                          className="rounded-md border border-black/10 bg-white px-3 py-1.5 font-semibold disabled:opacity-50"
                          onClick={async () => {
                            if (!window.confirm("Install the trusted Documi extension into this LibreOffice installation?")) return;
                            setInstallingExtension(true);
                            try {
                              setLibreOfficeExtension(await platform.installLibreOfficeExtension!(true));
                            } finally {
                              setInstallingExtension(false);
                            }
                          }}
                        >
                          {installingExtension ? "Installing…" : "Install extension"}
                        </button>
                      )}
                    </div>
                    <p className="mb-0 mt-2 text-xs text-black/50">
                      Local workspaces only. Installation is explicit; the extension receives only an opaque working-copy ID.
                    </p>
                    {libreOfficeExtension?.lastError && (
                      <p role="alert" className="mb-0 mt-2 text-xs text-red-800">
                        {libreOfficeExtension.lastError.replaceAll("_", " ")}
                      </p>
                    )}
                  </div>
                )}
                {!available && provider.reason && (
                  <p className="mb-0 text-sm text-black/45">
                    {provider.reason.replaceAll("_", " ")}
                  </p>
                )}
                <p className="mb-0 text-xs text-black/45">{provider.capabilities.join(" · ")}</p>
              </article>
            );
          })}
        </div>
      )}
      {section === "editors" && <WorkspaceAdministration platform={platform} view="policies" />}
      {section === "access" && (
        <div className="mt-6">
          <WorkspaceAdministration platform={platform} view="access" />
        </div>
      )}
      {section === "audit" && (
        <div className="mt-6">
          <WorkspaceAdministration platform={platform} view="audit" />
        </div>
      )}
    </section>
  );
}
