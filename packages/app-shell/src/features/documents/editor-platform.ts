export interface LibreOfficeStatus {
  installed: boolean;
  version?: string;
  executable?: string;
}

export interface EditorPlatform {
  kind: "web" | "desktop";
  checkLibreOffice?: () => Promise<LibreOfficeStatus>;
}
