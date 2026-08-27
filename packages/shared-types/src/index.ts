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
}
