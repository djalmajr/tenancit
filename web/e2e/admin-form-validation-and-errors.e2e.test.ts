import { expect, test } from "@playwright/test";
import {
  CatalogCleanup,
  createDefinitionFixture,
  createResourceFixture,
  findTenantBySlug,
  suspendActiveDefinitions,
} from "./fixtures/catalog";
import { adminRequest, uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";
import { createDefinitionViaUI, createTenantViaUI, loginViaUI } from "./support/ui";

test.use({ screenshot: "off", trace: "off" });

test("admin forms keep invalid and conflicting data visible for correction", { tag: "@full" }, async ({ page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();
  const slug = uniqueID("dup-tenant");
  const tenantName = `Duplicate ${slug}`;
  const key = uniqueID("dup-definition");
  const definitionName = `Duplicate ${key}`;
  const hostname = `${uniqueID("duplicate")}.e2e.local`;

  try {
    await flowStep("admin-form-validation-and-errors", 1, "autentica o operador", async () => {
      await loginViaUI(page);
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    });
    await flowStep("admin-form-validation-and-errors", 2, "habilita criar tenant só com campos válidos", async () => {
      await page.goto("/tenants");
      await page.getByRole("button", { name: "Novo tenant" }).click();
      const dialog = page.getByRole("dialog");
      const create = dialog.getByRole("button", { name: "Criar tenant" });
      await expect(create).toBeDisabled();
      await dialog.getByLabel("Nome", { exact: true }).fill(tenantName);
      await dialog.getByLabel("Slug", { exact: true }).fill(slug);
      await expect(create).toBeEnabled();
      await dialog.getByRole("button", { name: "Cancelar" }).click();
    });
    await flowStep("admin-form-validation-and-errors", 3, "preserva diálogo no slug duplicado", async () => {
      await createTenantViaUI(page, cleanup, tenantName, slug);
      await page.goto("/tenants");
      await page.getByRole("button", { name: "Novo tenant" }).click();
      const dialog = page.getByRole("dialog");
      await dialog.getByLabel("Nome", { exact: true }).fill(`${tenantName} second`);
      await dialog.getByLabel("Slug", { exact: true }).fill(slug);
      await dialog.getByRole("button", { name: "Criar tenant" }).click();
      await expect(dialog.getByText("Já existe um registro com esse valor. Verifique se não está duplicando.", { exact: true })).toBeVisible();
      await expect(page).toHaveURL(/\/tenants$/);
    });
    await flowStep("admin-form-validation-and-errors", 4, "cancela sem criar efeito colateral", async () => {
      await page.getByRole("dialog").getByRole("button", { name: "Cancelar" }).click();
      const tenants = await adminRequest<Array<{ slug: string }>>(request, "get", "/v1/admin/tenants");
      expect(tenants.filter((tenant) => tenant.slug === slug)).toHaveLength(1);
      await expect(page.getByRole("dialog")).toHaveCount(0);
    });
    await flowStep("admin-form-validation-and-errors", 5, "explica key duplicada sem navegar", async () => {
      await createDefinitionViaUI(page, cleanup, definitionName, key);
      await page.goto("/resource-definitions");
      await page.getByRole("button", { name: "Nova definição" }).first().click();
      const dialog = page.getByRole("dialog");
      await dialog.getByLabel("Key", { exact: true }).fill(key);
      await dialog.getByLabel("Nome", { exact: true }).fill(`${definitionName} second`);
      await dialog.getByRole("button", { name: "Criar definição" }).click();
      await expect(dialog.getByText("Já existe um registro com esse valor. Verifique se não está duplicando.", { exact: true })).toBeVisible();
      await expect(page).toHaveURL(/\/resource-definitions$/);
      await dialog.getByRole("button", { name: "Cancelar" }).click();
    });
    await flowStep("admin-form-validation-and-errors", 6, "explica hostname duplicado no diálogo", async () => {
      const tenant = await findTenantBySlug(request, slug);
      await page.goto(`/tenants/${tenant.id}`);
      await page.getByRole("tab", { name: "Domínios", exact: true }).click();
      await page.getByRole("button", { name: "Adicionar", exact: true }).click();
      let dialog = page.getByRole("dialog");
      await dialog.getByPlaceholder("app.cliente.com").fill(hostname);
      await dialog.getByRole("button", { name: "Adicionar", exact: true }).click();
      await expect(page.getByRole("cell", { name: hostname })).toBeVisible();
      await page.getByRole("button", { name: "Adicionar", exact: true }).click();
      dialog = page.getByRole("dialog");
      await dialog.getByPlaceholder("app.cliente.com").fill(hostname);
      await dialog.getByRole("button", { name: "Adicionar", exact: true }).click();
      await expect(dialog.getByText("Já existe um registro com esse valor. Verifique se não está duplicando.", { exact: true })).toBeVisible();
      await dialog.getByRole("button", { name: "Cancelar" }).click();
    });
    await flowStep("admin-form-validation-and-errors", 7, "desabilita salvar quando não há definições ativas", async () => {
      const tenant = await findTenantBySlug(request, slug);
      await createResourceFixture(request, tenant.id, key, {});
      await suspendActiveDefinitions(request, cleanup);
      await page.reload();
      await page.getByRole("tab", { name: "Recursos", exact: true }).click();
      await page.getByRole("button", { name: "Adicionar recurso" }).click();
      const dialog = page.getByRole("dialog");
      await expect(dialog.getByText("Não há definições de recurso ativas.", { exact: true })).toBeVisible();
      await expect(dialog.getByRole("button", { name: "Salvar recurso" })).toBeDisabled();
      await dialog.getByRole("button", { name: "Cancelar" }).click();
    });
    await flowStep("admin-form-validation-and-errors", 8, "valida obrigatório, número e segredo", async () => {
      const { definition } = await createDefinitionFixture(request, cleanup, {
        prefix: "typed-validation",
        fields: [
          { key: "port", label: "Port", dataType: "int", required: true },
          { key: "password", label: "Password", dataType: "string", required: true, isSecret: true },
        ],
      });
      await page.reload();
      await page.getByRole("tab", { name: "Recursos", exact: true }).click();
      await page.getByRole("button", { name: "Adicionar recurso" }).click();
      const dialog = page.getByRole("dialog");
      await dialog.getByRole("combobox").click();
      await page.getByRole("option", { name: definition.name }).click();
      await expect(dialog.getByPlaceholder("port")).toHaveAttribute("type", "number");
      await expect(dialog.getByPlaceholder("password")).toHaveAttribute("type", "password");
      await dialog.getByPlaceholder("port").fill("5432");
      await expect(dialog.getByRole("button", { name: "Salvar recurso" })).toBeDisabled();
      await dialog.getByPlaceholder("password").fill("validation-secret");
      await expect(dialog.getByRole("button", { name: "Salvar recurso" })).toBeEnabled();
      await dialog.getByRole("button", { name: "Cancelar" }).click();
    });
  } finally {
    await cleanup.run(request);
  }
});
