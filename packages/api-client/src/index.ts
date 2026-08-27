import type {
  EditorProviderId,
  EditorProviderList,
  EditorProviderStatus,
  EditorSession,
  SystemInfo,
} from "@documi/shared-types";

export interface ApiClientOptions {
  baseUrl: string;
  fetch?: typeof globalThis.fetch;
}

export function createApiClient({
  baseUrl,
  fetch: fetchImpl = globalThis.fetch,
}: ApiClientOptions) {
  const request = async <T>(path: string, init?: RequestInit): Promise<T> => {
    const response = await fetchImpl(new URL(path, baseUrl), init);
    if (!response.ok) throw new Error(`Documi API request failed with ${response.status}`);
    return response.json() as Promise<T>;
  };

  return {
    systemInfo: () => request<SystemInfo>("/api/v1/system/info"),
    documentEditors: () => request<EditorProviderList>("/api/v1/document-editors"),
    documentEditorStatus: (provider: EditorProviderId) =>
      request<EditorProviderStatus>(
        `/api/v1/document-editors/${encodeURIComponent(provider)}/status`,
      ),
    createEditorSession: (
      documentId: string,
      providerId: EditorProviderId,
      requestedEdit: boolean,
    ) =>
      request<EditorSession>(`/api/v1/documents/${encodeURIComponent(documentId)}/editor-session`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ providerId, requestedEdit }),
      }),
  };
}
