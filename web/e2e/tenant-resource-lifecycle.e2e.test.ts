import { expect, test } from "@playwright/test";
import {
  CatalogCleanup,
  createDefinitionFixture,
  createTenantFixture,
} from "./fixtures/catalog";
import { flowStep } from "./support/flow-step";
import { loginViaUI, navigateFromSidebar } from "./support/ui";
import { SECRET_MASK } from "../src/lib/secret-display";

test.use({ screenshot: "off", trace: "off" });

test("tenant resource protects secrets throughout its lifecycle", { tag: "@pr-critical" }, async ({ page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();

  try {
    const tenant = await createTenantFixture(request, cleanup, "resource-tenant");
    const { definition } = await createDefinitionFixture(request, cleanup, {
      prefix: "resource-definition",
      fields: [
        { key: "host", label: "Host", dataType: "string", required: true },
        { key: "password", label: "Password", dataType: "string", required: true, isSecret: true },
      ],
    });
    const clearSecret = `secret-${tenant.slug}`;

    await flowStep("tenant-resource-lifecycle", 1, "abre tenants autenticado", async () => {
      await loginViaUI(page);
      await navigateFromSidebar(page, "Tenants");
      await expect(page.getByRole("row").filter({ hasText: tenant.name })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 2, "abre o detalhe na aba recursos", async () => {
      await page.getByRole("row").filter({ hasText: tenant.name }).click();
      await expect(page.getByRole("heading", { name: tenant.name })).toBeVisible();
      await expect(page.getByText("Nenhum recurso", { exact: true })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 3, "abre tipos disponíveis a partir da prontidão", async () => {
      await expect(page.getByText("Prontidão para consumo", { exact: true })).toBeVisible();
      await page.getByRole("button", { name: "Adicionar recurso" }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Adicionar recurso" }) });
      await expect(dialog.getByRole("combobox")).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 4, "materializa os campos da definição", async () => {
      const dialog = page.getByRole("dialog");
      await dialog.getByRole("combobox").click();
      await page.getByRole("option", { name: definition.name }).click();
      await expect(dialog.getByPlaceholder("host", { exact: true })).toHaveAttribute("type", "text");
      await expect(dialog.getByPlaceholder("password")).toHaveAttribute("type", "password");
    });
    await flowStep("tenant-resource-lifecycle", 5, "salva recurso obrigatório e secreto", async () => {
      const dialog = page.getByRole("dialog");
      await dialog.getByPlaceholder("host", { exact: true }).fill(`db.${tenant.slug}.local`);
      await dialog.getByPlaceholder("password").fill(clearSecret);
      await dialog.getByRole("button", { name: "Salvar recurso" }).click();
      await expect(page.getByText(definition.name, { exact: true })).toBeVisible();
      await expect(page.getByText("Recurso adicionado.", { exact: true })).toBeVisible();
      await expect(page.getByText("1", { exact: true }).first()).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 6, "mantém o segredo mascarado", async () => {
      await expect(page.getByText(SECRET_MASK, { exact: true })).toBeVisible();
      expect(await page.locator("body").evaluate(
        (body, sensitive) => body.textContent?.includes(sensitive) ?? false,
        clearSecret,
      )).toBe(false);
    });
    await flowStep("tenant-resource-lifecycle", 7, "habilita controles de revelação", async () => {
      await page.getByRole("button", { name: "Habilitar revelação de segredos" }).click();
      await expect(page.getByText("Revelação por campo habilitada.", { exact: true })).toBeVisible();
      const passwordRow = page.getByRole("row").filter({
        has: page.getByRole("cell", { name: "Password", exact: true }),
      });
      await expect(passwordRow.getByRole("button", { name: "Revelar", exact: true })).toBeVisible();
      expect(await page.locator("body").evaluate(
        (body, sensitive) => body.textContent?.includes(sensitive) ?? false,
        clearSecret,
      )).toBe(false);
    });
    await flowStep("tenant-resource-lifecycle", 8, "revela somente após gesto explícito", async () => {
      const passwordRow = page.getByRole("row").filter({
        has: page.getByRole("cell", { name: "Password", exact: true }),
      });
      await passwordRow.getByRole("button", { name: "Revelar", exact: true }).click();
      expect(await page.locator("body").evaluate(
        (body, sensitive) => body.textContent?.includes(sensitive) ?? false,
        clearSecret,
      )).toBe(true);
      await expect(passwordRow.getByRole("button", { name: "Ocultar", exact: true })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 9, "desativa o recurso", async () => {
      const card = page.getByText(definition.name, { exact: true }).locator("../../..");
      await card.getByRole("button", { name: "Desativar" }).click();
      await expect(card.getByText("inativo", { exact: true })).toBeVisible();
      await expect(page.getByText("Recurso desativado.", { exact: true })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 10, "reativa o recurso", async () => {
      const card = page.getByText(definition.name, { exact: true }).locator("../../..");
      await card.getByRole("button", { name: "Reativar" }).click();
      await expect(card.getByText("ativo", { exact: true })).toBeVisible();
      await expect(page.getByText("Recurso reativado.", { exact: true })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 11, "remove o recurso com confirmação", async () => {
      const card = page.getByText(definition.name, { exact: true }).locator("../../..");
      await card.getByRole("button", { name: "Remover" }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Remover recurso?" }) });
      await expect(dialog).toContainText(definition.name);
      await dialog.getByRole("button", { name: "Remover" }).click();
      await expect(page.getByText("Nenhum recurso", { exact: true })).toBeVisible();
      await expect(page.getByText("Recurso removido.", { exact: true })).toBeVisible();
    });
  } finally {
    await cleanup.run(request);
  }
});
