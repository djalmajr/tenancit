import { expect, test } from "@playwright/test";
import {
  CatalogCleanup,
  createAPIClientFixture,
  createDefinitionFixture,
  createTenantFixture,
} from "./fixtures/catalog";
import { flowStep } from "./support/flow-step";
import { chooseLocale, chooseTheme, loginViaUI } from "./support/ui";

const portugueseRouteText =
  /Visão geral|Novo tenant|Definições de recurso|Nova definição|Prontidão para consumo|Domínios|Novo campo|Chaves de API|Nova chave|Linhas por página|Buscar por nome/;

test.use({ screenshot: "off", trace: "off" });

test("locale and theme cover every route and survive reload and logout", { tag: "@full" }, async ({ page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();

  try {
    const tenant = await createTenantFixture(request, cleanup, "i18n-tenant");
    const { definition } = await createDefinitionFixture(request, cleanup, { prefix: "i18n-definition" });
    await createAPIClientFixture(request, cleanup, "i18n-client");

    await flowStep("i18n-and-preferences-persistence", 1, "autentica em português", async () => {
      await loginViaUI(page);
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    });
    await flowStep("i18n-and-preferences-persistence", 2, "traduz overview e navegação para inglês", async () => {
      await chooseLocale(page, "English");
      await expect(page.locator("html")).toHaveAttribute("lang", "en-US");
      await expect(page.getByRole("heading", { name: "Overview" })).toBeVisible();
      await expect(page.getByText("Active tenants", { exact: true })).toBeVisible();
      await expect(page.getByRole("link", { name: "Resources", exact: true })).toBeVisible();
      await expect(page.getByRole("link", { name: "API Keys", exact: true })).toBeVisible();
      await expect(page.getByRole("link", { name: "Monthly usage", exact: true })).toBeVisible();
      await expect(page.getByRole("link", { name: "Audit", exact: true })).toBeVisible();
      await page.getByRole("button", { name: "Account" }).click();
      await expect(page.getByRole("menuitem", { name: "Log out" })).toBeVisible();
      await page.keyboard.press("Escape");
    });
    await flowStep("i18n-and-preferences-persistence", 3, "mantém todas as telas em inglês", async () => {
      await page.getByRole("link", { name: "Tenants", exact: true }).click();
      await expect(page.getByRole("heading", { name: "Tenants", exact: true })).toBeVisible();
      await expect(page.getByRole("button", { name: "New tenant" })).toBeVisible();
      await expect(page.getByRole("textbox", { name: /Search by name or slug/ })).toBeVisible();
      await expect(page.locator("main")).not.toContainText(portugueseRouteText);

      await page.getByRole("row").filter({ hasText: tenant.name }).click();
      await expect(page.getByRole("heading", { name: tenant.name })).toBeVisible();
      await expect(page.getByText("Consumption readiness", { exact: true })).toBeVisible();
      await expect(page.getByRole("tab", { name: "Domains", exact: true })).toBeVisible();
      await expect(page.getByRole("tab", { name: "Resources", exact: true })).toBeVisible();
      await expect(page.getByRole("button", { name: "Edit", exact: true })).toBeVisible();
      await expect(page.locator("main")).not.toContainText(portugueseRouteText);

      await page.getByRole("link", { name: "Monthly usage", exact: true }).click();
      await expect(page.getByRole("heading", { name: "API key usage" })).toBeVisible();
      await expect(page.locator("main")).not.toContainText(portugueseRouteText);

      await page.getByRole("link", { name: "Audit", exact: true }).click();
      await expect(page.getByRole("heading", { name: "Administrative audit" })).toBeVisible();
      await expect(page.locator("main")).not.toContainText(portugueseRouteText);

      await page.getByRole("link", { name: "Resources", exact: true }).click();
      await expect(page.getByRole("heading", { name: "Resource Definitions" })).toBeVisible();
      await expect(page.getByRole("button", { name: "New definition" }).first()).toBeVisible();
      const definitionCard = page.getByRole("link", {
        name: `Open definition ${definition.name}`,
        exact: true,
      });
      await expect(definitionCard).toBeVisible();
      await expect(page.locator("main")).not.toContainText(portugueseRouteText);

      await definitionCard.click();
      await expect(page.getByRole("heading", { name: definition.name })).toBeVisible();
      await expect(page.getByText("Fields", { exact: true })).toBeVisible();
      await expect(page.getByRole("button", { name: "New field" })).toBeVisible();
      await page.getByRole("button", { name: "Actions" }).click();
      await expect(page.getByRole("menuitem", { name: "Deactivate" })).toBeVisible();
      await page.keyboard.press("Escape");
      await expect(page.locator("main")).not.toContainText(portugueseRouteText);

      await page.getByRole("link", { name: "API Keys", exact: true }).click();
      await expect(page.getByRole("heading", { name: "API Keys" })).toBeVisible();
      await expect(page.getByRole("button", { name: "New API key" })).toBeVisible();
      await expect(page.getByRole("textbox", {
        name: "Search by name, date, or status...",
        exact: true,
      })).toBeVisible();
      await expect(page.locator("main")).not.toContainText(portugueseRouteText);
    });
    await flowStep("i18n-and-preferences-persistence", 4, "traduz superfícies principais para espanhol", async () => {
      await chooseLocale(page, "Español");
      await expect(page.locator("html")).toHaveAttribute("lang", "es-ES");
      await expect(page.getByRole("heading", { name: "Claves de API" })).toBeVisible();
      await expect(page.getByRole("button", { name: "Nueva clave" })).toBeVisible();
      await expect(page.getByRole("textbox", {
        name: "Buscar por nombre, fecha o estado...",
        exact: true,
      })).toBeVisible();

      await page.getByRole("link", { name: "Vista general", exact: true }).click();
      await expect(page.getByRole("heading", { name: "Vista general" })).toBeVisible();
      await expect(page.getByText("Tenants activos", { exact: true })).toBeVisible();

      await page.getByRole("link", { name: "Tenants", exact: true }).click();
      await expect(page.getByRole("heading", { name: "Tenants", exact: true })).toBeVisible();
      await expect(page.getByRole("button", { name: "Nuevo tenant" })).toBeVisible();
      await expect(page.getByRole("textbox", { name: /Buscar por nombre o slug/ })).toBeVisible();
      await expect(page.locator("main")).not.toContainText(/Visão geral|Novo tenant|Chaves de API|Nova chave/);
    });
    await flowStep("i18n-and-preferences-persistence", 5, "aplica tema escuro e sistema", async () => {
      await chooseTheme(page, "Oscuro");
      await expect(page.locator("html")).toHaveClass(/dark/);
      await expect.poll(() => page.evaluate(() => localStorage.getItem("tenancitTheme"))).toBe("dark");
      const palette = await page.evaluate(() => {
        const style = getComputedStyle(document.body);
        return { background: style.backgroundColor, foreground: style.color };
      });
      expect(palette.background).not.toBe("rgba(0, 0, 0, 0)");
      expect(palette.foreground).not.toBe(palette.background);
      await expect(page.getByRole("heading", { name: "Tenants" })).toBeVisible();

      await page.emulateMedia({ colorScheme: "dark" });
      await chooseTheme(page, "Sistema");
      await expect.poll(() => page.evaluate(() => localStorage.getItem("tenancitTheme"))).toBe("system");
      await expect(page.locator("html")).toHaveClass(/dark/);
    });
    await flowStep("i18n-and-preferences-persistence", 6, "persiste preferências após refresh", async () => {
      await page.reload();
      await expect(page.getByRole("button", { name: "Nuevo tenant" })).toBeVisible();
      await expect(page.locator("html")).toHaveAttribute("lang", "es-ES");
      await expect(page.locator("html")).toHaveClass(/dark/);
      await expect(page.getByRole("button", { name: "Tema: Sistema" })).toBeVisible();
      await expect.poll(() => page.evaluate(() => localStorage.getItem("tenancitLocale"))).toBe("es-ES");
      await expect.poll(() => page.evaluate(() => localStorage.getItem("tenancitTheme"))).toBe("system");
    });
    await flowStep("i18n-and-preferences-persistence", 7, "preserva preferências ao sair", async () => {
      await page.getByRole("button", { name: "Cuenta" }).click();
      await page.getByRole("menuitem", { name: "Salir" }).click();
      await expect(page.getByRole("heading", { name: "Acceso administrativo" })).toBeVisible();
      await expect(page.getByRole("button", { name: "Tema: Sistema" })).toBeVisible();
      await expect(page.locator("html")).toHaveAttribute("lang", "es-ES");
      await expect(page.locator("html")).toHaveClass(/dark/);
      await page.getByRole("button", { name: "Idioma" }).click();
      await expect(page.getByRole("menuitemradio", { name: "Español" })).toHaveAttribute("aria-checked", "true");
      await expect.poll(() => page.evaluate(() => localStorage.getItem("tenancitLocale"))).toBe("es-ES");
      await expect.poll(() => page.evaluate(() => localStorage.getItem("tenancitTheme"))).toBe("system");
    });
  } finally {
    await cleanup.run(request);
  }
});
