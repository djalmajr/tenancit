import { expect, test } from "@playwright/test";
import {
  CatalogCleanup,
  createTenantFixture,
  findDefinitionByKey,
  suspendActiveDefinitions,
} from "./fixtures/catalog";
import { adminRequest, uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";
import { addFieldViaUI, createDefinitionViaUI, loginViaUI } from "./support/ui";

test("inactive definitions disappear from provisioning and return after activation", { tag: "@full" }, async ({ page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();
  const key = uniqueID("toggle");
  const name = `Toggle ${key}`;
  let inactiveCount = -1;

  try {
    await suspendActiveDefinitions(request, cleanup);
    const tenant = await createTenantFixture(request, cleanup, "toggle-tenant");

    await flowStep("definition-deactivation-provisioning", 1, "autentica o operador", async () => {
      await loginViaUI(page);
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    });
    await flowStep("definition-deactivation-provisioning", 2, "cria definição ativa com campo obrigatório", async () => {
      await createDefinitionViaUI(page, cleanup, name, key);
      await addFieldViaUI(page, { key: "host", label: "Host", required: true });
      await expect(page.getByText("ativo", { exact: true })).toBeVisible();
    });
    await flowStep("definition-deactivation-provisioning", 3, "lista a definição no provisionamento", async () => {
      await page.goto(`/tenants/${tenant.id}`);
      await page.getByRole("tab", { name: "Recursos", exact: true }).click();
      await page.getByRole("button", { name: "Adicionar recurso" }).click();
      const dialog = page.getByRole("dialog");
      await dialog.getByRole("combobox").click();
      await expect(page.getByRole("option", { name })).toBeVisible();
      await page.keyboard.press("Escape");
      await dialog.getByRole("button", { name: "Cancelar" }).click();
    });
    await flowStep("definition-deactivation-provisioning", 4, "desativa pela tela de detalhe", async () => {
      const definition = await findDefinitionByKey(request, key);
      await page.goto(`/resource-definitions/${definition.id}`);
      await page.getByRole("button", { name: "Ações" }).click();
      await page.getByRole("menuitem", { name: "Desativar" }).click();
      await expect(page.getByText("inativo", { exact: true })).toBeVisible();
      const overview = await adminRequest<{ activeDefinitions: number }>(request, "get", "/v1/admin/overview");
      inactiveCount = overview.activeDefinitions;
      expect(inactiveCount).toBeGreaterThanOrEqual(0);
    });
    await flowStep("definition-deactivation-provisioning", 5, "remove a definição inativa dos tipos", async () => {
      await page.goto(`/tenants/${tenant.id}`);
      await page.getByRole("tab", { name: "Recursos", exact: true }).click();
      await page.getByRole("button", { name: "Adicionar recurso" }).click();
      const dialog = page.getByRole("dialog");
      await expect(dialog.getByText("Não há definições de recurso ativas.", { exact: true })).toBeVisible();
      await expect(page.getByRole("option", { name })).toHaveCount(0);
      await dialog.getByRole("button", { name: "Cancelar" }).click();
    });
    await flowStep("definition-deactivation-provisioning", 6, "reativa pela tela de detalhe", async () => {
      const definition = await findDefinitionByKey(request, key);
      await page.goto(`/resource-definitions/${definition.id}`);
      await page.getByRole("button", { name: "Ações" }).click();
      await page.getByRole("menuitem", { name: "Ativar" }).click();
      await expect(page.getByText("ativo", { exact: true })).toBeVisible();
      await expect(page.getByText("Definição ativada.", { exact: true })).toBeVisible();
    });
    await flowStep("definition-deactivation-provisioning", 7, "restaura o tipo no provisionamento", async () => {
      await page.goto(`/tenants/${tenant.id}`);
      await page.getByRole("tab", { name: "Recursos", exact: true }).click();
      await page.getByRole("button", { name: "Adicionar recurso" }).click();
      await page.getByRole("dialog").getByRole("combobox").click();
      await expect(page.getByRole("option", { name })).toBeVisible();
      await page.keyboard.press("Escape");
      await page.getByRole("dialog").getByRole("button", { name: "Cancelar" }).click();
    });
    await flowStep("definition-deactivation-provisioning", 8, "reflete a contagem ativa no overview", async () => {
      await page.goto("/");
      const overview = await adminRequest<{ activeDefinitions: number }>(request, "get", "/v1/admin/overview");
      expect(overview.activeDefinitions).toBe(inactiveCount + 1);
      const card = page.getByText("Definições ativas", { exact: true }).locator("../..");
      await expect(card.getByText(String(overview.activeDefinitions), { exact: true })).toBeVisible();
    });
  } finally {
    await cleanup.run(request);
  }
});
