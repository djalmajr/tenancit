import { expect, test } from "@playwright/test";
import { adminRequest, authenticate, uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";

test("auditor investigates, holds, and exports activity", { tag: "@full" }, async ({ page, request }) => {
  test.slow();
  await flowStep("audit-operations", 1, "abre activity e confirma saúde das partições", async () => {
    await authenticate(page);
    await page.goto("/audit-events");
    await expect(page.getByRole("heading", { name: "Auditoria administrativa" })).toBeVisible();
    await expect(page.getByText("Partições até", { exact: true })).toBeVisible();
    await expect(page.getByText("Eventos no legado", { exact: true })).toBeVisible();
  });
  await flowStep("audit-operations", 2, "filtra a activity no servidor", async () => {
    const filtered = await adminRequest<{ events: Array<{ action: string }> }>(
      request,
      "get",
      "/v1/admin/audit-events?action=audit.events_read&limit=200",
    );
    // Mutation captured: dropping the server-side action predicate returns mixed lifecycle events.
    expect(filtered.events.length).toBeGreaterThan(0);
    expect(new Set(filtered.events.map((event) => event.action))).toEqual(new Set(["audit.events_read"]));

    await page.reload();
    await page.getByRole("tab", { name: "Eventos", exact: true }).click();
    await page.getByPlaceholder("Ação, Alvo, Request ID").fill("audit.events_read");
    await expect(page.getByRole("row").filter({ hasText: "audit.events_read" }).first()).toBeVisible();
  });
  await flowStep("audit-operations", 3, "cria e libera legal hold", async () => {
    const reason = uniqueID("incident-hold");
    await page.getByRole("tab", { name: "Visão geral", exact: true }).click();
    await page.getByRole("button", { name: "Gerenciar legal holds" }).click();
    await page.getByRole("textbox", { name: "Motivo e referência" }).fill(reason);
    await page.getByRole("button", { name: "Criar legal hold" }).click();
    await expect(page.getByText(reason, { exact: true })).toBeVisible();
    await page.getByRole("button", { name: "Liberar" }).click();
    await expect(page.getByText("Liberado", { exact: true })).toBeVisible();
    await page.getByRole("button", { name: "Cancelar" }).click();
  });
  await flowStep("audit-operations", 4, "gera e baixa export cifrado one-shot", async () => {
    await page.getByRole("tab", { name: "Eventos", exact: true }).click();
    await page.getByRole("button", { name: "Exportar trilha" }).click();
    await expect(page.getByText(/Exportação: ready/)).toBeVisible();
    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("button", { name: "Baixar uma vez" }).click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/^tenancit-audit.*\.csv$/);
    await expect(page.getByRole("button", { name: "Baixar uma vez" })).toHaveCount(0);
  });
});
