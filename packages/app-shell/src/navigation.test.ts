import { describe, expect, it } from "vitest";
import { navigation } from "./navigation";

describe("navigation", () => {
  it("exposes every initial product area exactly once", () => {
    expect(navigation.map(({ label }) => label)).toEqual([
      "Home",
      "Documents",
      "Data",
      "Forms",
      "Tasks",
      "Automations",
      "AI",
      "Integrations",
      "Settings",
    ]);
    expect(new Set(navigation.map(({ id }) => id)).size).toBe(navigation.length);
  });
});
