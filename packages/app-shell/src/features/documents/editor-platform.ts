export interface LibreOfficeStatus {
  installed: boolean;
  version?: string;
  executable?: string;
}

export interface EditorPlatform {
  kind: "web" | "desktop";
  checkLibreOffice?: () => Promise<LibreOfficeStatus>;
  documents?: DocumentGateway;
  administration?: AdministrationGateway;
}

export interface AdministrationGateway {
  folders(workspaceId: string): Promise<FolderRecord[]>;
  createFolder(workspaceId: string, name: string, parentId?: string): Promise<FolderRecord>;
  groups(organizationId: string): Promise<GroupRecord[]>;
  createGroup(organizationId: string, name: string): Promise<GroupRecord>;
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
  audit(organizationId: string, cursor?: string): Promise<AuditPage>;
  editorPolicies(organizationId: string): Promise<EditorPolicy[]>;
  setEditorPolicy(
    organizationId: string,
    family: DocumentFamily,
    providerId: EditorProviderId,
  ): Promise<EditorPolicy>;
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
  AuditPage,
  EditorPolicy,
  DocumentFamily,
  EditorProviderId,
  EditorSession,
} from "@documi/shared-types";
