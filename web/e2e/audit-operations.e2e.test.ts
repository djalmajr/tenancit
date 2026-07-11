import { expect, test } from "@playwright/test";
import { authenticate, uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";

test("auditor investigates, holds, and exports activity", { tag: "@full" }, async ({ page }) => {
  test.slow();
  await flowStep("audit-operations", 1, "abre activity e confirma saúde das partições", async () => {
    await authenticate(page);
    await page.goto("/audit-events");
    await expect(page.getByRole("heading", { name: "Auditoria administrativa" })).toBeVisible();
    await expect(page.getByText("Partições até", { exact: true })).toBeVisible();
    await expect(page.getByText("Eventos no legado", { exact: true })).toBeVisible();
  });
  await flowStep("audit-operations", 2, "filtra a activity no servidor", async () => {
    await page.getByRole("textbox", { name: "Ação", exact: true }).fill("audit.events_read");
    const responsePromise = page.waitForResponse((response) => response.url().includes("/v1/admin/audit-events?") && response.url().includes("action=audit.events_read"));
    await page.getByRole("button", { name: "Aplicar filtros" }).click();
    expect((await responsePromise).ok()).toBe(true);
    await expect(page.getByText("audit.events_read", { exact: true }).first()).toBeVisible();
  });
  await flowStep("audit-operations", 3, "cria e libera legal hold", async () => {
    const reason = uniqueID("incident-hold");
    await page.getByRole("button", { name: "Gerenciar legal holds" }).click();
    await page.getByRole("textbox", { name: "Motivo e referência" }).fill(reason);
    await page.getByRole("button", { name: "Criar legal hold" }).click();
    await expect(page.getByText(reason, { exact: true })).toBeVisible();
    await page.getByRole("button", { name: "Liberar" }).click();
    await expect(page.getByText("Liberado", { exact: true })).toBeVisible();
    await page.getByRole("button", { name: "Cancelar" }).click();
  });
  await flowStep("audit-operations", 4, "gera e baixa export cifrado one-shot", async () => {
    await page.getByRole("button", { name: "Exportar trilha" }).click();
    await expect(page.getByText(/Exportação: ready/)).toBeVisible();
    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("button", { name: "Baixar uma vez" }).click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/^tenancit-audit.*\.csv$/);
    await expect(page.getByRole("button", { name: "Baixar uma vez" })).toHaveCount(0);
  });
});
