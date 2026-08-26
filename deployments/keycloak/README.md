# Keycloak development integration

The imported realm defines one public OIDC client with Authorization Code + PKCE for local browser and Tauri origins. It contains no users or client secret. Create local test users in the Keycloak console at `http://localhost:8081`; never commit exported credentials. Production realm policy, TLS, federation, claim mapping, and lifecycle automation are deferred.
