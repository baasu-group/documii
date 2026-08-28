import type { EditorProviderId, EditorSession } from "@documi/shared-types";
import { useEffect, useRef, useState } from "react";

import type { DocumentGateway } from "./editor-platform";

export function OfficeEditorContainer({
  documentId,
  providerId,
  gateway,
}: {
  documentId: string;
  providerId: EditorProviderId;
  gateway?: DocumentGateway;
}) {
  const [session, setSession] = useState<EditorSession>();
  const [error, setError] = useState<string>();
  const form = useRef<HTMLFormElement>(null);
  const [workingCopyId, setWorkingCopyId] = useState<string>();
  const [savedVersion, setSavedVersion] = useState<string>();

  useEffect(() => {
    if (providerId === "libreoffice-desktop" && gateway?.openInLibreOffice) return;
    if (!gateway?.createEditorSession) {
      setError("This workspace cannot create provider sessions.");
      return;
    }
    gateway
      .createEditorSession(documentId, providerId, true)
      .then(setSession)
      .catch(() => setError("An authorized editor session could not be created."));
  }, [documentId, gateway, providerId]);

  useEffect(() => {
    if (session?.mode === "iframe" && session.accessToken) form.current?.submit();
  }, [session]);

  if (providerId === "libreoffice-desktop" && gateway?.openInLibreOffice)
    return (
      <section className="rounded-xl border border-black/10 bg-white p-6">
        <p className="m-0 text-xs font-bold uppercase tracking-wider text-forest">
          Local desktop editor
        </p>
        <h1 className="mt-2 text-3xl font-semibold">LibreOffice working copy</h1>
        <p className="text-sm text-black/55">
          Documi creates a controlled local copy. LibreOffice never receives a database or server
          storage path. Saving back always creates a new immutable version.
        </p>
        {error && <p role="alert" className="rounded-lg bg-red-50 p-3 text-red-800">{error}</p>}
        <div className="flex flex-wrap gap-3">
          <button
            type="button"
            className="rounded-lg bg-forest px-4 py-2 font-semibold text-white"
            onClick={async () => {
              try {
                const copy = await gateway.openInLibreOffice!(documentId);
                setWorkingCopyId(copy.id);
                setError(undefined);
              } catch (reason) {
                setError(reason instanceof Error ? reason.message : "libreoffice_open_failed");
              }
            }}
          >
            Open in LibreOffice
          </button>
          <button
            type="button"
            disabled={!workingCopyId || !gateway.saveLibreOfficeWorkingCopy}
            className="rounded-lg border border-black/10 px-4 py-2 font-semibold disabled:opacity-50"
            onClick={async () => {
              if (!workingCopyId || !gateway.saveLibreOfficeWorkingCopy) return;
              if (!window.confirm("Save this LibreOffice working copy back as a new Documi version?")) return;
              try {
                const result = await gateway.saveLibreOfficeWorkingCopy(workingCopyId, crypto.randomUUID());
                setSavedVersion(result.currentVersionId);
                setError(undefined);
              } catch (reason) {
                setError(reason instanceof Error ? reason.message : "libreoffice_save_failed");
              }
            }}
          >
            Save back to Documi
          </button>
        </div>
        {workingCopyId && <p className="text-sm text-black/50">Working copy ready. Save in LibreOffice, then use Save back.</p>}
        {savedVersion && <p className="text-sm text-forest">Saved immutable version {savedVersion}</p>}
      </section>
    );

  if (error)
    return (
      <div
        role="alert"
        className="rounded-xl border border-amber-200 bg-amber-50 p-5 text-amber-900"
      >
        <strong>Office editor unavailable</strong>
        <p className="mb-0 text-sm">
          {error} Documi did not expose a storage path or unsigned WOPI URL.
        </p>
      </div>
    );
  if (!session) return <p>Creating an authorized editor session…</p>;

  if (session.mode === "external") {
    return (
      <a
        className="font-semibold text-forest underline"
        href={session.url}
        target="_blank"
        rel="noreferrer"
      >
        Open in the provider’s official editor
      </a>
    );
  }

  if (session.mode !== "iframe" || !session.url || !session.accessToken) {
    return <p role="alert">The server returned an unsupported editor session.</p>;
  }

  return (
    <div className="min-h-[70vh] overflow-hidden rounded-xl border border-black/10 bg-white">
      <form ref={form} action={session.url} method="post" target="documi-office-frame">
        <input type="hidden" name="access_token" value={session.accessToken} />
        {session.accessTokenTtl && (
          <input type="hidden" name="access_token_ttl" value={session.accessTokenTtl} />
        )}
      </form>
      <iframe
        className="h-[70vh] w-full border-0"
        name="documi-office-frame"
        title="Office document editor"
      />
    </div>
  );
}
