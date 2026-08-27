import type { JSONContent } from "@tiptap/react";

export const nativeDocumentFormat = "documi-json-v1" as const;

export interface NativeDocument {
  format: typeof nativeDocumentFormat;
  content: JSONContent;
}

export function createEmptyNativeDocument(): NativeDocument {
  return {
    format: nativeDocumentFormat,
    content: { type: "doc", content: [{ type: "paragraph" }] },
  };
}

export function isNativeDocument(value: unknown): value is NativeDocument {
  if (typeof value !== "object" || value === null) return false;
  const candidate = value as Partial<NativeDocument>;
  return (
    candidate.format === nativeDocumentFormat &&
    candidate.content?.type === "doc" &&
    Array.isArray(candidate.content.content)
  );
}
