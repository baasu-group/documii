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
