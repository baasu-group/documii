import type { DocumentFamily, EditorProviderStatus } from "@documi/shared-types";

const options: Array<{
  label: string;
  family: DocumentFamily;
  external?: EditorProviderStatus["id"];
}> = [
  { label: "Documi Document", family: "native" },
  { label: "Office · Text Document", family: "text" },
  { label: "Office · Spreadsheet", family: "spreadsheet" },
  { label: "Office · Presentation", family: "presentation" },
  { label: "External · Google Docs", family: "text", external: "google-docs" },
  { label: "External · Google Sheets", family: "spreadsheet", external: "google-sheets" },
  { label: "External · Google Slides", family: "presentation", external: "google-slides" },
];

export function NewDocumentMenu({
  providers,
  onCreateNative,
}: {
  providers: EditorProviderStatus[];
  onCreateNative: () => void;
}) {
  const available = options.filter((option) =>
    providers.some(
      (provider) =>
        provider.available &&
        provider.capabilities.includes("create") &&
        provider.families.includes(option.family) &&
        (!option.external || provider.id === option.external),
    ),
  );

  return (
    <div className="mt-5 grid gap-2 rounded-xl border border-black/10 bg-paper p-3 sm:max-w-md">
      {available.map((option) => (
        <button
          key={option.label}
          type="button"
          onClick={option.family === "native" ? onCreateNative : undefined}
          className="rounded-lg border border-black/[0.07] bg-white px-4 py-3 text-left text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60"
        >
          {option.label}
        </button>
      ))}
      <button
        type="button"
        className="rounded-lg border border-black/[0.07] bg-white px-4 py-3 text-left text-sm font-semibold"
      >
        Upload File
      </button>
      {!providers.some((provider) => provider.id.startsWith("google-") && provider.available) && (
        <p className="m-1 text-xs text-black/50">
          Google Workspace is not connected. An administrator can configure it in editor settings.
        </p>
      )}
    </div>
  );
}
