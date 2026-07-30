import { expect, test, type APIResponse } from "@playwright/test";
import { CatalogCleanup } from "./fixtures/catalog";
import { uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";
import { SECRET_MASK } from "../src/lib/secret-display";
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

test("admin configuration becomes a cache-safe Consumer API contract", { tag: "@pr-critical" }, async ({ context, page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();
  const key = uniqueID("golden-definition");
  const definitionName = `Golden ${key}`;
  const slug = uniqueID("golden-tenant");
  const tenantName = `Golden ${slug}`;
  const hostname = `${uniqueID("golden")}.e2e.local`;
  const clientName = uniqueID("golden-client");
  const secret = `secret-${slug}`;
  let token = "";
  let resolveResponse: APIResponse | undefined;
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);

  try {
    await flowStep("admin-to-consumer-golden-path", 1, "autentica no console", async () => {
      await loginViaUI(page);
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    });
    await flowStep("admin-to-consumer-golden-path", 2, "cria definição consumível pela UI", async () => {
      await createDefinitionViaUI(page, cleanup, definitionName, key);
      await addFieldViaUI(page, { key: "host", label: "Host", required: true });
      await addFieldViaUI(page, { key: "password", label: "Password", required: true, secret: true });
      const passwordRow = page.getByRole("row").filter({ has: page.getByRole("cell", { name: "password" }) });
      await expect(passwordRow.getByRole("cell").nth(3).locator("svg")).toHaveCount(1);
      await expect(passwordRow.getByRole("cell").nth(4).locator("svg")).toHaveCount(1);
    });
    await flowStep("admin-to-consumer-golden-path", 3, "cria tenant único pela UI", async () => {
      await createTenantViaUI(page, cleanup, tenantName, slug);
      await expect(page.getByRole("heading", { name: tenantName })).toBeVisible();
    });
    await flowStep("admin-to-consumer-golden-path", 4, "adiciona hostname pela UI", async () => {
      await addDomainViaUI(page, hostname);
      await expect(page.getByRole("cell", { name: hostname })).toBeVisible();
    });
    await flowStep("admin-to-consumer-golden-path", 5, "provisiona recurso com segredo mascarado", async () => {
      await addResourceViaUI(page, definitionName, { host: `db.${hostname}`, password: secret });
      const resourceRow = page.getByRole("row").filter({ hasText: definitionName });
      await expect(resourceRow).toContainText("ativo");
      await resourceRow.click();
      const resourceDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: definitionName, exact: true }),
      });
      await expect(resourceDialog.getByText(SECRET_MASK, { exact: true })).toBeVisible();
      expect(await page.locator("body").evaluate(
        (body, sensitive) => body.textContent?.includes(sensitive) ?? false,
        secret,
      )).toBe(false);
      await resourceDialog.getByRole("button", { name: "Fechar", exact: true }).click();
      await page.getByRole("tab", { name: "Visão geral", exact: true }).click();
      const resources = page.getByText("Recursos ativos/total", { exact: true }).locator("../..");
      await expect(resources.getByText("1/1", { exact: true })).toBeVisible();
    });
    await flowStep("admin-to-consumer-golden-path", 6, "confere snippets e gera token one-shot", async () => {
      await navigateFromSidebar(page, "Chaves de API");
      await page.getByRole("button", { name: "Ajuda" }).click();
      const helpDialog = page.getByRole("dialog").filter({
        has: page.getByRole("heading", { name: "Como consumir chaves de API", exact: true }),
      });
      await expect(helpDialog.getByText("1. Identificar na borda (sem segredos)", { exact: true })).toBeVisible();
      await expect(helpDialog.getByText("2. Resolver no app por tenantId", { exact: true })).toBeVisible();
      await page.keyboard.press("Escape");
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
    await flowStep("admin-to-consumer-golden-path", 7, "identifica sem retornar segredos", async () => {
      const response = await request.get(`/v1/identify?hostname=${encodeURIComponent(hostname)}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      expect(response.status()).toBe(200);
      expect(await response.json()).toEqual({ tenantSlug: slug });
      expect((await response.text()).includes(secret)).toBe(false);
    });
    await flowStep("admin-to-consumer-golden-path", 8, "resolve por tenantId sem cache de corpo", async () => {
      resolveResponse = await request.get(`/v1/resolve?tenantId=${encodeURIComponent(slug)}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      expect(resolveResponse.status()).toBe(200);
      expect(resolveResponse.headers()["cache-control"]).toBe("private, no-store");
      expect(resolveResponse.headers().etag).toBeTruthy();
      const body = await resolveResponse.text();
      expect(body).toContain(key);
      expect(body.includes(secret)).toBe(true);
    });
    await flowStep("admin-to-consumer-golden-path", 9, "revalida com ETag sem corpo", async () => {
      const etag = resolveResponse?.headers().etag ?? "";
      expect(etag).not.toBe("");
      const response = await request.get(`/v1/resolve?tenantId=${encodeURIComponent(slug)}`, {
        headers: { Authorization: `Bearer ${token}`, "If-None-Match": etag },
      });
      expect(response.status()).toBe(304);
      expect(await response.body()).toHaveLength(0);
    });
    await flowStep("admin-to-consumer-golden-path", 10, "rejeita identify e resolve sem token", async () => {
      const identify = await request.get(`/v1/identify?hostname=${encodeURIComponent(hostname)}`);
      const resolve = await request.get(`/v1/resolve?tenantId=${encodeURIComponent(slug)}`);
      expect(identify.status()).toBe(401);
      expect(resolve.status()).toBe(401);
    });
    await flowStep("admin-to-consumer-golden-path", 11, "retorna 404 para hostname desconhecido", async () => {
      const response = await request.get(`/v1/identify?hostname=${uniqueID("unknown")}.e2e.local`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      expect(response.status()).toBe(404);
      expect(await response.json()).toMatchObject({ error: "tenant not found" });
    });
  } finally {
    await cleanup.run(request);
  }
});
