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
    const updatedHost = `db-updated.${tenant.slug}.local`;
    const resourceAlias = `${definition.key}.primary`;

    await flowStep("tenant-resource-lifecycle", 1, "abre tenants autenticado", async () => {
      await loginViaUI(page);
      await navigateFromSidebar(page, "Tenants");
      await expect(page.getByRole("row").filter({ hasText: tenant.name })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 2, "abre o detalhe na aba recursos", async () => {
      await page.getByRole("row").filter({ hasText: tenant.name }).click();
      await expect(page.getByRole("heading", { name: tenant.name })).toBeVisible();
      await page.getByRole("tab", { name: "Recursos", exact: true }).click();
      await expect(page.getByRole("cell", { name: "Nenhum recurso", exact: true })).toBeVisible();
      await expect(page.getByRole("textbox", { name: "Buscar por nome, tipo ou status..." })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 3, "abre os tipos disponíveis", async () => {
      await page.getByRole("button", { name: "Adicionar recurso" }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Adicionar recurso" }) });
      await expect(dialog.getByRole("combobox")).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 4, "materializa identidade e campos da definição", async () => {
      const dialog = page.getByRole("dialog");
      await dialog.getByRole("combobox").click();
      await page.getByRole("option", { name: definition.name }).click();
      await dialog.getByPlaceholder("postgres.agility").fill(resourceAlias);
      await expect(dialog.getByPlaceholder("host", { exact: true })).toHaveAttribute("type", "text");
      await expect(dialog.getByPlaceholder("password")).toHaveAttribute("type", "password");
      const viewport = page.locator("[data-slot='dialog-viewport']");
      const formGrid = dialog.locator("[data-slot='resource-form-grid']");
      await expect(viewport).toHaveCSS("overflow-y", "auto");
      // Mutation captured: removing the responsive two-column grid returns the
      // desktop dialog to a narrow single-column stack.
      await expect(formGrid).toHaveCSS("display", "grid");
      expect(await formGrid.evaluate((element) =>
        getComputedStyle(element).gridTemplateColumns.split(/\s+/).filter(Boolean).length,
      )).toBe(2);
      for (const slot of ["resource-type-field", "resource-origin-field"]) {
        expect(await dialog.locator(`[data-slot='${slot}']`).evaluate((element) =>
          getComputedStyle(element).gridColumnEnd,
        )).toBe("span 2");
      }
      const nameBox = await dialog.locator("[data-slot='resource-name-field']").boundingBox();
      const aliasBox = await dialog.locator("[data-slot='resource-alias-field']").boundingBox();
      const valueBoxes = await dialog.locator("[data-slot='resource-value-field']").evaluateAll((elements) =>
        elements.map((element) => {
          const rect = element.getBoundingClientRect();
          return { x: rect.x, y: rect.y };
        }),
      );
      expect(nameBox?.y ?? -1).toBeCloseTo(aliasBox?.y ?? -2, 0);
      expect(nameBox?.x ?? 0).toBeLessThan(aliasBox?.x ?? 0);
      expect(valueBoxes).toHaveLength(2);
      expect(valueBoxes[0]?.y ?? -1).toBeCloseTo(valueBoxes[1]?.y ?? -2, 0);
      expect(valueBoxes[0]?.x ?? 0).toBeLessThan(valueBoxes[1]?.x ?? 0);
      expect((await dialog.boundingBox())?.width ?? 0).toBeGreaterThan(600);
    });
    await flowStep("tenant-resource-lifecycle", 5, "salva recurso obrigatório e secreto", async () => {
      const dialog = page.getByRole("dialog");
      await dialog.getByPlaceholder("host", { exact: true }).fill(`db.${tenant.slug}.local`);
      await dialog.getByPlaceholder("password").fill(clearSecret);
      await dialog.getByRole("button", { name: "Salvar recurso" }).click();
      await expect(page.getByText("Recurso adicionado.", { exact: true })).toBeVisible();
      const resourceRow = page.getByRole("row").filter({ hasText: definition.name });
      await expect(resourceRow.getByRole("cell", { name: resourceAlias, exact: true })).toBeVisible();
      await expect(resourceRow.getByRole("cell", { name: "ativo", exact: true })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 6, "abre o detalhe com segredo mascarado", async () => {
      const resourceRow = page.getByRole("row").filter({
        has: page.getByRole("cell", { name: resourceAlias, exact: true }),
      });
      await resourceRow.click();
      const resourceDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: definition.name, exact: true }),
      });
      await expect(resourceDialog).toContainText(resourceAlias);
      await expect(resourceDialog.getByText(SECRET_MASK, { exact: true })).toBeVisible();
      expect(await page.locator("body").evaluate(
        (body, sensitive) => body.textContent?.includes(sensitive) ?? false,
        clearSecret,
      )).toBe(false);
    });
    await flowStep("tenant-resource-lifecycle", 7, "edita um campo não secreto", async () => {
      const resourceDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: definition.name, exact: true }),
      });
      const hostRow = resourceDialog.getByRole("row").filter({
        has: page.getByRole("cell", { name: "Host", exact: true }),
      });
      await hostRow.getByRole("button", { name: "Editar", exact: true }).click();
      const editDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: "Editar campo", exact: true }),
      });
      await editDialog.getByLabel("Host", { exact: true }).fill(updatedHost);
      await editDialog.getByRole("button", { name: "Salvar", exact: true }).click();
      await expect(page.getByText("Campo atualizado.", { exact: true })).toBeVisible();
      await expect(hostRow.getByText(updatedHost, { exact: true })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 8, "revela e volta a ocultar após gesto explícito", async () => {
      const resourceDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: definition.name, exact: true }),
      });
      const passwordRow = resourceDialog.getByRole("row").filter({
        has: page.getByRole("cell", { name: "Password", exact: true }),
      });
      await resourceDialog.getByRole("button", { name: "Revelar", exact: true }).click();
      await expect(passwordRow.getByText(clearSecret, { exact: true })).toBeVisible();
      await resourceDialog.getByRole("button", { name: "Ocultar", exact: true }).click();
      await expect(passwordRow.getByText(SECRET_MASK, { exact: true })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 9, "desativa o recurso", async () => {
      const resourceDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: definition.name, exact: true }),
      });
      await resourceDialog.getByRole("button", { name: "Desativar", exact: true }).click();
      await expect(resourceDialog.getByText("inativo", { exact: true })).toBeVisible();
      await expect(page.getByText("Recurso desativado.", { exact: true })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 10, "reabre e reativa o recurso", async () => {
      let resourceDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: definition.name, exact: true }),
      });
      await resourceDialog.getByRole("button", { name: "Fechar", exact: true }).click();
      const resourceRow = page.getByRole("row").filter({
        has: page.getByRole("cell", { name: resourceAlias, exact: true }),
      });
      await resourceRow.click();
      resourceDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: definition.name, exact: true }),
      });
      await resourceDialog.getByRole("button", { name: "Reativar", exact: true }).click();
      await expect(resourceDialog.getByText("ativo", { exact: true })).toBeVisible();
      await expect(page.getByText("Recurso reativado.", { exact: true })).toBeVisible();
    });
    await flowStep("tenant-resource-lifecycle", 11, "remove o recurso com confirmação", async () => {
      const resourceDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: definition.name, exact: true }),
      });
      await resourceDialog.getByRole("button", { name: "Remover", exact: true }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Remover recurso?" }) });
      await expect(dialog).toContainText(resourceAlias);
      await dialog.getByRole("button", { name: "Remover" }).click();
      await expect(page.getByRole("cell", { name: "Nenhum recurso", exact: true })).toBeVisible();
      await expect(page.getByText("Recurso removido.", { exact: true })).toBeVisible();
    });
  } finally {
    await cleanup.run(request);
  }
});
