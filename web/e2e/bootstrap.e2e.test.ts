import { expect, test } from "@playwright/test";
import { logoutViaUI } from "./support/ui";

function adminTokenFromEnvironment() {
  const token = process.env.TENANCIT_E2E_ADMIN_TOKEN;
  if (!token) throw new Error("TENANCIT_E2E_ADMIN_TOKEN is required");
  return token;
}

test("admin access requires a token and supports login and logout", async ({ page }) => {
  const adminToken = adminTokenFromEnvironment();
  await page.goto("/");

  await expect(page.getByRole("heading", { name: "Acesso administrativo" })).toBeVisible();
  await page.getByRole("textbox", { name: "Token", exact: true }).fill(adminToken);
  await page.getByRole("button", { name: "Entrar" }).click();

  await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
  await logoutViaUI(page);
});
