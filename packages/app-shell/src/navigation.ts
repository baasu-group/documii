import type { IconName } from "@documi/ui";

export const navigation = [
  { id: "home", label: "Home", icon: "home" },
  { id: "documents", label: "Documents", icon: "documents" },
  { id: "data", label: "Data", icon: "data" },
  { id: "forms", label: "Forms", icon: "forms" },
  { id: "tasks", label: "Tasks", icon: "tasks" },
  { id: "automations", label: "Automations", icon: "automations" },
  { id: "ai", label: "AI", icon: "ai" },
  { id: "integrations", label: "Integrations", icon: "integrations" },
  { id: "settings", label: "Settings", icon: "settings" },
] as const satisfies ReadonlyArray<{ id: string; label: string; icon: IconName }>;

export type NavigationId = (typeof navigation)[number]["id"];
