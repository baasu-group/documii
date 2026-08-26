import { describe, expect, it } from "vitest";

describe("web bootstrap", () => {
  it("has a browser document", () => {
    expect(typeof document.createElement).toBe("function");
  });
});
