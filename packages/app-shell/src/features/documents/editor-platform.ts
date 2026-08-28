export interface LibreOfficeStatus {
  installed: boolean;
  version?: string;
  executable?: string;
}

export interface LibreOfficeExtensionStatus {
  installed: boolean;
  version?: string;
  packageAvailable: boolean;
  lastError?: string;
}

export interface LibreOfficeWorkingCopy {
  id: string;
  documentId: string;
  baseVersionId: string;
  fileName: string;
}

export interface EditorPlatform {
  kind: "web" | "desktop";
  checkLibreOffice?: () => Promise<LibreOfficeStatus>;
  checkLibreOfficeExtension?: () => Promise<LibreOfficeExtensionStatus>;
  installLibreOfficeExtension?: (confirmed: boolean) => Promise<LibreOfficeExtensionStatus>;
  documents?: DocumentGateway;
  administration?: AdministrationGateway;
  organizations?: OrganizationGateway;
}

export interface OrganizationGateway {
  session(): Promise<SessionContext>;
  platformOrganizations(): Promise<OrganizationRecord[]>;
  createOrganization(
    name: string,
    ownerEmail: string,
  ): Promise<{ organization: OrganizationRecord; ownerInvitation?: OrganizationInvitation }>;
  updateOrganization(
    id: string,
    value: { status?: "active" | "suspended"; shareDocumentTitlesWithPlatformAdmins?: boolean },
  ): Promise<OrganizationRecord>;
  platformDashboard(): Promise<OrganizationDashboard>;
  organizationDashboard(id: string): Promise<OrganizationDashboard>;
  reportDefinitions(): Promise<ReportDefinition[]>;
  reportRuns(organizationId: string): Promise<ReportRun[]>;
  createReportRun(
    organizationId: string,
    definitionId: ReportRun["definitionId"],
    format: ReportRun["format"],
    filters: Record<string, string>,
  ): Promise<ReportRun>;
  exportReport(runId: string): Promise<ReportRun>;
  members(organizationId: string): Promise<OrganizationMember[]>;
  invitations(organizationId: string): Promise<OrganizationInvitation[]>;
  invite(
    organizationId: string,
    email: string,
    role: AuthorizationRole,
  ): Promise<OrganizationInvitation>;
  updateMember(
    organizationId: string,
    userId: string,
    value: { role?: AuthorizationRole; status?: "active" | "suspended" },
  ): Promise<void>;
  removeMember(organizationId: string, userId: string): Promise<void>;
  resendInvitation(id: string): Promise<OrganizationInvitation>;
  revokeInvitation(id: string): Promise<void>;
  settings(organizationId: string): Promise<OrganizationRecord>;
  updateSettings(
    organizationId: string,
    shareDocumentTitlesWithPlatformAdmins: boolean,
  ): Promise<OrganizationRecord>;
  integrations(): Promise<IntegrationStatus[]>;
  connections(organizationId: string): Promise<IntegrationConnection[]>;
  setIntegrationEnabled(id: string, enabled: boolean): Promise<IntegrationStatus>;
  deleteConnection(id: string): Promise<void>;
  platformDocuments(organizationId: string): Promise<PlatformDocumentMetadata[]>;
  startGoogleOrganizationConnection(
    organizationId: string,
  ): Promise<{ authorizationUrl: string; expiresAt: string }>;
  createGoogleDocument(
    workspaceId: string,
    value: {
      connectionId: string;
      family: "text" | "spreadsheet" | "presentation";
      title: string;
      externalFileId?: string;
    },
  ): Promise<DocumentRecord>;
}

export interface AdministrationGateway {
  folders(workspaceId: string): Promise<FolderRecord[]>;
  createFolder(workspaceId: string, name: string, parentId?: string): Promise<FolderRecord>;
  updateFolder(
    folderId: string,
    value: { name?: string; parentId?: string | null },
  ): Promise<FolderRecord>;
  deleteFolder(folderId: string): Promise<void>;
  groups(organizationId: string): Promise<GroupRecord[]>;
  createGroup(organizationId: string, name: string): Promise<GroupRecord>;
  updateGroup(groupId: string, name: string): Promise<GroupRecord>;
  deleteGroup(groupId: string): Promise<void>;
  groupMembers(groupId: string): Promise<GroupMember[]>;
  addGroupMember(groupId: string, userId: string): Promise<void>;
  removeGroupMember(groupId: string, userId: string): Promise<void>;
  grants(organizationId: string): Promise<AuthorizationGrant[]>;
  createGrant(
    organizationId: string,
    value: {
      principalType: "user" | "group";
      principalId: string;
      scopeType: "organization" | "workspace" | "folder" | "document";
      scopeId: string;
      role: AuthorizationRole;
    },
  ): Promise<AuthorizationGrant>;
  deleteGrant(grantId: string): Promise<void>;
  audit(
    organizationId: string,
    options?: { cursor?: string; workspaceId?: string; eventType?: string; limit?: number },
  ): Promise<AuditPage>;
  editorPolicies(organizationId: string): Promise<EditorPolicy[]>;
  setEditorPolicy(
    organizationId: string,
    family: DocumentFamily,
    providerId: EditorProviderId,
  ): Promise<EditorPolicy>;
  effectiveAccess(
    subjectId: string,
    scopeType: "organization" | "workspace" | "folder" | "document",
    scopeId: string,
  ): Promise<AuthorizationCapability[]>;
}

export interface DocumentGateway {
  initializeWorkspace(): Promise<Workspace>;
  list(workspaceId: string): Promise<DocumentRecord[]>;
  createNative(request: CreateNativeDocumentRequest): Promise<DocumentRecord>;
  createOffice?(
    workspaceId: string,
    family: "text" | "spreadsheet" | "presentation",
    title: string,
    idempotencyKey: string,
  ): Promise<DocumentRecord>;
  uploadOffice?(workspaceId: string, file: File, idempotencyKey: string): Promise<DocumentRecord>;
  createEditorSession?(
    documentId: string,
    providerId: EditorProviderId,
    requestedEdit: boolean,
  ): Promise<EditorSession>;
  get(documentId: string): Promise<DocumentRecord>;
  saveNative(request: SaveNativeDocumentRequest): Promise<DocumentRecord>;
  versions(documentId: string): Promise<DocumentVersion[]>;
  openInLibreOffice?(documentId: string): Promise<LibreOfficeWorkingCopy>;
  saveLibreOfficeWorkingCopy?(
    workingCopyId: string,
    idempotencyKey: string,
  ): Promise<DocumentRecord>;
}
import type {
  CreateNativeDocumentRequest,
  DocumentRecord,
  DocumentVersion,
  SaveNativeDocumentRequest,
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
  EditorProviderId,
  EditorSession,
  SessionContext,
  OrganizationRecord,
  OrganizationDashboard,
  OrganizationInvitation,
  ReportDefinition,
  ReportRun,
  OrganizationMember,
  IntegrationStatus,
  IntegrationConnection,
  PlatformDocumentMetadata,
} from "@documi/shared-types";
