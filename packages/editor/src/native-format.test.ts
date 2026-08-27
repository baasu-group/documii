import { describe, expect, it } from "vitest";

import { createEmptyNativeDocument, isNativeDocument } from "./native-format";

describe("native document format", () => {
  it("creates a valid empty document", () => {
    expect(isNativeDocument(createEmptyNativeDocument())).toBe(true);
  });

  it("rejects arbitrary editor JSON without the Documi envelope", () => {
    expect(isNativeDocument({ type: "doc", content: [] })).toBe(false);
  });
});
