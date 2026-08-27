import type {
  EditorProviderId,
  EditorProviderList,
  EditorProviderStatus,
  EditorSession,
  CreateNativeDocumentRequest,
  DocumentRecord,
  DocumentVersion,
  SaveNativeDocumentRequest,
  SystemInfo,
  Workspace,
} from "@documi/shared-types";

export interface ApiClientOptions {
  baseUrl: string;
  fetch?: typeof globalThis.fetch;
  accessToken?: () => string | undefined | Promise<string | undefined>;
}

let defaultAccessToken: ApiClientOptions["accessToken"];

export function setDefaultAccessTokenProvider(provider: ApiClientOptions["accessToken"]) {
  defaultAccessToken = provider;
}

export function createApiClient({
  baseUrl,
  fetch: fetchImpl = globalThis.fetch,
  accessToken,
}: ApiClientOptions) {
  const request = async <T>(path: string, init?: RequestInit): Promise<T> => {
    const token = await (accessToken ?? defaultAccessToken)?.();
    const headers = new Headers(init?.headers);
    if (token) headers.set("Authorization", `Bearer ${token}`);
    const response = await fetchImpl(new URL(path, baseUrl), { ...init, headers });
    if (!response.ok) {
      const problem = (await response.json().catch(() => undefined)) as
        { error?: string } | undefined;
      throw new Error(problem?.error ?? `documi_api_${response.status}`);
    }
    return response.json() as Promise<T>;
  };

  return {
    systemInfo: () => request<SystemInfo>("/api/v1/system/info"),
    documentEditors: () => request<EditorProviderList>("/api/v1/document-editors"),
    workspaces: () => request<{ workspaces: Workspace[] }>("/api/v1/workspaces"),
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
    listDocuments: (workspaceId: string) =>
      request<{ documents: DocumentRecord[] }>(
        `/api/v1/workspaces/${encodeURIComponent(workspaceId)}/documents`,
      ),
    createNativeDocument: (value: CreateNativeDocumentRequest) =>
      request<DocumentRecord>(
        `/api/v1/workspaces/${encodeURIComponent(value.workspaceId)}/documents`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json", "Idempotency-Key": value.idempotencyKey },
          body: JSON.stringify({ title: value.title, family: "native", content: value.content }),
        },
      ),
    getDocument: (documentId: string) =>
      request<DocumentRecord>(`/api/v1/documents/${encodeURIComponent(documentId)}`),
    saveNativeDocument: (value: SaveNativeDocumentRequest) =>
      request<DocumentRecord>(
        `/api/v1/documents/${encodeURIComponent(value.documentId)}/versions`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json", "Idempotency-Key": value.idempotencyKey },
          body: JSON.stringify({ baseVersionId: value.baseVersionId, content: value.content }),
        },
      ),
    documentVersions: (documentId: string) =>
      request<{ versions: DocumentVersion[] }>(
        `/api/v1/documents/${encodeURIComponent(documentId)}/versions`,
      ),
  };
}

export function createServerDocumentGateway(options: ApiClientOptions) {
  const client = createApiClient(options);
  return {
    initializeWorkspace: async () => {
      const { workspaces } = await client.workspaces();
      const workspace = workspaces[0];
      if (!workspace) throw new Error("no_authorized_workspace");
      return workspace;
    },
    list: async (workspaceId: string) => (await client.listDocuments(workspaceId)).documents,
    createNative: client.createNativeDocument,
    get: client.getDocument,
    saveNative: client.saveNativeDocument,
    versions: async (documentId: string) => (await client.documentVersions(documentId)).versions,
  };
}
