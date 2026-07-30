import { expect, test } from "@playwright/test";
import { adminToken } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";
import { chooseLocale, chooseTheme, logoutViaUI, navigateFromSidebar } from "./support/ui";

test("operator authenticates, reviews overview, navigates, and logs out", { tag: "@pr-critical" }, async ({ page }) => {
  test.slow();

  try {
    await flowStep("admin-auth-overview", 1, "bloqueia o painel sem token", async () => {
      await page.goto("/");
      await expect(page.getByRole("heading", { name: "Acesso administrativo" })).toBeVisible();
      await expect(page.getByRole("link", { name: "Tenants", exact: true })).toHaveCount(0);
    });
    await flowStep("admin-auth-overview", 2, "alterna os três idiomas no login", async () => {
      await chooseLocale(page, "English");
      await expect(page.getByRole("heading", { name: "Administrative access" })).toBeVisible();
      await chooseLocale(page, "Español");
      await expect(page.getByRole("heading", { name: "Acceso administrativo" })).toBeVisible();
      await chooseLocale(page, "Português");
      await expect(page.getByRole("heading", { name: "Acesso administrativo" })).toBeVisible();
    });
    await flowStep("admin-auth-overview", 3, "alterna os três temas no login", async () => {
      await chooseTheme(page, "Escuro");
      await expect(page.locator("html")).toHaveClass(/dark/);
      await chooseTheme(page, "Claro");
      await expect(page.locator("html")).not.toHaveClass(/dark/);
      await chooseTheme(page, "Sistema");
      await expect.poll(() => page.evaluate(() => localStorage.getItem("tenancitTheme"))).toBe("system");
    });
    await flowStep("admin-auth-overview", 4, "entra e preserva preferências", async () => {
      await page.getByRole("textbox", { name: "Token", exact: true }).fill(adminToken);
      await page.getByRole("button", { name: "Entrar" }).click();
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
      await expect.poll(() => page.evaluate(() => localStorage.getItem("tenancitLocale"))).toBe("pt-BR");
      await expect.poll(() => page.evaluate(() => localStorage.getItem("tenancitTheme"))).toBe("system");
    });
    await flowStep("admin-auth-overview", 5, "exibe os quatro KPIs operacionais", async () => {
      for (const label of ["Tenants ativos", "Domínios", "Recursos provisionados", "Definições ativas"]) {
        await expect(page.getByText(label, { exact: true })).toBeVisible();
      }
    });
    await flowStep("admin-auth-overview", 6, "resume o pulso operacional no overview", async () => {
      for (const label of ["Saúde operacional", "Requisições no mês", "Chaves expirando", "Dead letters"]) {
        await expect(page.getByText(label, { exact: true })).toBeVisible();
      }
    });
    await flowStep("admin-auth-overview", 7, "traduz o shell autenticado", async () => {
      await chooseLocale(page, "English");
      await expect(page.getByRole("link", { name: "Overview", exact: true })).toBeVisible();
      await chooseLocale(page, "Español");
      await expect(page.getByRole("link", { name: "Recursos", exact: true })).toBeVisible();
      await chooseLocale(page, "Português");
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    });
    await flowStep("admin-auth-overview", 8, "aplica tema no shell sem perder conteúdo", async () => {
      await chooseTheme(page, "Escuro");
      await expect(page.locator("html")).toHaveClass(/dark/);
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
      await chooseTheme(page, "Claro");
      await expect(page.locator("html")).not.toHaveClass(/dark/);
      await chooseTheme(page, "Sistema");
    });
    await flowStep("admin-auth-overview", 9, "navega por todas as seções", async () => {
      await navigateFromSidebar(page, "Tenants");
      await navigateFromSidebar(page, "Recursos");
      await navigateFromSidebar(page, "Chaves de API");
      await navigateFromSidebar(page, "Visão geral");
      await expect(page.locator("header").getByText("Visão geral", { exact: true })).toBeVisible();
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    });
    await flowStep("admin-auth-overview", 10, "mantém navegação acessível recolhida", async () => {
      const sidebar = page.locator("aside");
      await page.getByRole("button", { name: "Toggle sidebar" }).click();
      await expect(sidebar).toHaveAttribute("data-state", "collapsed");
      await expect(page.getByRole("link", { name: "Tenants", exact: true })).toBeVisible();
      await page.getByRole("button", { name: "Toggle sidebar" }).click();
      await expect(sidebar).toHaveAttribute("data-state", "expanded");
    });
    await flowStep("admin-auth-overview", 11, "remove a credencial ao sair", async () => {
      await logoutViaUI(page);
      await expect.poll(() => page.evaluate(() => localStorage.getItem("tenancitAdminToken"))).toBeNull();
    });
  } finally {
    await page.close();
  }
});
