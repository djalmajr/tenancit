import { expect, test } from "@playwright/test";
import { logoutViaUI } from "./support/ui";

function breakGlassTokenFromEnvironment() {
  const token = process.env.TENANCIT_E2E_ADMIN_TOKEN;
  if (!token) throw new Error("TENANCIT_E2E_ADMIN_TOKEN is required");
  return token;
}

test("OIDC login, CSRF, logout, and break-glass stay on distinct boundaries", async ({ page, request }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Acesso administrativo" })).toBeVisible();
  await page.getByRole("button", { name: "Entrar com SSO" }).click();

  await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
  await expect.poll(() => page.evaluate(() => localStorage.getItem("tenancitAdminToken"))).toBeNull();

  const rejectedCSRF = await page.evaluate(async () => {
    const response = await fetch("/v1/admin/tenants", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ slug: "csrf-must-fail", name: "CSRF must fail" }),
    });
    return { status: response.status, body: await response.json() };
  });
  expect(rejectedCSRF).toEqual({ status: 403, body: { error: "csrf_invalid" } });

  await logoutViaUI(page);
  await expect(page.getByRole("button", { name: "Entrar com SSO" })).toBeVisible();
  expect((await request.get("/v1/auth/session")).status()).toBe(401);

  const breakGlassToken = breakGlassTokenFromEnvironment();
  const recoveryRead = await request.get("/v1/admin/overview", {
    headers: { Authorization: `Bearer ${breakGlassToken}` },
  });
  expect(recoveryRead.status()).toBe(200);
  const destructiveAttempt = await request.post("/v1/admin/tenants", {
    headers: { Authorization: `Bearer ${breakGlassToken}` },
    data: { slug: "break-glass-must-fail", name: "Break-glass must fail" },
  });
  expect(destructiveAttempt.status()).toBe(403);
  expect(await destructiveAttempt.json()).toEqual({ error: "insufficient permission" });
});
