// The checked-in browser build predates the source-level Log out button.
// Add the control to that build until the missing workspace files are restored.
(function () {
  let displayName;
  const today = new Date();
  const hour = today.getHours();
  const greeting = hour < 12 ? "Good morning" : hour < 17 ? "Good afternoon" : "Good evening";

  function updateHomeGreeting() {
    if (!displayName) return;
    const heading = Array.from(document.querySelectorAll("#root main h1")).find((item) =>
      /^Good (morning|afternoon|evening),/.test(item.textContent.trim())
    );
    if (!heading) return;
    const firstName = displayName.trim().split(/\s+/)[0] || "there";
    const desired = `${greeting}, ${firstName}.`;
    if (heading.textContent !== desired) heading.textContent = desired;
    const date = heading.parentElement?.querySelector("p");
    if (date) {
      const formatted = new Intl.DateTimeFormat(undefined, { weekday: "long", day: "numeric", month: "long" }).format(today);
      if (date.textContent !== formatted) date.textContent = formatted;
    }
  }

  fetch("/api/v1/session-context")
    .then((response) => response.ok ? response.json() : undefined)
    .then((session) => {
      displayName = session?.user?.displayName;
      updateHomeGreeting();
    })
    .catch(() => {});

  async function logout(button) {
    button.disabled = true;
    try {
      const response = await fetch("/api/v1/auth/config");
      if (!response.ok) throw new Error("Authentication configuration is unavailable.");
      const config = await response.json();
      if (config.mode === "development") {
        window.location.assign("/");
        return;
      }
      const discoveryResponse = await fetch(`${config.issuer}/.well-known/openid-configuration`);
      if (!discoveryResponse.ok) throw new Error("Sign-out service is unavailable.");
      const discovery = await discoveryResponse.json();
      if (!discovery.end_session_endpoint) throw new Error("Sign-out is not configured.");
      const key = `user:${config.issuer}:${config.clientId}`;
      const stored = JSON.parse(sessionStorage.getItem(key) || "null");
      const url = new URL(discovery.end_session_endpoint);
      url.searchParams.set("client_id", config.clientId);
      url.searchParams.set("post_logout_redirect_uri", window.location.origin + "/");
      if (stored?.id_token) url.searchParams.set("id_token_hint", stored.id_token);
      sessionStorage.removeItem(key);
      window.location.assign(url.toString());
    } catch (error) {
      button.disabled = false;
      window.alert(error instanceof Error ? error.message : "Unable to log out.");
    }
  }

  function addButton() {
    const header = document.querySelector("#root main > header");
    if (!header) return false;
    if (Array.from(header.querySelectorAll("button")).some((item) => item.textContent.trim() === "Log out")) return true;
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = "Log out";
    button.setAttribute("aria-label", "Log out of Documi");
    button.style.cssText = "margin-left:16px;padding:8px 15px;border:1px solid #d8e0d9;border-radius:12px;background:#fff;color:#163c2f;font:600 14px system-ui,sans-serif;cursor:pointer;white-space:nowrap";
    button.addEventListener("click", () => void logout(button));
    header.appendChild(button);
    return true;
  }

  const observer = new MutationObserver(() => {
    addButton();
    updateHomeGreeting();
  });
  observer.observe(document.getElementById("root"), { childList: true, subtree: true });
  addButton();
})();
