import {
  NativeDocumentEditor,
  createEmptyNativeDocument,
  isNativeDocument,
  type NativeDocument,
} from "@documi/editor";
import type {
  DocumentRecord,
  EditorProviderId,
  EditorProviderStatus,
  Workspace,
} from "@documi/shared-types";
import { Button } from "@documi/ui";
import { useEffect, useState } from "react";

import type { DocumentGateway } from "./editor-platform";
import { NewDocumentMenu } from "./new-document-menu";
import { OfficeEditorContainer } from "./office-editor-container";
import { useEditorProviders } from "./use-editor-providers";

const mutationKey = () => crypto.randomUUID();

export function DocumentsFeature({
  documentId,
  providerId = "native",
  onOpenNative,
  gateway,
  localOnly = false,
}: {
  documentId?: string;
  providerId?: EditorProviderId;
  onOpenNative: (documentId: string) => void;
  gateway?: DocumentGateway;
  localOnly?: boolean;
}) {
  const { providers, error: providerError } = useEditorProviders(localOnly);
  const [workspace, setWorkspace] = useState<Workspace>();
  const [documents, setDocuments] = useState<DocumentRecord[]>([]);
  const [record, setRecord] = useState<DocumentRecord>();
  const [document, setDocument] = useState<NativeDocument>(createEmptyNativeDocument());
  const [status, setStatus] = useState<string>();
  const [error, setError] = useState<string>();

  useEffect(() => {
    if (!gateway) return;
    gateway
      .initializeWorkspace()
      .then(async (next) => {
        setWorkspace(next);
        setDocuments(await gateway.list(next.id));
      })
      .catch((reason: Error) => setError(reason.message));
  }, [gateway]);

  useEffect(() => {
    if (!gateway || !documentId) return;
    gateway
      .get(documentId)
      .then((next) => {
        if (!isNativeDocument(next.content)) throw new Error("invalid_native_document");
        setRecord(next);
        setDocument(next.content);
        setStatus(undefined);
      })
      .catch((reason: Error) => setError(reason.message));
  }, [documentId, gateway]);

  if (documentId && providerId !== "native")
    return <OfficeEditorContainer documentId={documentId} providerId={providerId} />;
  if (documentId)
    return (
      <section>
        <div className="mb-5 flex items-center justify-between">
          <div>
            <p className="m-0 text-xs font-bold uppercase tracking-wider text-forest">
              Native editor
            </p>
            <h1 className="mb-0 mt-2 text-3xl font-semibold">
              {record?.title ?? "Loading document…"}
            </h1>
          </div>
          <Button
            variant="secondary"
            onClick={() => {
              window.history.pushState({}, "", "/");
              window.dispatchEvent(new PopStateEvent("popstate"));
            }}
          >
            Close
          </Button>
        </div>
        {error && (
          <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-800">
            {error === "document_version_conflict"
              ? "This document changed since you opened it. Reopen it before saving so no work is overwritten."
              : error}
          </p>
        )}
        {record && (
          <NativeDocumentEditor
            initialDocument={document}
            onChange={(next) => {
              setDocument(next);
              setStatus("Unsaved changes");
            }}
            onSave={async (next) => {
              if (!gateway) return setError("Document persistence is unavailable.");
              try {
                const saved = await gateway.saveNative({
                  documentId: record.id,
                  baseVersionId: record.currentVersionId,
                  content: next,
                  idempotencyKey: mutationKey(),
                });
                setRecord(saved);
                setDocument(next);
                setStatus(`Saved version ${saved.currentVersionId}`);
                setError(undefined);
              } catch (reason) {
                setError(reason instanceof Error ? reason.message : "save_failed");
              }
            }}
          />
        )}
        <p className="text-sm text-black/50">
          {status ?? (record ? `Current version ${record.currentVersionId}` : "Loading…")}
        </p>
      </section>
    );

  const createNative = async () => {
    if (!gateway || !workspace)
      return setError("Document persistence is unavailable for this workspace.");
    try {
      const created = await gateway.createNative({
        workspaceId: workspace.id,
        title: "Untitled Documi document",
        content: createEmptyNativeDocument(),
        idempotencyKey: mutationKey(),
      });
      setDocuments((current) => [created, ...current]);
      onOpenNative(created.id);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "create_failed");
    }
  };
  const fallbackProviders: EditorProviderStatus[] = [
    {
      id: "native",
      available: true,
      capabilities: ["create", "edit", "view", "desktop", "offline"],
      families: ["native"],
    },
  ];
  return (
    <section className="rounded-2xl border border-black/[0.07] bg-white p-8 shadow-soft">
      <h1 className="m-0 text-4xl font-semibold tracking-[-0.04em]">Documents</h1>
      <p className="mt-3 text-black/55">
        Create managed Documi documents or use an available Office/external provider.
      </p>
      {(error || providerError) && (
        <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-800">
          {error ?? providerError}
        </p>
      )}
      <NewDocumentMenu
        providers={providers.length ? providers : [...fallbackProviders]}
        onCreateNative={createNative}
      />
      <div className="mt-8 grid gap-2">
        {documents.map((item) => (
          <button
            key={item.id}
            type="button"
            onClick={() => onOpenNative(item.id)}
            className="rounded-lg border border-black/[0.07] bg-white px-4 py-3 text-left"
          >
            <strong className="block">{item.title}</strong>
            <span className="text-xs text-black/45">Updated {item.updatedAt}</span>
          </button>
        ))}
      </div>
    </section>
  );
}
