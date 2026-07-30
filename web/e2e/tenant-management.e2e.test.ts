import { expect, test } from "@playwright/test";
import {
  CatalogCleanup,
  createDefinitionFixture,
  createDomainFixture,
  createResourceFixture,
  findTenantBySlug,
} from "./fixtures/catalog";
import { adminToken, uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";
import { loginViaUI, navigateFromSidebar } from "./support/ui";

test.use({ screenshot: "off", trace: "off" });

test("tenant is created, edited, searched, and permanently deleted through the UI", { tag: "@pr-critical" }, async ({ page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();
  const slug = uniqueID("tenant-ui");
  const hostname = `${uniqueID("domain")}.e2e.local`;
  const initialName = `Tenant ${slug}`;
  const updatedName = `${initialName} updated`;
  let tenantId = "";

  try {
    await flowStep("tenant-management", 1, "abre a lista autenticada de tenants", async () => {
      await loginViaUI(page);
      await navigateFromSidebar(page, "Tenants");
      await expect(page.getByRole("textbox", { name: /Buscar por nome ou slug/ })).toBeVisible();
      await expect(page.getByRole("button", { name: "Novo tenant" })).toBeVisible();
    });
    await flowStep("tenant-management", 2, "abre o formulário de novo tenant", async () => {
      await page.getByRole("button", { name: "Novo tenant" }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Novo tenant" }) });
      await expect(dialog.getByLabel("Nome", { exact: true })).toBeVisible();
      await expect(dialog.getByLabel("Slug", { exact: true })).toBeVisible();
    });
    await flowStep("tenant-management", 3, "cria e navega ao detalhe", async () => {
      const dialog = page.getByRole("dialog");
      await dialog.getByLabel("Nome", { exact: true }).fill(initialName);
      await dialog.getByLabel("Slug", { exact: true }).fill(slug);
      await dialog.getByRole("button", { name: "Criar tenant" }).click();
      await expect(page).toHaveURL(/\/tenants\/[^/]+$/);
      const tenant = await findTenantBySlug(request, slug);
      tenantId = tenant.id;
      cleanup.trackTenant(tenant.id);
      await expect(page).toHaveURL(new RegExp(`/tenants/${tenant.id}$`));
      await expect(page.getByRole("heading", { name: initialName })).toBeVisible();
    });
    await flowStep("tenant-management", 4, "edita dados persistidos", async () => {
      await page.getByRole("button", { name: "Editar" }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Editar tenant" }) });
      await dialog.locator("input").first().fill(updatedName);
      await dialog.getByRole("button", { name: "Salvar" }).click();
      await expect(page.getByRole("heading", { name: updatedName })).toBeVisible();
      await expect(page.getByText("Tenant atualizado.", { exact: true })).toBeVisible();
    });
    await flowStep("tenant-management", 5, "abre domínios a partir da prontidão", async () => {
      await expect(page.getByText("Prontidão para consumo", { exact: true })).toBeVisible();
      await page.getByRole("tab", { name: "Domínios", exact: true }).click();
      await expect(page.getByText("Nenhum domínio. Adicione ao menos um para resolver o tenant.", { exact: true })).toBeVisible();
    });
    await flowStep("tenant-management", 6, "adiciona domínio e atualiza prontidão", async () => {
      await page.getByRole("button", { name: "Adicionar", exact: true }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Adicionar domínio" }) });
      await dialog.getByPlaceholder("app.cliente.com").fill(hostname);
      await dialog.getByRole("button", { name: "Adicionar", exact: true }).click();
      await expect(page.getByRole("cell", { name: hostname, exact: true })).toBeVisible();
      await expect(page.getByText("Domínio adicionado.", { exact: true })).toBeVisible();
    });
    await flowStep("tenant-management", 7, "encontra o tenant por busca", async () => {
      await navigateFromSidebar(page, "Tenants");
      const search = page.getByRole("textbox", { name: /Buscar por nome ou slug/ });
      await search.fill(slug);
      const row = page.getByRole("row").filter({ hasText: updatedName });
      await expect(row).toContainText(slug);
    });
    await flowStep("tenant-management", 8, "remove o domínio com confirmação", async () => {
      await page.getByRole("row").filter({ hasText: updatedName }).click();
      await page.getByRole("tab", { name: "Domínios", exact: true }).click();
      const row = page.getByRole("row").filter({ hasText: hostname });
      await row.getByRole("button", { name: "Ações", exact: true }).click();
      await page.getByRole("menuitem", { name: "Remover", exact: true }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Remover domínio?" }) });
      await expect(dialog).toContainText(hostname);
      await dialog.getByRole("button", { name: "Remover" }).click();
      await expect(page.getByRole("cell", {
        name: "Nenhum domínio. Adicione ao menos um para resolver o tenant.",
        exact: true,
      })).toBeVisible();
      await expect(page.getByText("Domínio removido.", { exact: true })).toBeVisible();
    });
    await flowStep("tenant-management", 9, "protege a exclusão sem slug", async () => {
      const { definition } = await createDefinitionFixture(request, cleanup, {
        fields: [{ key: "password", label: "Password", required: true, isSecret: true }],
        prefix: "tenant-cascade",
      });
      await createDomainFixture(request, tenantId, `${uniqueID("cascade")}.e2e.local`);
      await createResourceFixture(request, tenantId, definition.key, {
        password: `cascade-${slug}`,
      });
      await page.getByRole("tab", { name: "Visão geral", exact: true }).click();
      await page.getByRole("button", { name: "Excluir tenant permanentemente" }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Excluir tenant permanentemente?" }) });
      await expect(dialog).toContainText("domínios, recursos e valores secretos");
      await expect(dialog.getByRole("button", { name: "Excluir permanentemente" })).toBeDisabled();
    });
    await flowStep("tenant-management", 10, "confirma exclusão em cascata", async () => {
      const dialog = page.getByRole("dialog");
      await dialog.getByLabel(`Digite ${slug} para confirmar`).fill(slug);
      await dialog.getByRole("button", { name: "Excluir permanentemente" }).click();
      await expect(page.getByRole("heading", { name: "Tenants", exact: true })).toBeVisible();
      await page.getByRole("textbox", { name: /Buscar por nome ou slug/ }).fill(slug);
      await expect(page.getByRole("row").filter({ hasText: slug })).toHaveCount(0);
      const deletedTenant = await request.get(`/v1/admin/tenants/${tenantId}`, {
        headers: { Authorization: `Bearer ${adminToken}` },
      });
      expect(deletedTenant.status()).toBe(404);
    });
  } finally {
    await cleanup.run(request);
  }
});
