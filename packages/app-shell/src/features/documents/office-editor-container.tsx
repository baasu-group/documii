import { createApiClient } from "@documi/api-client";
import type { EditorProviderId, EditorSession } from "@documi/shared-types";
import { useEffect, useRef, useState } from "react";

export function OfficeEditorContainer({
  documentId,
  providerId,
}: {
  documentId: string;
  providerId: EditorProviderId;
}) {
  const [session, setSession] = useState<EditorSession>();
  const [error, setError] = useState<string>();
  const form = useRef<HTMLFormElement>(null);

  useEffect(() => {
    const client = createApiClient({ baseUrl: window.location.origin });
    client
      .createEditorSession(documentId, providerId, true)
      .then(setSession)
      .catch(() => setError("An authorized editor session could not be created."));
  }, [documentId, providerId]);

  useEffect(() => {
    if (session?.mode === "iframe" && session.accessToken) form.current?.submit();
  }, [session]);

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
      </form>
      <iframe
        className="h-[70vh] w-full border-0"
        name="documi-office-frame"
        title="Office document editor"
      />
    </div>
  );
}
