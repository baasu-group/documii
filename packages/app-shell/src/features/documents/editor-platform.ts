export interface LibreOfficeStatus {
  installed: boolean;
  version?: string;
  executable?: string;
}

export interface EditorPlatform {
  kind: "web" | "desktop";
  checkLibreOffice?: () => Promise<LibreOfficeStatus>;
  documents?: DocumentGateway;
}

export interface DocumentGateway {
  initializeWorkspace(): Promise<Workspace>;
  list(workspaceId: string): Promise<DocumentRecord[]>;
  createNative(request: CreateNativeDocumentRequest): Promise<DocumentRecord>;
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
} from "@documi/shared-types";
