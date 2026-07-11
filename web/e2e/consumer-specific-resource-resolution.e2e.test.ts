import { expect, test } from "@playwright/test";
import { CatalogCleanup } from "./fixtures/catalog";
import { uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";
import {
  addDomainViaUI,
  addFieldViaUI,
  addResourceViaUI,
  createAPIClientViaUI,
  createDefinitionViaUI,
  createTenantViaUI,
  loginViaUI,
  navigateFromSidebar,
} from "./support/ui";

test.use({ screenshot: "off", trace: "off" });

test("specific resource endpoint isolates definitions and honors revocation", { tag: "@pr-critical" }, async ({ context, page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();
  const keyOne = uniqueID("specific-one");
  const keyTwo = uniqueID("specific-two");
  const nameOne = `Specific ${keyOne}`;
  const nameTwo = `Specific ${keyTwo}`;
  const slug = uniqueID("specific-tenant");
  const tenantName = `Specific ${slug}`;
  const hostname = `${uniqueID("specific")}.e2e.local`;
  const clientName = uniqueID("specific-client");
  let token = "";
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);

  try {
    await flowStep("consumer-specific-resource-resolution", 1, "autentica no console", async () => {
      await loginViaUI(page);
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    });
    await flowStep("consumer-specific-resource-resolution", 2, "cria duas definições ativas", async () => {
      await createDefinitionViaUI(page, cleanup, nameOne, keyOne);
      await addFieldViaUI(page, { key: "host", label: "Host", required: true });
      await createDefinitionViaUI(page, cleanup, nameTwo, keyTwo);
      await addFieldViaUI(page, { key: "endpoint", label: "Endpoint", required: true });
      await expect(page.getByRole("cell", { name: "endpoint", exact: true })).toBeVisible();
    });
    await flowStep("consumer-specific-resource-resolution", 3, "cria tenant único", async () => {
      await createTenantViaUI(page, cleanup, tenantName, slug);
      await expect(page.getByRole("heading", { name: tenantName })).toBeVisible();
    });
    await flowStep("consumer-specific-resource-resolution", 4, "associa hostname único", async () => {
      await addDomainViaUI(page, hostname);
      await expect(page.getByRole("cell", { name: hostname })).toBeVisible();
    });
    await flowStep("consumer-specific-resource-resolution", 5, "provisiona os dois recursos", async () => {
      await addResourceViaUI(page, nameOne, { host: `db.${hostname}` });
      await addResourceViaUI(page, nameTwo, { endpoint: `https://${hostname}` });
      await expect(page.getByText(nameOne, { exact: true })).toBeVisible();
      await expect(page.getByText(nameTwo, { exact: true })).toBeVisible();
    });
    await flowStep("consumer-specific-resource-resolution", 6, "gera uma chave de consumo", async () => {
      await navigateFromSidebar(page, "Chaves de API");
      const created = await createAPIClientViaUI(page, cleanup, clientName);
      token = created.token;
      await created.dialog.getByRole("button", { name: "Copiar" }).click();
      await expect(created.dialog.getByRole("button", { name: "Copiado" })).toBeVisible();
      await expect.poll(() => page.evaluate(
        async (expected) => (await navigator.clipboard.readText()) === expected,
        token,
      )).toBe(true);
      await created.dialog.getByRole("button", { name: "Concluir" }).click();
      await expect(page.getByRole("row").filter({ hasText: clientName })).toContainText("ativo");
    });
    await flowStep("consumer-specific-resource-resolution", 7, "retorna somente a definição solicitada", async () => {
      const response = await request.get(
        `/v1/resolve/${encodeURIComponent(hostname)}/resources/${encodeURIComponent(keyOne)}`,
        { headers: { Authorization: `Bearer ${token}` } },
      );
      expect(response.status()).toBe(200);
      expect(response.headers()["cache-control"]).toBe("private, no-store");
      const body = await response.text();
      expect(body).toContain(keyOne);
      expect(body).not.toContain(keyTwo);
    });
    await flowStep("consumer-specific-resource-resolution", 8, "retorna 404 para definição ausente", async () => {
      const response = await request.get(
        `/v1/resolve/${encodeURIComponent(hostname)}/resources/${uniqueID("missing")}`,
        { headers: { Authorization: `Bearer ${token}` } },
      );
      expect(response.status()).toBe(404);
      expect(await response.json()).toMatchObject({ error: "resource not found" });
    });
    await flowStep("consumer-specific-resource-resolution", 9, "revoga na UI e bloqueia o token", async () => {
      const row = page.getByRole("row").filter({ hasText: clientName });
      await row.getByRole("button", { name: "Revogar" }).click();
      await expect(row).toContainText("revogado");
      const response = await request.get(
        `/v1/resolve/${encodeURIComponent(hostname)}/resources/${encodeURIComponent(keyOne)}`,
        { headers: { Authorization: `Bearer ${token}` } },
      );
      expect(response.status()).toBe(401);
    });
  } finally {
    await cleanup.run(request);
  }
});
