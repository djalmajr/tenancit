import { expect, test, type Locator, type Page } from "@playwright/test";
import {
  type APIClientRecord,
  CatalogCleanup,
  createDefinitionFixture,
  createResourceFixture,
  createTenantFixture,
} from "./fixtures/catalog";
import { adminToken, uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";

test.use({ screenshot: "off", trace: "off" });

async function tabTo(
  page: Page,
  target: Locator,
  maximumTabs = 80,
  key: "Shift+Tab" | "Tab" = "Tab",
) {
  await expect(target).toBeVisible();
  for (let tabCount = 0; tabCount <= maximumTabs; tabCount += 1) {
    if (await target.evaluate((element) => element === document.activeElement)) return;
    await page.keyboard.press(key);
  }
  throw new Error(`Could not reach ${await target.getAttribute("aria-label") ?? await target.textContent()} with Tab`);
}

async function expectKeyboardFocus(target: Locator) {
  await expect(target).toBeFocused();
  await expect.poll(() => target.evaluate((element) => element.matches(":focus-visible"))).toBe(true);
}

test("core administration remains operable with a real keyboard", { tag: "@full" }, async ({ context, page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();

  try {
    const tenant = await createTenantFixture(request, cleanup, "keyboard-tenant");
    const clearSecret = `keyboard-secret-${tenant.slug}`;
    const { definition } = await createDefinitionFixture(request, cleanup, {
      prefix: "keyboard-definition",
      fields: [{ key: "password", label: "Password", required: true, isSecret: true }],
    });
    await createResourceFixture(request, tenant.id, definition.key, { password: clearSecret });
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    await flowStep("keyboard-accessibility-core", 1, "entra usando foco, toggle e Enter", async () => {
      await page.goto("/");
      const tokenInput = page.getByRole("textbox", { name: "Token", exact: true });
      await expect(tokenInput).toBeFocused();
      await tokenInput.fill(adminToken);

      await page.keyboard.press("Tab");
      const showToken = page.getByRole("button", { name: "Mostrar token" });
      await expectKeyboardFocus(showToken);
      await page.keyboard.press("Enter");
      await expect(tokenInput).toHaveAttribute("type", "text");
      await expect(page.getByRole("button", { name: "Ocultar token" })).toBeFocused();

      await page.keyboard.press("Shift+Tab");
      await expectKeyboardFocus(tokenInput);
      await page.keyboard.press("Enter");
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    });

    await flowStep("keyboard-accessibility-core", 2, "navega e opera menus pelo teclado", async () => {
      const navigationLabels = ["Visão geral", "Tenants", "Recursos", "Chaves de API"] as const;
      for (const label of navigationLabels) {
        const link = page.getByRole("link", { name: label, exact: true });
        await expect(link).toHaveAttribute("aria-label", label);
        await tabTo(page, link);
        await expectKeyboardFocus(link);
      }

      const tenantsLink = page.getByRole("link", { name: "Tenants", exact: true });
      await tabTo(page, tenantsLink);
      await expectKeyboardFocus(tenantsLink);
      await page.keyboard.press("Enter");
      await expect(page.getByRole("heading", { name: "Tenants", exact: true })).toBeVisible();

      const languageButton = page.getByRole("button", { name: "Idioma" });
      await tabTo(page, languageButton);
      await expectKeyboardFocus(languageButton);
      await page.keyboard.press("Enter");
      const englishOption = page.getByRole("menuitemradio", { name: "English" });
      await expect(englishOption).toBeVisible();
      await englishOption.press("Enter");
      await expect(page.getByRole("button", { name: "Language" })).toBeVisible();

      const themeButton = page.getByRole("button", { name: /^Theme:/ });
      await tabTo(page, themeButton);
      await expectKeyboardFocus(themeButton);
      await page.keyboard.press("Enter");
      const darkOption = page.getByRole("menuitemradio", { name: "Dark" });
      await expect(darkOption).toBeVisible();
      await darkOption.press("Enter");
      await expect(page.locator("html")).toHaveClass(/\bdark\b/);

      const translatedLanguageButton = page.getByRole("button", { name: "Language" });
      await tabTo(page, translatedLanguageButton);
      await page.keyboard.press("Enter");
      await page.getByRole("menuitemradio", { name: "Português" }).press("Enter");
      await expect(page.getByRole("button", { name: "Idioma" })).toBeVisible();
    });

    await flowStep("keyboard-accessibility-core", 3, "prende e restaura foco no novo tenant", async () => {
      const trigger = page.getByRole("button", { name: "Novo tenant" });
      await tabTo(page, trigger);
      await expectKeyboardFocus(trigger);
      await page.keyboard.press("Enter");

      const dialog = page.getByRole("dialog");
      const nameInput = dialog.getByLabel("Nome", { exact: true });
      const slugInput = dialog.getByLabel("Slug", { exact: true });
      const cancel = dialog.getByRole("button", { name: "Cancelar" });
      await expect(nameInput).toBeFocused();
      await page.keyboard.press("Tab");
      await expect(slugInput).toBeFocused();
      await page.keyboard.press("Tab");
      await expect(cancel).toBeFocused();
      await page.keyboard.press("Tab");
      await expect(nameInput).toBeFocused();
      await page.keyboard.press("Shift+Tab");
      await expect(cancel).toBeFocused();

      await page.keyboard.press("Escape");
      await expect(dialog).toHaveCount(0);
      await expect(trigger).toBeFocused();
    });

    await flowStep("keyboard-accessibility-core", 4, "alcança revelação e ações destrutivas nomeadas", async () => {
      await page.goto(`/tenants/${tenant.id}`);
      await expect(page.getByRole("heading", { name: tenant.name, exact: true })).toBeVisible();

      const overviewTab = page.getByRole("tab", { name: "Visão geral", exact: true });
      const resourcesTab = page.getByRole("tab", { name: "Recursos", exact: true });
      await tabTo(page, overviewTab);
      await expectKeyboardFocus(overviewTab);
      await page.keyboard.press("ArrowRight");
      await expectKeyboardFocus(resourcesTab);
      await page.keyboard.press("Enter");
      await expect(resourcesTab).toHaveAttribute("aria-selected", "true");

      const resourceRow = page.getByRole("row").filter({ hasText: definition.key });
      await tabTo(page, resourceRow);
      await expectKeyboardFocus(resourceRow);
      // Mutation captured: removing the row keyboard handler prevents Enter from opening resource details.
      await page.keyboard.press("Enter");
      const resourceDialog = page.getByRole("dialog").filter({ hasText: definition.key });
      await expect(resourceDialog).toBeVisible();

      const revealSecret = resourceDialog.getByRole("button", { name: "Revelar", exact: true });
      await tabTo(page, revealSecret);
      await expectKeyboardFocus(revealSecret);
      await page.keyboard.press("Enter");
      const hideSecret = resourceDialog.getByRole("button", { name: "Ocultar", exact: true });
      await tabTo(page, hideSecret);
      await expectKeyboardFocus(hideSecret);
      expect(await page.locator("body").evaluate(
        (body, sensitive) => body.textContent?.includes(sensitive) ?? false,
        clearSecret,
      )).toBe(true);

      const deactivate = resourceDialog.getByRole("button", { name: "Desativar", exact: true });
      const remove = resourceDialog.getByRole("button", { name: "Remover", exact: true });
      await tabTo(page, deactivate, 10, "Shift+Tab");
      await expectKeyboardFocus(deactivate);
      await expect(deactivate).toHaveAccessibleName("Desativar");

      await page.keyboard.press("Tab");
      await expectKeyboardFocus(remove);
      await expect(remove).toHaveAccessibleName("Remover");
      await page.keyboard.press("Enter");
      let confirmDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: "Remover recurso?" }),
      });
      await expect(confirmDialog.getByRole("button", { name: "Cancelar" })).toBeFocused();
      await page.keyboard.press("Escape");
      await expect(confirmDialog).toHaveCount(0);
      await expect(resourceDialog).toHaveCount(0);
      await tabTo(page, resourceRow);
      await expectKeyboardFocus(resourceRow);

      await page.keyboard.press("Enter");
      await expect(resourceDialog).toBeVisible();
      const reopenedRemove = resourceDialog.getByRole("button", { name: "Remover", exact: true });
      await tabTo(page, reopenedRemove);
      await expectKeyboardFocus(reopenedRemove);
      await page.keyboard.press("Enter");
      confirmDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: "Remover recurso?" }),
      });
      const cancel = confirmDialog.getByRole("button", { name: "Cancelar" });
      await expect(cancel).toBeFocused();
      await page.keyboard.press("Enter");
      await expect(confirmDialog).toHaveCount(0);
      await expect(resourceDialog).toHaveCount(0);
      await tabTo(page, resourceRow);
      await expectKeyboardFocus(resourceRow);

      await page.keyboard.press("Enter");
      await expect(resourceDialog).toBeVisible();
      const close = resourceDialog.getByRole("button", { name: "Fechar", exact: true });
      await tabTo(page, close, 10);
      await expectKeyboardFocus(close);
      await page.keyboard.press("Enter");
      await expect(resourceDialog).toHaveCount(0);
      await expect(resourceRow).toBeFocused();
    });

    await flowStep("keyboard-accessibility-core", 5, "gera e copia token pelo teclado", async () => {
      const apiClientsLink = page.getByRole("link", { name: "Chaves de API", exact: true });
      // The focused resource row is after the sidebar in DOM order.
      await tabTo(page, apiClientsLink, 30, "Shift+Tab");
      await expectKeyboardFocus(apiClientsLink);
      await page.keyboard.press("Enter");
      await expect(page.getByRole("heading", { name: "Chaves de API" })).toBeVisible();

      const createTrigger = page.getByRole("button", { name: "Nova chave" });
      await tabTo(page, createTrigger);
      await expectKeyboardFocus(createTrigger);
      await page.keyboard.press("Enter");
      let dialog = page.getByRole("dialog");
      const nameInput = dialog.getByPlaceholder("billing-service");
      await expect(nameInput).toBeFocused();
      const clientName = uniqueID("keyboard-client");
      await page.keyboard.type(clientName);
      const generateToken = dialog.getByRole("button", { name: "Gerar token" });
      await tabTo(page, generateToken, 10);
      await expectKeyboardFocus(generateToken);
      const [response] = await Promise.all([
        page.waitForResponse((candidate) =>
          candidate.request().method() === "POST"
          && new URL(candidate.url()).pathname === "/v1/admin/api-clients"),
        page.keyboard.press("Enter"),
      ]);
      expect(response.ok(), "API client creation response").toBe(true);
      const created = await response.json() as { client: APIClientRecord; token: string };
      cleanup.trackAPIClient(created.client.id);

      dialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: "Token gerado" }),
      });
      const tokenField = dialog.getByRole("textbox", { name: "Token" });
      expect((await tokenField.inputValue()) === created.token).toBe(true);
      const generatedToken = created.token;

      const tokenRegion = tokenField.locator("..");
      await tabTo(page, tokenField, 5);
      await expectKeyboardFocus(tokenField);
      await expect(tokenField).toHaveAccessibleName("Token");
      const copy = tokenRegion.getByRole("button", { name: "Copiar" });
      await tabTo(page, copy, 5);
      await expectKeyboardFocus(copy);
      await expect(copy).toHaveAccessibleName("Copiar");
      await page.keyboard.press("Enter");
      await expect(tokenRegion.getByRole("button", { name: "Copiado" })).toBeVisible();
      await expect.poll(() => page.evaluate(
        async (expected) => (await navigator.clipboard.readText()) === expected,
        generatedToken,
      )).toBe(true);

      const done = dialog.getByRole("button", { name: "Concluir" });
      await tabTo(page, done, 5);
      await page.keyboard.press("Enter");
      await expect(dialog).toHaveCount(0);
    });
  } finally {
    await cleanup.run(request);
  }
});
