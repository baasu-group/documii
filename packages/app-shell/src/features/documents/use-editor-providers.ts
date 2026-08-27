import { createApiClient } from "@documi/api-client";
import type { EditorProviderStatus } from "@documi/shared-types";
import { useEffect, useState } from "react";

export function useEditorProviders() {
  const [providers, setProviders] = useState<EditorProviderStatus[]>([]);
  const [error, setError] = useState<string>();

  useEffect(() => {
    const client = createApiClient({ baseUrl: window.location.origin });
    client
      .documentEditors()
      .then((result) => setProviders(result.providers))
      .catch(() => {
        setError("Editor availability could not be loaded. Start the Documi API and try again.");
      });
  }, []);

  return { providers, error };
}
