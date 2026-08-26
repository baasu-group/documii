import type { SystemInfo } from "@documi/shared-types";

export interface ApiClientOptions {
  baseUrl: string;
  fetch?: typeof globalThis.fetch;
}

export function createApiClient({
  baseUrl,
  fetch: fetchImpl = globalThis.fetch,
}: ApiClientOptions) {
  const request = async <T>(path: string): Promise<T> => {
    const response = await fetchImpl(new URL(path, baseUrl));
    if (!response.ok) throw new Error(`Documi API request failed with ${response.status}`);
    return response.json() as Promise<T>;
  };

  return {
    systemInfo: () => request<SystemInfo>("/api/v1/system/info"),
  };
}
