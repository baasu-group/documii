import {
  NativeDocumentEditor,
  createEmptyNativeDocument,
  type NativeDocument,
} from "@documi/editor";
import { Button } from "@documi/ui";
import type { EditorProviderId } from "@documi/shared-types";
import { useState } from "react";

import { NewDocumentMenu } from "./new-document-menu";
import { OfficeEditorContainer } from "./office-editor-container";
import { useEditorProviders } from "./use-editor-providers";

export function DocumentsFeature({
  documentId,
  providerId = "native",
  onOpenNative,
}: {
  documentId?: string;
  providerId?: EditorProviderId;
  onOpenNative: (documentId: string) => void;
}) {
  const { providers, error } = useEditorProviders();
  const [creating, setCreating] = useState(false);
  const [document, setDocument] = useState<NativeDocument>(createEmptyNativeDocument());
  const [saved, setSaved] = useState(false);

  if (documentId && providerId !== "native") {
    return <OfficeEditorContainer documentId={documentId} providerId={providerId} />;
  }

  if (creating || documentId)
    return (
      <section>
        <div className="mb-5 flex items-center justify-between">
          <div>
            <p className="m-0 text-xs font-bold uppercase tracking-wider text-forest">
              Native editor
            </p>
            <h1 className="mb-0 mt-2 text-3xl font-semibold">Untitled Documi document</h1>
          </div>
          <Button
            variant="secondary"
            onClick={() => {
              setCreating(false);
              window.history.pushState({}, "", "/");
              window.dispatchEvent(new PopStateEvent("popstate"));
            }}
          >
            Close
          </Button>
        </div>
        <NativeDocumentEditor
          initialDocument={document}
          onChange={(next) => {
            setDocument(next);
            setSaved(false);
          }}
          onSave={(next) => {
            setDocument(next);
            setSaved(true);
          }}
        />
        <p className="text-sm text-black/50">
          {saved
            ? "Saved in this local preview."
            : "Unsaved preview — persistence awaits the Documents version service."}
        </p>
      </section>
    );

  return (
    <section className="rounded-2xl border border-black/[0.07] bg-white p-8 shadow-soft">
      <h1 className="m-0 text-4xl font-semibold tracking-[-0.04em]">Documents</h1>
      <p className="mt-3 text-black/55">
        Create managed Documi documents or use an available Office/external provider.
      </p>
      {error && (
        <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-800">
          {error}
        </p>
      )}
      <NewDocumentMenu
        providers={providers}
        onCreateNative={() => {
          setCreating(true);
          onOpenNative("local-preview");
        }}
      />
    </section>
  );
}
