import { expect, test } from "@playwright/test";
import { CatalogCleanup } from "./fixtures/catalog";
import { uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";
import { createTenantViaUI, loginViaUI, navigateFromSidebar } from "./support/ui";

test("first run explains every empty state and next action", { tag: "@pr-critical" }, async ({ page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();
  const slug = uniqueID("first-run");
  const name = `First run ${slug}`;

  try {
    await flowStep("first-run-empty-states", 1, "autentica em banco vazio", async () => {
      await loginViaUI(page);
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    });
    await flowStep("first-run-empty-states", 2, "mostra todos os KPIs zerados", async () => {
      for (const label of ["Tenants ativos", "Domínios", "Recursos provisionados", "Definições ativas"]) {
        const card = page.getByText(label, { exact: true }).locator("../..");
        await expect(card.getByText("0", { exact: true })).toBeVisible();
      }
    });
    await flowStep("first-run-empty-states", 3, "orienta no card vazio de tenants", async () => {
      await expect(page.getByText("Nenhum tenant ainda.", { exact: true })).toBeVisible();
    });
    await flowStep("first-run-empty-states", 4, "mantém CTA na tabela vazia de tenants", async () => {
      await navigateFromSidebar(page, "Tenants");
      await expect(page.getByRole("cell", { name: "Nenhum tenant encontrado." })).toBeVisible();
      await expect(page.getByRole("button", { name: "Novo tenant" })).toBeEnabled();
    });
    await flowStep("first-run-empty-states", 5, "orienta na grade vazia de definições", async () => {
      await navigateFromSidebar(page, "Recursos");
      await expect(page.getByText("Nenhuma definição ainda.", { exact: true })).toBeVisible();
      await expect(page.getByRole("button", { name: "Nova definição" }).first()).toBeEnabled();
    });
    await flowStep("first-run-empty-states", 6, "explica a lista vazia de chaves", async () => {
      await navigateFromSidebar(page, "Chaves de API");
      await expect(page.getByRole("cell", { name: "Nenhuma chave ativa." })).toBeVisible();
      await page.getByRole("button", { name: "Ajuda" }).click();
      await expect(page.getByText("Acesso a segredos em claro", { exact: true })).toBeVisible();
      await expect(page.getByText(/\/v1\/resolve/).first()).toBeVisible();
      await page.keyboard.press("Escape");
    });
    await flowStep("first-run-empty-states", 7, "cria o primeiro tenant pela UI", async () => {
      const tenant = await createTenantViaUI(page, cleanup, name, slug);
      await expect(page).toHaveURL(new RegExp(`/tenants/${tenant.id}$`));
    });
    await flowStep("first-run-empty-states", 8, "mostra prontidão e pendências", async () => {
      const readiness = page.getByText("Prontidão para consumo", { exact: true }).locator("../..");
      await expect(readiness.getByText("ativo", { exact: true })).toBeVisible();
      await expect(readiness.getByText("adicione um domínio", { exact: true })).toBeVisible();
      await expect(readiness.getByText("adicione um recurso ativo", { exact: true })).toBeVisible();
      await expect(readiness.getByText("crie uma chave em Chaves de API", { exact: true })).toBeVisible();
    });
    await flowStep("first-run-empty-states", 9, "explica recursos e domínios vazios", async () => {
      await expect(page.getByText("Nenhum recurso", { exact: true })).toBeVisible();
      await page.getByRole("tab", { name: "Domínios", exact: true }).click();
      await expect(page.getByText("Nenhum domínio. Adicione ao menos um para resolver o tenant.", { exact: true })).toBeVisible();
    });
  } finally {
    await cleanup.run(request);
  }
});
