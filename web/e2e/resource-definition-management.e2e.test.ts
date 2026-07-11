import { expect, test } from "@playwright/test";
import { CatalogCleanup, type DefinitionRecord, findDefinitionByKey } from "./fixtures/catalog";
import { uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";
import { addFieldViaUI, loginViaUI, navigateFromSidebar } from "./support/ui";

test("resource definition and fields are managed through the UI", { tag: "@full" }, async ({ page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();
  const key = uniqueID("definition-ui");
  const name = `Definition ${key}`;

  try {
    await flowStep("resource-definition-management", 1, "abre o catálogo de recursos", async () => {
      await loginViaUI(page);
      await navigateFromSidebar(page, "Recursos");
      await expect(page.getByRole("heading", { name: "Definições de recurso" })).toBeVisible();
    });
    await flowStep("resource-definition-management", 2, "abre a criação de definição", async () => {
      await page.getByRole("button", { name: "Nova definição" }).first().click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Nova definição" }) });
      await expect(dialog.getByLabel("Key", { exact: true })).toBeVisible();
      await expect(dialog.getByLabel("Nome", { exact: true })).toBeVisible();
      await expect(dialog.getByLabel("Descrição", { exact: true })).toBeVisible();
    });
    await flowStep("resource-definition-management", 3, "cria a definição e abre seu detalhe", async () => {
      const dialog = page.getByRole("dialog");
      await dialog.getByLabel("Key", { exact: true }).fill(key);
      await dialog.getByLabel("Nome", { exact: true }).fill(name);
      await dialog.getByPlaceholder("Conexão de banco por tenant").fill(`E2E ${key}`);
      const [response] = await Promise.all([
        page.waitForResponse((candidate) =>
          candidate.request().method() === "POST"
          && new URL(candidate.url()).pathname === "/v1/admin/resource-definitions"),
        dialog.getByRole("button", { name: "Criar definição" }).click(),
      ]);
      expect(response.ok(), "definition creation response").toBe(true);
      const created = await response.json() as DefinitionRecord;
      cleanup.trackDefinition(created.id);
      await expect(page).toHaveURL(/\/resource-definitions\/[^/]+$/);
      const definitionId = new URL(page.url()).pathname.split("/").at(-1);
      expect(definitionId).toBeTruthy();
      expect(definitionId).toBe(created.id);
      await expect(page.getByRole("heading", { name })).toBeVisible();
      const definition = await findDefinitionByKey(request, key);
      expect(definition.id).toBe(definitionId);
      await expect(page).toHaveURL(new RegExp(`/resource-definitions/${definition.id}$`));
    });
    await flowStep("resource-definition-management", 4, "abre o formulário de campo", async () => {
      await page.getByRole("button", { name: "Novo campo" }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Novo campo" }) });
      await expect(dialog.getByLabel("Key", { exact: true })).toBeVisible();
      await expect(dialog.getByLabel("Label", { exact: true })).toBeVisible();
    });
    await flowStep("resource-definition-management", 5, "adiciona campo host obrigatório", async () => {
      const dialog = page.getByRole("dialog");
      await dialog.getByLabel("Key", { exact: true }).fill("host");
      await dialog.getByLabel("Label", { exact: true }).fill("Host");
      await dialog.getByRole("checkbox", { name: "Obrigatório" }).check();
      await dialog.getByRole("button", { name: "Adicionar campo" }).click();
      const row = page.getByRole("row").filter({ has: page.getByRole("cell", { name: "host", exact: true }) });
      await expect(row).toContainText("Host");
      await expect(page.getByText("Campo adicionado.", { exact: true })).toBeVisible();
    });
    await flowStep("resource-definition-management", 6, "adiciona campo password secreto", async () => {
      await addFieldViaUI(page, { key: "password", label: "Password", required: true, secret: true });
      const row = page.getByRole("row").filter({ has: page.getByRole("cell", { name: "password", exact: true }) });
      await expect(row.getByRole("cell").nth(3).locator("svg")).toHaveCount(1);
      await expect(row.getByRole("cell").nth(4).locator("svg")).toHaveCount(1);
    });
    await flowStep("resource-definition-management", 7, "desativa a definição", async () => {
      await page.getByRole("button", { name: "Desativar" }).click();
      await expect(page.getByText("inativo", { exact: true })).toBeVisible();
      await expect(page.getByText("Definição desativada.", { exact: true })).toBeVisible();
      await expect(page.getByRole("button", { name: "Ativar" })).toBeVisible();
    });
    await flowStep("resource-definition-management", 8, "reativa a definição", async () => {
      await page.getByRole("button", { name: "Ativar" }).click();
      await expect(page.getByText("ativo", { exact: true })).toBeVisible();
      await expect(page.getByText("Definição ativada.", { exact: true })).toBeVisible();
    });
    await flowStep("resource-definition-management", 9, "remove campo com confirmação", async () => {
      const row = page.getByRole("row").filter({ has: page.getByRole("cell", { name: "password", exact: true }) });
      await row.getByRole("button", { name: "Remover" }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Remover campo?" }) });
      await expect(dialog).toContainText("password");
      await dialog.getByRole("button", { name: "Remover" }).click();
      await expect(page.getByRole("cell", { name: "password", exact: true })).toHaveCount(0);
      await expect(page.getByText("Campo removido.", { exact: true })).toBeVisible();
    });
    await flowStep("resource-definition-management", 10, "atualiza contagens no card", async () => {
      await page.getByRole("link", { name: "Definições de recurso", exact: true }).click();
      const card = page.getByRole("button").filter({ hasText: name });
      await expect(card).toContainText("Campos: 1");
      await expect(card).toContainText("Segredos: 0");
    });
  } finally {
    await cleanup.run(request);
  }
});
