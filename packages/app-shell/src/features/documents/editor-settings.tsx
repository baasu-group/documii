import type { EditorProviderStatus } from "@documi/shared-types";
import { useEffect, useState } from "react";

import type { EditorPlatform, LibreOfficeStatus } from "./editor-platform";
import { useEditorProviders } from "./use-editor-providers";

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
  const { providers, error } = useEditorProviders();
  const [libreOffice, setLibreOffice] = useState<LibreOfficeStatus>();
  useEffect(() => {
    if (platform.kind === "desktop")
      platform
        .checkLibreOffice?.()
        .then(setLibreOffice)
        .catch(() => setLibreOffice({ installed: false }));
  }, [platform]);

  return (
    <section>
      <h1 className="m-0 text-4xl font-semibold">Document Editors</h1>
      <p className="text-black/55">
        Availability comes from the server. Secret values are never returned here.
      </p>
      {error && (
        <p role="alert" className="text-red-800">
          {error}
        </p>
      )}
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
              {!available && provider.reason && (
                <p className="mb-0 text-sm text-black/45">{provider.reason.replaceAll("_", " ")}</p>
              )}
              <p className="mb-0 text-xs text-black/45">{provider.capabilities.join(" · ")}</p>
            </article>
          );
        })}
      </div>
    </section>
  );
}
