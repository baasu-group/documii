export interface SystemInfo {
  name: string;
  version: string;
  environment: string;
  airGapped: boolean;
}

export interface ProblemDetails {
  type: string;
  title: string;
  status: number;
  detail?: string;
  requestId?: string;
}

export type EditorProviderId =
  | "native"
  | "collabora"
  | "onlyoffice"
  | "libreoffice-desktop"
  | "google-docs"
  | "google-sheets"
  | "google-slides";

export type EditorCapability =
  | "create"
  | "edit"
  | "view"
  | "convert"
  | "export"
  | "collaborative"
  | "browser"
  | "desktop"
  | "offline"
  | "external"
  | "external-process";

export type DocumentFamily = "native" | "text" | "spreadsheet" | "presentation" | "pdf";

export interface EditorProviderStatus {
  id: EditorProviderId;
  available: boolean;
  reason?: string;
  capabilities: EditorCapability[];
  families: DocumentFamily[];
}

export interface EditorProviderList {
  providers: EditorProviderStatus[];
}

export interface EditorSession {
  providerId: EditorProviderId;
  mode: string;
  url: string;
  accessToken?: string;
  expiresAt?: string;
  readOnly: boolean;
  accessTokenTtl?: number;
}

export type WorkspaceMode = "local" | "server";

export interface Workspace {
  id: string;
  organizationId?: string;
  name: string;
  mode: WorkspaceMode;
}

export interface DocumentVersion {
  id: string;
  documentId: string;
  parentVersionId?: string;
  versionNumber: number;
  sha256: string;
  size: number;
  createdAt: string;
}

export interface DocumentRecord {
  id: string;
  workspaceId: string;
  folderId?: string;
  title: string;
  family: DocumentFamily;
  mimeType: string;
  editorProvider: EditorProviderId;
  storageMode: "managed" | "external";
  currentVersionId: string;
  createdAt: string;
  updatedAt: string;
  content?: unknown;
  externalProvider?: string;
  externalFileId?: string;
  externalUrl?: string;
  externalConnectionId?: string;
}

export interface CreateNativeDocumentRequest {
  workspaceId: string;
  title: string;
  content: unknown;
  idempotencyKey: string;
}

export interface SaveNativeDocumentRequest {
  documentId: string;
  baseVersionId: string;
  content: unknown;
  idempotencyKey: string;
}

export type AuthorizationRole =
  "owner" | "admin" | "manager" | "editor" | "contributor" | "viewer" | "guest";

export type AuthorizationCapability =
  | "read"
  | "create"
  | "edit"
  | "download"
  | "delete"
  | "manage_permissions"
  | "manage_members"
  | "transfer_ownership";

export interface AuthorizationGrant {
  id: string;
  organizationId: string;
  principalType: "user" | "group";
  principalId: string;
  scopeType: "organization" | "workspace" | "folder" | "document";
  scopeId: string;
  role: AuthorizationRole;
  createdAt: string;
}

export interface AuditEvent {
  id: string;
  type: string;
  organizationId?: string;
  workspaceId: string;
  actorId?: string;
  resourceId?: string;
  requestId?: string;
  timestamp: string;
  data: Record<string, unknown>;
}

export interface FolderRecord {
  id: string;
  organizationId: string;
  workspaceId: string;
  parentId?: string;
  name: string;
  createdAt: string;
  updatedAt: string;
}

export interface GroupRecord {
  id: string;
  organizationId: string;
  name: string;
  createdAt: string;
}

export interface GroupMember {
  id: string;
  displayName: string;
  email?: string;
}

export interface EditorPolicy {
  organizationId: string;
  family: DocumentFamily;
  providerId: EditorProviderId;
  updatedAt: string;
}

export interface AuditPage {
  events: AuditEvent[];
  nextCursor?: string;
}

export type OrganizationStatus = "pending" | "active" | "suspended";
export interface SessionUser {
  id: string;
  issuer: string;
  externalSubject: string;
  email?: string;
  displayName: string;
  emailVerified: boolean;
}
export interface SessionOrganization {
  id: string;
  name: string;
  role: AuthorizationRole;
  status: OrganizationStatus;
}
export interface SessionContext {
  user: SessionUser;
  platformAdmin: boolean;
  organizations: SessionOrganization[];
  activeOrganizationId?: string;
  workspaces: Workspace[];
  platformCapabilities: string[];
}
export interface OrganizationRecord {
  id: string;
  name: string;
  status: OrganizationStatus;
  shareDocumentTitlesWithPlatformAdmins: boolean;
  createdAt: string;
  updatedAt: string;
}
export interface PlatformDocumentMetadata {
  id: string;
  title?: string;
  family: DocumentFamily;
  mimeType: string;
  editorProvider: EditorProviderId;
  storageMode: "managed" | "external";
  status: string;
  currentVersionId: string;
  size: number;
  createdAt: string;
  updatedAt: string;
}
export interface OrganizationInvitation {
  id: string;
  organizationId: string;
  email: string;
  role: AuthorizationRole;
  status: "pending" | "accepted" | "revoked" | "expired";
  expiresAt: string;
  createdAt: string;
  acceptanceUrl?: string;
  notificationError?: string;
  lastSentAt?: string;
}
export interface OrganizationMember {
  userId: string;
  email?: string;
  displayName: string;
  role: AuthorizationRole;
  status: "active" | "suspended";
  joinedAt: string;
}
export interface OrganizationDashboard {
  organizations?: number;
  suspendedOrganizations?: number;
  members: number;
  documents: number;
  versions: number;
  storageBytes: number;
  externalDocuments: number;
  pendingInvitations: number;
  providerErrors?: number;
  reportFailures?: number;
}
export type IntegrationState =
  | "installed"
  | "disabled"
  | "setup-required"
  | "connected"
  | "error"
  | "unavailable"
  | "blocked-by-air-gap";
export interface IntegrationStatus {
  id: string;
  name: string;
  version: string;
  state: IntegrationState;
  capabilities: string[];
  airGapCompatible: boolean;
  description: string;
}
export interface IntegrationConnection {
  id: string;
  integrationId: string;
  organizationId?: string;
  userId?: string;
  connectionType: string;
  status: string;
  safeMetadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}
export type ReportDefinitionId =
  | "platform-organization-overview"
  | "organization-document-storage"
  | "membership-access-review"
  | "audit-activity"
  | "external-editor-risk";
export interface ReportDefinition {
  id: ReportDefinitionId;
  name: string;
  description: string;
  formats: ("csv" | "xlsx" | "pdf")[];
}
export interface ReportRun {
  id: string;
  organizationId: string;
  workspaceId?: string;
  definitionId: ReportDefinitionId;
  format: "csv" | "xlsx" | "pdf";
  filters: Record<string, string>;
  status: "queued" | "running" | "completed" | "failed";
  documentId?: string;
  errorCode?: string;
  createdBy: string;
  createdAt: string;
  completedAt?: string;
}
