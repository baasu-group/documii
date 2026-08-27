import { UserManager, WebStorageStateStore } from "oidc-client-ts";

interface PublicAuthConfig {
  mode: "development" | "oidc";
  issuer: string;
  clientId: string;
  audience: string;
}

export async function configureWebAuthentication() {
  const response = await fetch("/api/v1/auth/config");
  if (!response.ok) throw new Error("auth_configuration_unavailable");
  const configuration = (await response.json()) as PublicAuthConfig;
  if (configuration.mode === "development") return async () => undefined;

  const manager = new UserManager({
    authority: configuration.issuer,
    client_id: configuration.clientId,
    redirect_uri: new URL("/auth/callback", window.location.origin).toString(),
    post_logout_redirect_uri: window.location.origin,
    response_type: "code",
    scope: "openid profile email",
    userStore: new WebStorageStateStore({ store: window.sessionStorage }),
  });
  if (window.location.pathname === "/auth/callback") {
    await manager.signinRedirectCallback();
    window.history.replaceState({}, "", "/");
  }
  let user = await manager.getUser();
  if (!user || user.expired) {
    await manager.signinRedirect();
    return async () => undefined;
  }
  manager.events.addAccessTokenExpired(() => void manager.signinRedirect());
  return async () => {
    user = await manager.getUser();
    return user?.expired ? undefined : user?.access_token;
  };
}
