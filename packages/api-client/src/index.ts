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
  FolderRecord,
  GroupRecord,
  GroupMember,
  AuthorizationGrant,
  AuthorizationRole,
  AuthorizationCapability,
  AuditPage,
  EditorPolicy,
  DocumentFamily,
  SessionContext,
  OrganizationRecord,
  OrganizationDashboard,
  OrganizationInvitation,
  ReportDefinition,
  ReportRun,
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
    if (response.status === 204) return undefined as T;
    return response.json() as Promise<T>;
  };

  return {
    systemInfo: () => request<SystemInfo>("/api/v1/system/info"),
    sessionContext: () => request<SessionContext>("/api/v1/session-context"),
    platformOrganizations: () =>
      request<{ organizations: OrganizationRecord[] }>("/api/v1/platform/organizations"),
    createOrganization: (name: string, ownerEmail: string) =>
      request<{ organization: OrganizationRecord; ownerInvitation?: OrganizationInvitation }>(
        "/api/v1/platform/organizations",
        {
          method: "POST",
          headers: { "Content-Type": "application/json", "Idempotency-Key": crypto.randomUUID() },
          body: JSON.stringify({ name, ownerEmail }),
        },
      ),
    updateOrganization: (
      id: string,
      value: { status?: "active" | "suspended"; shareDocumentTitlesWithPlatformAdmins?: boolean },
    ) =>
      request<OrganizationRecord>(`/api/v1/platform/organizations/${encodeURIComponent(id)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(value),
      }),
    platformDashboard: () => request<OrganizationDashboard>("/api/v1/platform/dashboard"),
    organizationDashboard: (id: string) =>
      request<OrganizationDashboard>(`/api/v1/organizations/${encodeURIComponent(id)}/dashboard`),
    reportDefinitions: () =>
      request<{ definitions: ReportDefinition[] }>("/api/v1/report-definitions"),
    reportRuns: (organizationId: string) =>
      request<{ runs: ReportRun[] }>(
        `/api/v1/organizations/${encodeURIComponent(organizationId)}/report-runs`,
      ),
    createReportRun: (
      organizationId: string,
      definitionId: ReportRun["definitionId"],
      format: ReportRun["format"],
      filters: Record<string, string>,
    ) =>
      request<ReportRun>(
        `/api/v1/organizations/${encodeURIComponent(organizationId)}/report-runs`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json", "Idempotency-Key": crypto.randomUUID() },
          body: JSON.stringify({ definitionId, format, filters }),
        },
      ),
    exportReport: (runId: string) =>
      request<ReportRun>(`/api/v1/report-runs/${encodeURIComponent(runId)}/exports`, {
        method: "POST",
        headers: { "Idempotency-Key": crypto.randomUUID() },
      }),
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
    createOfficeDocument: (
      workspaceId: string,
      family: "text" | "spreadsheet" | "presentation",
      title: string,
      idempotencyKey: string,
    ) =>
      request<DocumentRecord>(`/api/v1/workspaces/${encodeURIComponent(workspaceId)}/documents`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "Idempotency-Key": idempotencyKey },
        body: JSON.stringify({ title, family }),
      }),
    uploadOfficeDocument: (workspaceId: string, file: File, idempotencyKey: string) => {
      const body = new FormData();
      body.set("file", file);
      return request<DocumentRecord>(
        `/api/v1/workspaces/${encodeURIComponent(workspaceId)}/document-uploads`,
        { method: "POST", headers: { "Idempotency-Key": idempotencyKey }, body },
      );
    },
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
    folders: (workspaceId: string) =>
      request<{ folders: FolderRecord[] }>(
        `/api/v1/workspaces/${encodeURIComponent(workspaceId)}/folders`,
      ),
    createFolder: (workspaceId: string, name: string, parentId?: string) =>
      request<FolderRecord>(`/api/v1/workspaces/${encodeURIComponent(workspaceId)}/folders`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, parentId }),
      }),
    updateFolder: (folderId: string, value: { name?: string; parentId?: string | null }) =>
      request<FolderRecord>(`/api/v1/folders/${encodeURIComponent(folderId)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(value),
      }),
    deleteFolder: (folderId: string) =>
      request<void>(`/api/v1/folders/${encodeURIComponent(folderId)}`, { method: "DELETE" }),
    groups: (organizationId: string) =>
      request<{ groups: GroupRecord[] }>(
        `/api/v1/organizations/${encodeURIComponent(organizationId)}/groups`,
      ),
    createGroup: (organizationId: string, name: string) =>
      request<GroupRecord>(`/api/v1/organizations/${encodeURIComponent(organizationId)}/groups`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      }),
    updateGroup: (groupId: string, name: string) =>
      request<GroupRecord>(`/api/v1/groups/${encodeURIComponent(groupId)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      }),
    deleteGroup: (groupId: string) =>
      request<void>(`/api/v1/groups/${encodeURIComponent(groupId)}`, { method: "DELETE" }),
    groupMembers: (groupId: string) =>
      request<{ members: GroupMember[] }>(`/api/v1/groups/${encodeURIComponent(groupId)}/members`),
    addGroupMember: (groupId: string, userId: string) =>
      request<void>(`/api/v1/groups/${encodeURIComponent(groupId)}/members`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ userId }),
      }),
    removeGroupMember: (groupId: string, userId: string) =>
      request<void>(
        `/api/v1/groups/${encodeURIComponent(groupId)}/members/${encodeURIComponent(userId)}`,
        { method: "DELETE" },
      ),
    grants: (organizationId: string) =>
      request<{ grants: AuthorizationGrant[] }>(
        `/api/v1/organizations/${encodeURIComponent(organizationId)}/grants`,
      ),
    createGrant: (
      organizationId: string,
      value: {
        principalType: "user" | "group";
        principalId: string;
        scopeType: "organization" | "workspace" | "folder" | "document";
        scopeId: string;
        role: AuthorizationRole;
      },
    ) =>
      request<AuthorizationGrant>(
        `/api/v1/organizations/${encodeURIComponent(organizationId)}/grants`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(value),
        },
      ),
    deleteGrant: (grantId: string) =>
      request<void>(`/api/v1/grants/${encodeURIComponent(grantId)}`, { method: "DELETE" }),
    audit: (
      organizationId: string,
      options: { cursor?: string; workspaceId?: string; eventType?: string; limit?: number } = {},
    ) => {
      const query = new URLSearchParams({ limit: String(options.limit ?? 50) });
      if (options.cursor) query.set("cursor", options.cursor);
      if (options.workspaceId) query.set("workspaceId", options.workspaceId);
      if (options.eventType) query.set("eventType", options.eventType);
      return request<AuditPage>(
        `/api/v1/organizations/${encodeURIComponent(organizationId)}/audit?${query}`,
      );
    },
    editorPolicies: (organizationId: string) =>
      request<{ policies: EditorPolicy[] }>(
        `/api/v1/organizations/${encodeURIComponent(organizationId)}/editor-policies`,
      ),
    setEditorPolicy: (
      organizationId: string,
      family: DocumentFamily,
      providerId: EditorProviderId,
    ) =>
      request<EditorPolicy>(
        `/api/v1/organizations/${encodeURIComponent(organizationId)}/editor-policies`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ family, providerId }),
        },
      ),
    effectiveAccess: (
      subjectId: string,
      scopeType: "organization" | "workspace" | "folder" | "document",
      scopeId: string,
    ) => {
      const query = new URLSearchParams({ subjectId, scopeType, scopeId });
      return request<{ capabilities: AuthorizationCapability[] }>(
        `/api/v1/access/effective?${query}`,
      );
    },
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
    createOffice: client.createOfficeDocument,
    uploadOffice: client.uploadOfficeDocument,
    createEditorSession: client.createEditorSession,
    get: client.getDocument,
    saveNative: client.saveNativeDocument,
    versions: async (documentId: string) => (await client.documentVersions(documentId)).versions,
  };
}

export function createServerOrganizationGateway(options: ApiClientOptions) {
  const client = createApiClient(options);
  return {
    session: client.sessionContext,
    platformOrganizations: async () => (await client.platformOrganizations()).organizations,
    createOrganization: client.createOrganization,
    updateOrganization: client.updateOrganization,
    platformDashboard: client.platformDashboard,
    organizationDashboard: client.organizationDashboard,
    reportDefinitions: async () => (await client.reportDefinitions()).definitions,
    reportRuns: async (organizationId: string) => (await client.reportRuns(organizationId)).runs,
    createReportRun: client.createReportRun,
    exportReport: client.exportReport,
  };
}

export function createServerAdministrationGateway(options: ApiClientOptions) {
  const client = createApiClient(options);
  return {
    folders: async (workspaceId: string) => (await client.folders(workspaceId)).folders,
    createFolder: client.createFolder,
    updateFolder: client.updateFolder,
    deleteFolder: client.deleteFolder,
    groups: async (organizationId: string) => (await client.groups(organizationId)).groups,
    createGroup: client.createGroup,
    updateGroup: client.updateGroup,
    deleteGroup: client.deleteGroup,
    groupMembers: async (groupId: string) => (await client.groupMembers(groupId)).members,
    addGroupMember: client.addGroupMember,
    removeGroupMember: client.removeGroupMember,
    grants: async (organizationId: string) => (await client.grants(organizationId)).grants,
    createGrant: client.createGrant,
    deleteGrant: client.deleteGrant,
    audit: client.audit,
    editorPolicies: async (organizationId: string) =>
      (await client.editorPolicies(organizationId)).policies,
    setEditorPolicy: client.setEditorPolicy,
    effectiveAccess: async (
      subjectId: string,
      scopeType: "organization" | "workspace" | "folder" | "document",
      scopeId: string,
    ) => (await client.effectiveAccess(subjectId, scopeType, scopeId)).capabilities,
  };
}
