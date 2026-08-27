import { createApiClient } from "@documi/api-client";
import type { EditorProviderStatus } from "@documi/shared-types";
import { useEffect, useState } from "react";

const localProviders: EditorProviderStatus[] = [
  {
    id: "native",
    available: true,
    capabilities: ["create", "edit", "view", "desktop", "offline"],
    families: ["native"],
  },
  {
    id: "libreoffice-desktop",
    available: false,
    reason: "not_checked",
    capabilities: ["edit", "view", "desktop", "offline", "external-process"],
    families: ["text", "spreadsheet", "presentation"],
  },
];

export function useEditorProviders(localOnly = false) {
  const [providers, setProviders] = useState<EditorProviderStatus[]>(
    localOnly ? localProviders : [],
  );
  const [error, setError] = useState<string>();

  useEffect(() => {
    if (localOnly) return;
    const client = createApiClient({ baseUrl: window.location.origin });
    client
      .documentEditors()
      .then((result) => setProviders(result.providers))
      .catch(() => {
        setError("Editor availability could not be loaded. Start the Documi API and try again.");
      });
  }, [localOnly]);

  return { providers, error };
}
