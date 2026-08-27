import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";

const apiURL = process.env.DOCUMI_E2E_API_URL ?? "http://127.0.0.1:8080";

async function accessToken(page) {
  return page.evaluate(() => {
    for (let index = 0; index < sessionStorage.length; index += 1) {
      const value = sessionStorage.getItem(sessionStorage.key(index));
      if (!value) continue;
      try {
        const parsed = JSON.parse(value);
        if (parsed.access_token) return parsed.access_token;
      } catch {
        // Ignore unrelated application session data.
      }
    }
    throw new Error("OIDC access token was not stored in the browser session");
  });
}

async function api(page, token, path) {
  const response = await page.request.get(`${apiURL}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return response.json();
}

test("OIDC user creates, opens, locks, and versions a Collabora document", async ({ page }) => {
  await page.goto("/");
  await page.locator("#username").fill("acceptance-owner");
  await page.locator("#password").fill("documi_acceptance_only");
  await page.locator("#kc-login").click();
  await expect(
    page.getByRole("heading", { name: /Good (morning|afternoon|evening)/ }),
  ).toBeVisible();

  const token = await accessToken(page);
  const { workspaces } = await api(page, token, "/api/v1/workspaces");
  expect(workspaces).toHaveLength(1);

  await page.getByRole("button", { name: "Documents" }).click();
  await expect(page.getByRole("button", { name: "Office · Text Document" })).toBeEnabled();
  await page.getByRole("button", { name: "Office · Text Document" }).click();

  await expect(page).toHaveURL(/\/documents\/[0-9a-f-]+\/edit\?provider=collabora$/);
  const documentId = new URL(page.url()).pathname.split("/")[2];
  await expect(page.locator('input[name="access_token"]')).toHaveValue(/.+/);
  const wopiToken = await page.locator('input[name="access_token"]').inputValue();

  await expect
    .poll(() => page.frames().some((frame) => frame.url().startsWith("http://localhost:9980/")))
    .toBeTruthy();

  const before = await api(page, token, `/api/v1/documents/${documentId}/versions`);
  expect(before.versions).toHaveLength(1);

  const wopiURL = `${apiURL}/api/v1/wopi/files/${documentId}`;
  const currentLockResponse = await page.request.post(
    `${wopiURL}?access_token=${encodeURIComponent(wopiToken)}`,
    { headers: { "X-WOPI-Override": "GET_LOCK" } },
  );
  expect(currentLockResponse.status(), await currentLockResponse.text()).toBe(200);
  let lock = currentLockResponse.headers()["x-wopi-lock"];
  if (!lock) {
    lock = `acceptance-${crypto.randomUUID()}`;
    const lockResponse = await page.request.post(
      `${wopiURL}?access_token=${encodeURIComponent(wopiToken)}`,
      { headers: { "X-WOPI-Override": "LOCK", "X-WOPI-Lock": lock } },
    );
    expect(lockResponse.status(), await lockResponse.text()).toBe(200);
  }

  const encoded = await readFile(
    new URL(
      "../../services/core/internal/modules/documents/templates/assets/blank.docx.b64",
      import.meta.url,
    ),
    "utf8",
  );
  const putResponse = await page.request.post(
    `${wopiURL}/contents?access_token=${encodeURIComponent(wopiToken)}`,
    { headers: { "X-WOPI-Lock": lock }, data: Buffer.from(encoded.replace(/\s/g, ""), "base64") },
  );
  expect(putResponse.status(), await putResponse.text()).toBe(200);

  const after = await api(page, token, `/api/v1/documents/${documentId}/versions`);
  expect(after.versions).toHaveLength(2);
  expect(after.versions[0].parentVersionId).toBe(before.versions[0].id);

  const audit = await api(
    page,
    token,
    `/api/v1/organizations/${workspaces[0].organizationId}/audit?limit=100`,
  );
  expect(audit.events.map((event) => event.type)).toEqual(
    expect.arrayContaining([
      "document.editor.opened",
      "document.collabora.opened",
      "document.editor.saved",
      "document.collabora.saved",
    ]),
  );
});
