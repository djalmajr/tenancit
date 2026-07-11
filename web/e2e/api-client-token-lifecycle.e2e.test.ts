import { expect, test } from "@playwright/test";
import { type APIClientRecord, CatalogCleanup, findAPIClientByName } from "./fixtures/catalog";
import { uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";
import { loginViaUI, navigateFromSidebar } from "./support/ui";

test.use({ screenshot: "off", trace: "off" });

test("API token is shown once, masked, revoked terminally, and deleted", { tag: "@pr-critical" }, async ({ context, page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();
  const name = uniqueID("token-lifecycle");
  let token = "";
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);

  try {
    await flowStep("api-client-token-lifecycle", 1, "abre chaves e orientação de consumo", async () => {
      await loginViaUI(page);
      await navigateFromSidebar(page, "Chaves de API");
      await page.getByRole("button", { name: "Ajuda" }).click();
      await expect(page.getByText("Acesso a segredos em claro", { exact: true })).toBeVisible();
      await expect(page.getByText(/\/v1\/resolve/).first()).toBeVisible();
      await expect(page.getByRole("table")).toBeVisible();
    });
    await flowStep("api-client-token-lifecycle", 2, "copia snippet com feedback", async () => {
      const copy = page.getByRole("button", { name: "Copiar snippet" }).first();
      await copy.click();
      await expect(page.getByText("Snippet copiado", { exact: true }).last()).toBeVisible();
      await page.keyboard.press("Escape");
    });
    await flowStep("api-client-token-lifecycle", 3, "abre a criação de chave", async () => {
      await page.getByRole("button", { name: "Nova chave" }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Nova chave de API" }) });
      await expect(dialog.getByPlaceholder("billing-service")).toBeVisible();
    });
    await flowStep("api-client-token-lifecycle", 4, "gera o token one-shot", async () => {
      const dialog = page.getByRole("dialog");
      await dialog.getByPlaceholder("billing-service").fill(name);
      const [response] = await Promise.all([
        page.waitForResponse((candidate) =>
          candidate.request().method() === "POST"
          && new URL(candidate.url()).pathname === "/v1/admin/api-clients"),
        dialog.getByRole("button", { name: "Gerar token" }).click(),
      ]);
      expect(response.ok(), "API client creation response").toBe(true);
      const created = await response.json() as { client: APIClientRecord; token: string };
      cleanup.trackAPIClient(created.client.id);
      const tokenDialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Token gerado" }) });
      expect(
        (await tokenDialog.getByRole("textbox", { name: "Token" }).inputValue()) === created.token,
      ).toBe(true);
      token = created.token;
    });
    await flowStep("api-client-token-lifecycle", 5, "copia o token explícito", async () => {
      const dialog = page.getByRole("dialog");
      await dialog.getByRole("button", { name: "Copiar" }).click();
      await expect(dialog.getByRole("button", { name: "Copiado" })).toBeVisible();
      await expect.poll(() => page.evaluate(
        async (expected) => (await navigator.clipboard.readText()) === expected,
        token,
      )).toBe(true);
    });
    await flowStep("api-client-token-lifecycle", 6, "conclui com preview mascarado", async () => {
      await page.getByRole("dialog").getByRole("button", { name: "Concluir" }).click();
      const client = await findAPIClientByName(request, name);
      const row = page.getByRole("row").filter({ hasText: name });
      await expect(row).toBeVisible();
      expect((await row.textContent())?.includes(token) ?? false).toBe(false);
      await expect(row).toContainText(client.key_preview ?? "tnc_••••••••");
    });
    await flowStep("api-client-token-lifecycle", 7, "revoga terminalmente", async () => {
      const row = page.getByRole("row").filter({ hasText: name });
      await row.getByRole("button", { name: "Revogar" }).click();
      await expect(row).toContainText("revogado");
      await expect(row.getByRole("button", { name: "Remover" })).toBeVisible();
      await expect(page.getByText("Chave revogada.", { exact: true })).toBeVisible();
    });
    await flowStep("api-client-token-lifecycle", 8, "remove a chave revogada", async () => {
      const row = page.getByRole("row").filter({ hasText: name });
      await row.getByRole("button", { name: "Remover" }).click();
      const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Remover chave de API" }) });
      await dialog.getByRole("button", { name: "Remover" }).click();
      await expect(row).toHaveCount(0);
      await expect(page.getByText("Chave removida.", { exact: true })).toBeVisible();
    });
  } finally {
    await cleanup.run(request);
  }
});
