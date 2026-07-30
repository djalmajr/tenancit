import { expect, test, type Locator, type Page } from "@playwright/test";
import {
  type APIClientRecord,
  CatalogCleanup,
  createDefinitionFixture,
  createTenantFixture,
} from "./fixtures/catalog";
import { adminToken, uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";

const mobileViewport = { height: 844, width: 390 };

test.use({ screenshot: "off", trace: "off", viewport: mobileViewport });

async function expectContainedInViewport(locator: Locator, page: Page) {
  await expect(locator).toBeVisible();
  await locator.scrollIntoViewIfNeeded();
  const box = await locator.boundingBox();
  const viewport = page.viewportSize();
  expect(box, "visible element has a bounding box").not.toBeNull();
  expect(viewport, "test has a fixed mobile viewport").not.toBeNull();
  if (!box || !viewport) return;
  expect(box.x).toBeGreaterThanOrEqual(0);
  expect(box.y).toBeGreaterThanOrEqual(0);
  expect(box.x + box.width).toBeLessThanOrEqual(viewport.width + 1);
  expect(box.y + box.height).toBeLessThanOrEqual(viewport.height + 1);
}

async function openMobileNavigation(page: Page) {
  await page.getByRole("button", { name: "Toggle sidebar" }).click();
  const sheet = page.getByRole("dialog");
  await expect(sheet).toBeVisible();
  return sheet;
}

async function tapMobileRoute(page: Page, name: string, heading: string) {
  const sheet = await openMobileNavigation(page);
  await sheet.getByRole("link", { name, exact: true }).click();
  await expect(sheet).toHaveCount(0);
  await expect(page.getByRole("heading", { name: heading, exact: true })).toBeVisible();
}

test("mobile navigation and dialogs fit a narrow viewport", { tag: "@full" }, async ({ context, page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();

  try {
    const tenant = await createTenantFixture(request, cleanup, "mobile-tenant");
    const { definition } = await createDefinitionFixture(request, cleanup, {
      prefix: "mobile-definition",
      fields: [
        { key: "host", label: "Host", dataType: "string", required: true },
        { key: "password", label: "Password", dataType: "string", required: true, isSecret: true },
      ],
    });
    const { definition: secondDefinition } = await createDefinitionFixture(request, cleanup, {
      prefix: "mobile-definition-second",
    });
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    await flowStep("responsive-mobile-navigation", 1, "autentica em viewport mobile", async () => {
      await page.goto("/");
      expect(page.viewportSize()).toEqual(mobileViewport);
      const authCard = page.getByRole("heading", { name: "Acesso administrativo" }).locator("xpath=ancestor::form");
      await expectContainedInViewport(authCard, page);

      const language = page.getByRole("button", { name: "Idioma" });
      await expectContainedInViewport(language, page);
      await language.click();
      await expect(page.getByRole("menuitemradio", { name: "Português" })).toBeVisible();
      await page.getByRole("menuitemradio", { name: "Português" }).click();

      const theme = page.getByRole("button", { name: /^Tema:/ });
      await expectContainedInViewport(theme, page);
      await theme.click();
      await expect(page.getByRole("menuitemradio", { name: "Sistema" })).toBeVisible();
      await page.getByRole("menuitemradio", { name: "Sistema" }).click();

      await page.getByRole("textbox", { name: "Token", exact: true }).fill(adminToken);
      await page.getByRole("button", { name: "Entrar" }).click();
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    });

    await flowStep("responsive-mobile-navigation", 2, "abre sidebar como sheet", async () => {
      await expect(page.locator("aside")).toHaveCount(0);
      const sheet = await openMobileNavigation(page);
      await expect(sheet).toHaveAttribute("aria-modal", "true");
      await expectContainedInViewport(sheet, page);
      await expect(sheet.getByRole("link", { name: "Tenants", exact: true })).toBeVisible();
    });

    await flowStep("responsive-mobile-navigation", 3, "fecha menu após cada navegação", async () => {
      let sheet = page.getByRole("dialog");
      await sheet.getByRole("link", { name: "Tenants", exact: true }).click();
      await expect(sheet).toHaveCount(0);
      await expect(page.getByRole("heading", { name: "Tenants", exact: true })).toBeVisible();

      sheet = await openMobileNavigation(page);
      await sheet.getByRole("link", { name: "Recursos", exact: true }).click();
      await expect(sheet).toHaveCount(0);
      await expect(page.getByRole("heading", { name: "Definições de recurso" })).toBeVisible();

      sheet = await openMobileNavigation(page);
      await sheet.getByRole("link", { name: "Chaves de API", exact: true }).click();
      await expect(sheet).toHaveCount(0);
      await expect(page.getByRole("heading", { name: "Chaves de API" })).toBeVisible();
    });

    await flowStep("responsive-mobile-navigation", 4, "mantém busca tabela e CTA utilizáveis", async () => {
      await tapMobileRoute(page, "Tenants", "Tenants");
      const search = page.getByRole("textbox", { name: /Buscar por nome ou slug/ });
      const createTenant = page.getByRole("button", { name: "Novo tenant" });
      await expectContainedInViewport(search, page);
      await expectContainedInViewport(createTenant, page);
      await search.fill(tenant.slug);
      await expect(page.getByRole("row").filter({ hasText: tenant.name })).toBeVisible();

      const scroller = page.getByRole("table").locator("..");
      await expect(scroller).toHaveCSS("overflow-x", "auto");
      expect(await scroller.evaluate((element) => element.scrollWidth >= element.clientWidth)).toBe(true);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);

      await createTenant.click();
      const dialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: "Novo tenant" }),
      });
      await expectContainedInViewport(dialog, page);
      await expect(dialog.getByLabel("Nome", { exact: true })).toBeVisible();
      await page.keyboard.press("Escape");
      await expect(dialog).toHaveCount(0);
    });

    await flowStep("responsive-mobile-navigation", 5, "empilha prontidão e contém o diálogo", async () => {
      await page.getByRole("row").filter({ hasText: tenant.name }).click();
      await expect(page.getByRole("heading", { name: tenant.name, exact: true })).toBeVisible();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);

      const readinessLabels = [
        "Prontidão para consumo",
        "Domínios",
        "Recursos ativos/total",
        "Configurações incompletas",
      ];
      const cardBoxes = [];
      for (const label of readinessLabels) {
        const card = page.getByRole("heading", { name: label, exact: true })
          .locator("xpath=ancestor::*[@data-slot='card'][1]");
        const box = await card.boundingBox();
        expect(box, `${label} has a visible readiness card`).not.toBeNull();
        if (box) cardBoxes.push(box);
      }
      expect(cardBoxes).toHaveLength(4);
      for (let index = 1; index < cardBoxes.length; index += 1) {
        expect(cardBoxes[index].y).toBeGreaterThan(cardBoxes[index - 1].y);
        expect(Math.abs(cardBoxes[index].x - cardBoxes[index - 1].x)).toBeLessThanOrEqual(1);
      }
      await expectContainedInViewport(page.getByRole("tab", { name: "Recursos", exact: true }), page);
      await expectContainedInViewport(page.getByRole("tab", { name: "Domínios", exact: true }), page);

      await page.getByRole("tab", { name: "Recursos", exact: true }).click();
      await page.getByRole("button", { name: "Adicionar recurso" }).click();
      const dialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: "Adicionar recurso" }),
      });
      await expectContainedInViewport(dialog, page);
      await dialog.getByRole("combobox").click();
      await page.getByRole("option", { name: definition.name, exact: true }).click();
      await dialog.getByPlaceholder("host", { exact: true }).fill(`db.${tenant.slug}.local`);
      await dialog.getByPlaceholder("password").fill("mobile-secret");
      await expect(dialog.getByRole("button", { name: "Salvar recurso" })).toBeEnabled();
      await dialog.getByRole("button", { name: "Cancelar" }).click();
      await expect(dialog).toHaveCount(0);
    });

    await flowStep("responsive-mobile-navigation", 6, "reflui definitions para uma coluna", async () => {
      await tapMobileRoute(page, "Recursos", "Definições de recurso");
      const firstCard = page.getByRole("link", {
        name: `Abrir definição ${definition.name}`,
        exact: true,
      });
      const secondCard = page.getByRole("link", {
        name: `Abrir definição ${secondDefinition.name}`,
        exact: true,
      });
      await expect(firstCard).toBeVisible();
      await expect(secondCard).toBeVisible();
      const grid = firstCard.locator("..");
      const columnCount = await grid.evaluate((element) =>
        getComputedStyle(element).gridTemplateColumns.split(/\s+/).filter(Boolean).length,
      );
      expect(columnCount).toBe(1);
      const firstBox = await firstCard.boundingBox();
      const secondBox = await secondCard.boundingBox();
      expect(firstBox).not.toBeNull();
      expect(secondBox).not.toBeNull();
      expect(Math.abs((secondBox?.y ?? 0) - (firstBox?.y ?? 0))).toBeGreaterThan(1);
      expect(Math.abs((secondBox?.x ?? Number.POSITIVE_INFINITY) - (firstBox?.x ?? 0))).toBeLessThanOrEqual(1);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    });

    await flowStep("responsive-mobile-navigation", 7, "gera e copia token sem corte", async () => {
      await tapMobileRoute(page, "Chaves de API", "Chaves de API");
      const clientName = uniqueID("mobile-client");
      await page.getByRole("button", { name: "Nova chave" }).click();
      let dialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: "Nova chave de API" }),
      });
      await expectContainedInViewport(dialog, page);
      await dialog.getByPlaceholder("billing-service").fill(clientName);
      const [response] = await Promise.all([
        page.waitForResponse((candidate) =>
          candidate.request().method() === "POST"
          && new URL(candidate.url()).pathname === "/v1/admin/api-clients"),
        dialog.getByRole("button", { name: "Gerar token" }).click(),
      ]);
      expect(response.ok(), "API client creation response").toBe(true);
      const created = await response.json() as { client: APIClientRecord; token: string };
      cleanup.trackAPIClient(created.client.id);

      dialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: "Token gerado" }),
      });
      await expectContainedInViewport(dialog, page);
      const tokenField = dialog.getByRole("textbox", { name: "Token" });
      expect((await tokenField.inputValue()) === created.token).toBe(true);
      const generatedToken = created.token;

      const tokenRegion = tokenField.locator("..");
      expect(await tokenRegion.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(true);
      const copy = tokenRegion.getByRole("button", { name: "Copiar" });
      await expectContainedInViewport(copy, page);
      await copy.click();
      await expect(tokenRegion.getByRole("button", { name: "Copiado" })).toBeVisible();
      await expect.poll(() => page.evaluate(
        async (expected) => (await navigator.clipboard.readText()) === expected,
        generatedToken,
      )).toBe(true);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);

      await dialog.getByRole("button", { name: "Concluir" }).click();
      await expect(dialog).toHaveCount(0);
    });
  } finally {
    await cleanup.run(request);
  }
});
