import { expect, test } from "@playwright/test";
import { adminToken } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";

test("invalid admin token can be corrected entirely through the UI", { tag: "@pr-critical" }, async ({ page }) => {
  await flowStep("admin-invalid-token-recovery", 1, "abre a barreira administrativa", async () => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Acesso administrativo" })).toBeVisible();
    await expect(page.getByRole("textbox", { name: "Token", exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Entrar" })).toBeVisible();
  });
  await flowStep("admin-invalid-token-recovery", 2, "rejeita token inválido sem vazar o shell", async () => {
    const tokenInput = page.getByRole("textbox", { name: "Token", exact: true });
    const enter = page.getByRole("button", { name: "Entrar" });
    await tokenInput.fill("tnc_admin_invalido");
    await expect(tokenInput).toHaveValue("tnc_admin_invalido");
    await expect(enter).toBeEnabled();
    await enter.click();
    await expect(page.getByRole("heading", { name: "Acesso administrativo" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Tenants", exact: true })).toHaveCount(0);
  });
  await flowStep("admin-invalid-token-recovery", 3, "explica como recuperar o acesso", async () => {
    await expect(page.getByText("Token inválido ou expirado. Informe um token administrativo válido.", { exact: true })).toBeVisible();
  });
  await flowStep("admin-invalid-token-recovery", 4, "aceita a credencial corrigida", async () => {
    await page.getByRole("textbox", { name: "Token", exact: true }).fill(adminToken);
    await page.getByRole("button", { name: "Entrar" }).click();
    await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Tenants", exact: true })).toBeVisible();
  });
  await flowStep("admin-invalid-token-recovery", 5, "encerra a sessão recuperada", async () => {
    await page.getByRole("button", { name: "Sair" }).click();
    await expect(page.getByRole("heading", { name: "Acesso administrativo" })).toBeVisible();
  });
});
